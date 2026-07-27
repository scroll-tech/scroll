package database

import (
	"context"
	"database/sql/driver"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// rdsIAMConnector is a driver.Connector that mints a fresh AWS RDS IAM auth
// token for every new connection (tokens expire after 15 minutes; already-open
// connections keep working since the token is only checked at connect time).
type rdsIAMConnector struct {
	base     *pgx.ConnConfig         // parsed DSN; Password is set to the token per-connect
	endpoint string                  // host:port the token is signed for
	region   string                  // AWS region the token is signed for
	creds    aws.CredentialsProvider // refreshes credentials internally
	metrics  *rdsIAMMetrics
}

// NewRDSIAMConnector returns a connector that injects a fresh RDS IAM token as
// the password on each new connection. region falls back to the default AWS
// config chain (e.g. AWS_REGION) when empty.
//
// IAM auth requires TLS, so a DSN with sslmode=disable is rejected rather than
// silently downgraded; sslmode=verify-full (+ sslrootcert) is recommended.
func NewRDSIAMConnector(ctx context.Context, dsn, region string) (driver.Connector, error) {
	base, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("rds iam: parse dsn: %w", err)
	}
	if base.User == "" {
		return nil, fmt.Errorf("rds iam: dsn must specify a database user")
	}

	var opts []func(*awsconfig.LoadOptions) error
	if region != "" {
		opts = append(opts, awsconfig.WithRegion(region))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("rds iam: load aws config: %w", err)
	}
	if region == "" {
		region = awsCfg.Region
	}
	if region == "" {
		return nil, fmt.Errorf("rds iam: aws region is not set (configure awsRegion or AWS_REGION)")
	}

	// Fail closed on any path that could send the token (which is the password)
	// in cleartext or to the wrong host. The token is scoped to a single
	// host:port, so reject:
	//   - TLS-less DSNs: sslmode=disable/allow leave the primary TLSConfig nil;
	//   - plaintext fallbacks: sslmode=prefer (the pgx default) keeps a non-TLS
	//     entry in Fallbacks that pgx retries on if SSL negotiation fails;
	//   - multi-host DSNs: a fallback host the single token is not signed for.
	if base.TLSConfig == nil {
		return nil, fmt.Errorf("rds iam: TLS is required, set sslmode=require or higher (verify-full recommended) in the dsn")
	}
	for _, fb := range base.Fallbacks {
		if fb.TLSConfig == nil {
			return nil, fmt.Errorf("rds iam: dsn allows a plaintext fallback (sslmode=prefer); set sslmode=require or higher")
		}
		if fb.Host != base.Host || fb.Port != base.Port {
			return nil, fmt.Errorf("rds iam: multi-host dsn is not supported; the IAM token is scoped to a single host:port")
		}
	}

	return &rdsIAMConnector{
		base:     base,
		endpoint: net.JoinHostPort(base.Host, strconv.Itoa(int(base.Port))),
		region:   region,
		creds:    awsCfg.Credentials,
		metrics:  initRDSIAMMetrics(),
	}, nil
}

// Connect generates a fresh IAM auth token and opens a new connection with it.
func (c *rdsIAMConnector) Connect(ctx context.Context) (driver.Conn, error) {
	start := time.Now()
	token, err := auth.BuildAuthToken(ctx, c.endpoint, c.region, c.base.User, c.creds)
	c.metrics.tokenDuration.Observe(time.Since(start).Seconds())
	if err != nil {
		c.metrics.tokenFailureTotal.Inc()
		return nil, fmt.Errorf("rds iam: build auth token: %w", err)
	}
	c.metrics.tokenTotal.Inc()

	cfg := c.base.Copy()
	cfg.Password = token
	return stdlib.GetConnector(*cfg).Connect(ctx)
}

// Driver returns the underlying pgx stdlib driver.
func (c *rdsIAMConnector) Driver() driver.Driver { return stdlib.GetDefaultDriver() }

package database

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRDSIAMConnector(t *testing.T) {
	ctx := context.Background()

	t.Run("parses endpoint, user and region", func(t *testing.T) {
		c, err := NewRDSIAMConnector(ctx,
			"postgres://svc_user@mydb.abc123.us-east-1.rds.amazonaws.com:5432/scroll?sslmode=verify-full",
			"us-east-1")
		require.NoError(t, err)

		conn, ok := c.(*rdsIAMConnector)
		require.True(t, ok)
		assert.Equal(t, "mydb.abc123.us-east-1.rds.amazonaws.com:5432", conn.endpoint)
		assert.Equal(t, "svc_user", conn.base.User)
		assert.Equal(t, "us-east-1", conn.region)
		// TLS requested in the DSN must be preserved.
		require.NotNil(t, conn.base.TLSConfig)
	})

	t.Run("defaults port to 5432", func(t *testing.T) {
		c, err := NewRDSIAMConnector(ctx,
			"postgres://svc_user@mydb.example.rds.amazonaws.com/scroll?sslmode=require",
			"eu-west-1")
		require.NoError(t, err)
		assert.Equal(t, "mydb.example.rds.amazonaws.com:5432", c.(*rdsIAMConnector).endpoint)
	})

	t.Run("rejects sslmode=disable", func(t *testing.T) {
		// IAM auth requires TLS; a plaintext DSN must fail closed rather than
		// silently sending the token in cleartext.
		_, err := NewRDSIAMConnector(ctx,
			"postgres://svc_user@mydb.example.rds.amazonaws.com:5432/scroll?sslmode=disable",
			"us-east-1")
		assert.ErrorContains(t, err, "TLS")
	})

	t.Run("rejects sslmode=prefer (plaintext fallback)", func(t *testing.T) {
		// prefer keeps a non-TLS entry in Fallbacks that pgx would retry on,
		// which would leak the token in cleartext.
		_, err := NewRDSIAMConnector(ctx,
			"postgres://svc_user@mydb.example.rds.amazonaws.com:5432/scroll?sslmode=prefer",
			"us-east-1")
		assert.ErrorContains(t, err, "plaintext")
	})

	t.Run("rejects sslmode=allow (plaintext primary)", func(t *testing.T) {
		_, err := NewRDSIAMConnector(ctx,
			"postgres://svc_user@mydb.example.rds.amazonaws.com:5432/scroll?sslmode=allow",
			"us-east-1")
		assert.Error(t, err)
	})

	t.Run("rejects multi-host dsn", func(t *testing.T) {
		// The token is signed for a single host:port, so failover would fail auth.
		_, err := NewRDSIAMConnector(ctx,
			"host=host1.rds.amazonaws.com,host2.rds.amazonaws.com port=5432 user=svc_user dbname=scroll sslmode=require",
			"us-east-1")
		assert.ErrorContains(t, err, "multi-host")
	})

	t.Run("region falls back to explicit empty error when unresolved", func(t *testing.T) {
		// With no region argument, no AWS_REGION in the environment, and no
		// shared config file, region resolution must fail rather than silently
		// signing with an empty region.
		t.Setenv("AWS_REGION", "")
		t.Setenv("AWS_DEFAULT_REGION", "")
		// Isolate from any ~/.aws/config on the dev/CI machine, which would
		// otherwise resolve a default region and make this test nondeterministic.
		t.Setenv("AWS_CONFIG_FILE", filepath.Join(t.TempDir(), "no-config"))
		t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(t.TempDir(), "no-creds"))
		_, err := NewRDSIAMConnector(ctx,
			"postgres://svc_user@mydb.example.rds.amazonaws.com:5432/scroll?sslmode=require",
			"")
		assert.ErrorContains(t, err, "region")
	})

	t.Run("rejects invalid dsn", func(t *testing.T) {
		_, err := NewRDSIAMConnector(ctx, "::not a dsn::", "us-east-1")
		assert.Error(t, err)
	})
}

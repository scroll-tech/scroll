# DATABASE CLIENT

This repo contains the Scroll database client.

Database client will provide init, show version, rollback, check status services

## Build

``` bash
make db_cli
```

## Usage
``` bash
# Migrate
db_cli migrate
# Reset
db_cli reset
# Status
db_cli status
# Version
db_cli version
# RollBack
db_cli rollback
```

## AWS RDS IAM authentication

Instead of a static password in the DSN, services can authenticate to AWS
RDS/Aurora PostgreSQL using short-lived IAM auth tokens. This removes the need
to rotate database passwords: access is granted via IAM and tokens are
regenerated automatically (they expire every 15 minutes) for each new
connection.

Enable it per service in the DB config block:

```json
{
  "dsn": "postgres://svc_user@mydb.abc123.us-east-1.rds.amazonaws.com:5432/scroll?sslmode=require",
  "driver_name": "postgres",
  "maxOpenNum": 200,
  "maxIdleNum": 20,
  "useIAMAuth": true,
  "awsRegion": "us-east-1"
}
```

Notes:

- Omit the password from the `dsn`; it is supplied by the generated IAM token.
- `awsRegion` is optional — when empty it is resolved from the default AWS
  config chain (e.g. the `AWS_REGION` environment variable).
- IAM auth requires TLS, so the DSN must set `sslmode=require` or higher.
  `sslmode=disable`/`allow`/`prefer` (including an unset `sslmode`, which
  defaults to `prefer`) are rejected at startup rather than silently sending the
  token over a connection that may fall back to plaintext. `require` encrypts
  but does not verify the server certificate; for that, use `sslmode=verify-full`
  with `sslrootcert` pointing at the
  [RDS CA bundle](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.SSL.html).
- The database role must be granted `rds_iam`, and the service's IAM role needs
  `rds-db:connect` on the `dbuser` resource. Leaving `useIAMAuth` unset
  preserves the previous password-based behavior.

## Test

```bash
make test
```

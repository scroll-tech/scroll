package database

// Config db config
type Config struct {
	// data source name
	DSN        string `json:"dsn"`
	DriverName string `json:"driver_name"`

	MaxOpenNum int `json:"maxOpenNum"`
	MaxIdleNum int `json:"maxIdleNum"`

	// UseIAMAuth authenticates to AWS RDS/Aurora with short-lived IAM tokens
	// instead of a DSN password. The DSN should omit the password and enable TLS.
	UseIAMAuth bool `json:"useIAMAuth"`
	// AWSRegion signs the IAM tokens. Optional; falls back to the default AWS
	// config chain (e.g. AWS_REGION) when empty.
	AWSRegion string `json:"awsRegion"`
}

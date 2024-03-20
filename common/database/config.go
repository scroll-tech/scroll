package database

// Config db config
type Config struct {
	// data source name
	DSN        string `json:"dsn"`
	DriverName string `json:"driver_name"`

	MaxOpenNum int `json:"maxOpenNum"`
	MaxIdleNum int `json:"maxIdleNum"`
	MaxLifetime int `json:"maxLifetime,omitempty"`
	MaxIdleTime int `json:"maxIdleTime,omitempty"`
}

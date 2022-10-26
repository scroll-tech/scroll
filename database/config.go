package database

// DBConfig db config
type DBConfig struct {
	// data source name
	DSN        string `json:"dsn"`
	DriverName string `json:"driver_name"`

	MaxOpenNum int `json:"maxOpenNum" default:"200"`
	MaxIdleNUm int `json:"maxIdleNum" default:"20"`
}

package docker

import (
	"database/sql"
	"testing"
	"time"

	"scroll-tech/database"
)

var (
	l1StartPort = 10000
	l2StartPort = 20000
	dbStartPort = 30000
)

// AppAPI app interface.
type AppAPI interface {
	IsRunning() bool
	WaitResult(t *testing.T, timeout time.Duration, keyword string) bool
	RunApp(waitResult func() bool)
	WaitExit()
	ExpectWithTimeout(t *testing.T, parallel bool, timeout time.Duration, keyword string)
}

// App is collection struct of runtime docker images
type App struct {
	L1gethImg GethImgInstance
	L2gethImg GethImgInstance
	DBImg     ImgInstance

	dbClient     *sql.DB
	DBConfig     *database.DBConfig
	DBConfigFile string

	// common time stamp.
	Timestamp int
}

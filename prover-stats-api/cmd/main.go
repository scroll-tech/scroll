package main

import (
	"scroll-tech/prover-stats-api/cmd/app"
	_ "scroll-tech/prover-stats-api/docs"
)

// @title           Scroll Core Stats API
// @version         1.0
// @description     This is an API server for Provers.

// @contact.name   Core Stats API Support
// @contact.email  Be Pending

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8990
// @BasePath  /api/v1

// @securityDefinitions.basic  BasicAuth
func main() {
	app.Run()
}

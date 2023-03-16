package integration

import (
	_ "scroll-tech/bridge/cmd/app"
	app2 "scroll-tech/bridge/cmd/app"
	"scroll-tech/common/docker"
	_ "scroll-tech/coordinator/cmd/app"
	app3 "scroll-tech/coordinator/cmd/app"
	_ "scroll-tech/database/cmd/app"
	_ "scroll-tech/roller/cmd/app"
	app4 "scroll-tech/roller/cmd/app"
	"testing"
)

var (
	base        *docker.DockerApp
	bridge      *app2.BridgeApp
	coordinator *app3.CoordinatorApp
	rollers     app4.RollerApps
)

func TestMain(m *testing.M) {
	base = docker.NewDockerApp()
	bridge = app2.NewBridgeApp(base, "../../bridge/config.json")
	coordinator = app3.NewCoordinatorApp(base, "../../coordinator/config.json")
	rollers = append(rollers, app4.NewRollerApp(base, coordinator.WSEndpoint(), "../../roller/config.json"))

	m.Run()

	base.Free()
	bridge.Free()
	coordinator.Free()
	rollers.Free()
}

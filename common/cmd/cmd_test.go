package cmd_test

import (
	"fmt"
	"testing"
	"time"

	"scroll-tech/common/cmd"
)

func TestCmd(t *testing.T) {
	app := cmd.NewCmd(t, "curTime", "date", "+%Y-%m-%d %H:%M")

	tm := time.Now()
	curTime := fmt.Sprintf("%d-%d-%d %d:%d", tm.Year(), tm.Month(), tm.Day(), tm.Hour(), tm.Minute())
	app.RunCmd(true)
	app.ExpectWithTimeout(false, time.Second, curTime)
}

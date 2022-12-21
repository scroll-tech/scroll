package cmd_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"scroll-tech/common/cmd"
)

func TestCmd(t *testing.T) {
	app := cmd.NewCmd(t, "curTime", "date", "+%Y-%m-%d %H:%M")

	tm := time.Now()
	curTime := fmt.Sprintf("%d-%d-%d %d:%d", tm.Year(), tm.Month(), tm.Day(), tm.Hour(), tm.Minute())

	okCh := make(chan struct{}, 1)
	app.RegistFunc(curTime, func(buf string) {
		if strings.Contains(buf, curTime) {
			select {
			case okCh <- struct{}{}:
			default:
				return
			}
		}
	})
	defer app.UnRegistFunc(curTime)

	// Run cmd.
	app.RunCmd(true)

	// Wait result.
	select {
	case <-okCh:
		return
	case <-time.After(time.Second):
		assert.Fail(t, fmt.Sprintf("didn't get the desired result before timeout, keyword: %s", curTime))
	}
}

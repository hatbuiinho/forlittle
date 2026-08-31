//go:build !windows

package timecontrol

import (
	"context"
	"fmt"
)

func activeInteractiveSessionID() (uint32, error) {
	return 0, fmt.Errorf("interactive sessions are only supported on Windows")
}

func restartAgent(context.Context, string) (uint32, error) {
	return 0, fmt.Errorf("agent restart is only supported on Windows")
}

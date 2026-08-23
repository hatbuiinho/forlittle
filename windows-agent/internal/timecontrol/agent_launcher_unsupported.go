//go:build !windows

package timecontrol

import (
	"context"
	"fmt"
)

func restartAgent(context.Context, string) error {
	return fmt.Errorf("agent restart is only supported on Windows")
}

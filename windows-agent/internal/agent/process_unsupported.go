//go:build !windows

package agent

import (
	"context"
	"errors"
)

func listChromeProcesses(context.Context) ([]ChromeProcess, error) {
	return nil, errors.New("windows process inspection is only supported on Windows")
}

func killProcess(context.Context, int) error {
	return errors.New("windows process termination is only supported on Windows")
}

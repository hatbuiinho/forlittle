//go:build windows

package timecontrol

import (
	"context"
	"fmt"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

// A service runs in Session 0. WTSQueryUserToken plus CreateProcessAsUser starts
// the visible agent on the active user's interactive desktop instead.
func restartAgent(ctx context.Context, agentPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if agentPath == "" {
		return fmt.Errorf("agent path is empty")
	}
	sessionID := windows.WTSGetActiveConsoleSessionId()
	if sessionID == 0xffffffff {
		return fmt.Errorf("no active interactive session")
	}
	var userToken windows.Token
	if err := windows.WTSQueryUserToken(sessionID, &userToken); err != nil {
		return fmt.Errorf("query active user token: %w", err)
	}
	defer userToken.Close()
	var primaryToken windows.Token
	if err := windows.DuplicateTokenEx(userToken, windows.TOKEN_ALL_ACCESS, nil, windows.SecurityIdentification, windows.TokenPrimary, &primaryToken); err != nil {
		return fmt.Errorf("duplicate user token: %w", err)
	}
	defer primaryToken.Close()

	application, err := windows.UTF16PtrFromString(agentPath)
	if err != nil {
		return err
	}
	commandLine, err := windows.UTF16PtrFromString(`"` + agentPath + `"`)
	if err != nil {
		return err
	}
	desktop, err := windows.UTF16PtrFromString(`winsta0\default`)
	if err != nil {
		return err
	}
	workingDirectory, err := windows.UTF16PtrFromString(filepath.Dir(agentPath))
	if err != nil {
		return err
	}
	startup := windows.StartupInfo{Cb: uint32(unsafe.Sizeof(windows.StartupInfo{})), Desktop: desktop}
	var process windows.ProcessInformation
	if err := windows.CreateProcessAsUser(primaryToken, application, commandLine, nil, nil, false, windows.CREATE_NEW_CONSOLE, nil, workingDirectory, &startup, &process); err != nil {
		return fmt.Errorf("start agent in user session: %w", err)
	}
	_ = windows.CloseHandle(process.Process)
	_ = windows.CloseHandle(process.Thread)
	return nil
}

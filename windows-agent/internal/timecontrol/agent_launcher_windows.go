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
func activeInteractiveSessionID() (uint32, error) {
	sessionID := windows.WTSGetActiveConsoleSessionId()
	if sessionID == 0xffffffff {
		return 0, fmt.Errorf("no active interactive session")
	}
	return sessionID, nil
}

func restartAgent(ctx context.Context, agentPath string) (uint32, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if agentPath == "" {
		return 0, fmt.Errorf("agent path is empty")
	}
	sessionID, err := activeInteractiveSessionID()
	if err != nil {
		return 0, err
	}
	var userToken windows.Token
	if err := windows.WTSQueryUserToken(sessionID, &userToken); err != nil {
		return 0, fmt.Errorf("query active user token: %w", err)
	}
	defer userToken.Close()
	var primaryToken windows.Token
	if err := windows.DuplicateTokenEx(userToken, windows.TOKEN_ALL_ACCESS, nil, windows.SecurityIdentification, windows.TokenPrimary, &primaryToken); err != nil {
		return 0, fmt.Errorf("duplicate user token: %w", err)
	}
	defer primaryToken.Close()

	// CreateProcessAsUser does not automatically use the target user's
	// environment. Without this block, LOCALAPPDATA and other profile paths
	// resolve to SYSTEM, which breaks per-user agent data and WPF behavior.
	var environment *uint16
	if err := windows.CreateEnvironmentBlock(&environment, primaryToken, false); err != nil {
		return 0, fmt.Errorf("create user environment: %w", err)
	}
	defer windows.DestroyEnvironmentBlock(environment)

	application, err := windows.UTF16PtrFromString(agentPath)
	if err != nil {
		return 0, err
	}
	commandLine, err := windows.UTF16PtrFromString(`"` + agentPath + `"`)
	if err != nil {
		return 0, err
	}
	desktop, err := windows.UTF16PtrFromString(`winsta0\default`)
	if err != nil {
		return 0, err
	}
	workingDirectory, err := windows.UTF16PtrFromString(filepath.Dir(agentPath))
	if err != nil {
		return 0, err
	}
	startup := windows.StartupInfo{Cb: uint32(unsafe.Sizeof(windows.StartupInfo{})), Desktop: desktop}
	var process windows.ProcessInformation
	flags := uint32(windows.CREATE_NEW_CONSOLE | windows.CREATE_UNICODE_ENVIRONMENT)
	if err := windows.CreateProcessAsUser(primaryToken, application, commandLine, nil, nil, false, flags, environment, workingDirectory, &startup, &process); err != nil {
		return 0, fmt.Errorf("start agent in user session: %w", err)
	}
	_ = windows.CloseHandle(process.Process)
	_ = windows.CloseHandle(process.Thread)
	return sessionID, nil
}

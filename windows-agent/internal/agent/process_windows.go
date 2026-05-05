//go:build windows

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

type cimChromeProcess struct {
	ProcessID       any    `json:"ProcessId"`
	ParentProcessID any    `json:"ParentProcessId"`
	CommandLine     string `json:"CommandLine"`
}

func listChromeProcesses(ctx context.Context) ([]ChromeProcess, error) {
	script := `Get-CimInstance Win32_Process -Filter "Name = 'chrome.exe'" | Select-Object ProcessId,ParentProcessId,CommandLine | ConvertTo-Json -Compress`
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("query chrome processes: %w", err)
	}

	if len(output) == 0 {
		return nil, nil
	}

	var processes []cimChromeProcess
	if output[0] == '[' {
		if err := json.Unmarshal(output, &processes); err != nil {
			return nil, fmt.Errorf("parse chrome process list: %w", err)
		}
	} else {
		var process cimChromeProcess
		if err := json.Unmarshal(output, &process); err != nil {
			return nil, fmt.Errorf("parse chrome process: %w", err)
		}
		processes = append(processes, process)
	}

	result := make([]ChromeProcess, 0, len(processes))
	for _, process := range processes {
		pid, ok := parseProcessID(process.ProcessID)
		if !ok || process.CommandLine == "" {
			continue
		}

		parentPID, _ := parseProcessID(process.ParentProcessID)
		result = append(result, ChromeProcess{
			PID:         pid,
			ParentPID:   parentPID,
			CommandLine: process.CommandLine,
		})
	}

	return result, nil
}

func killProcess(ctx context.Context, pid int) error {
	cmd := exec.CommandContext(ctx, "taskkill.exe", "/PID", strconv.Itoa(pid), "/F", "/T")
	return cmd.Run()
}

func parseProcessID(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		return int(typed), typed > 0
	case string:
		parsed, err := strconv.Atoi(typed)
		return parsed, err == nil && parsed > 0
	default:
		return 0, false
	}
}

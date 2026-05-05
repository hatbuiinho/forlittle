package agent

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"forlittle/windows-agent/internal/config"
)

type Runner struct {
	cfg    config.Config
	logger *log.Logger
}

func New(cfg config.Config, logger *log.Logger) Runner {
	return Runner{cfg: cfg, logger: logger}
}

func (r Runner) Run(ctx context.Context) error {
	r.logger.Printf("starting with profile=%q extension=%q", r.cfg.ProfilePath, r.cfg.ExtensionPath)

	if err := os.MkdirAll(r.cfg.ProfilePath, 0o755); err != nil {
		return fmt.Errorf("create profile path: %w", err)
	}

	ticker := time.NewTicker(r.cfg.ScanInterval())
	defer ticker.Stop()

	if err := r.enforce(ctx); err != nil {
		r.logger.Printf("initial enforce failed: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			r.logger.Println("stopping")
			return nil
		case <-ticker.C:
			if err := r.enforce(ctx); err != nil {
				r.logger.Printf("enforce failed: %v", err)
			}
		}
	}
}

func (r Runner) enforce(ctx context.Context) error {
	processes, err := listChromeProcesses(ctx)
	if err != nil {
		return err
	}

	chromePIDs := make(map[int]struct{}, len(processes))
	for _, process := range processes {
		chromePIDs[process.PID] = struct{}{}
	}

	managedCount := 0
	for _, process := range processes {
		if _, parentIsChrome := chromePIDs[process.ParentPID]; parentIsChrome {
			continue
		}

		if r.isManagedChrome(process.CommandLine) {
			managedCount++
			continue
		}

		if r.cfg.KillUnmanagedChrome {
			r.logger.Printf("killing unmanaged chrome root pid=%d", process.PID)
			if err := killProcess(ctx, process.PID); err != nil {
				r.logger.Printf("kill unmanaged chrome root pid=%d failed: %v", process.PID, err)
			}
		}
	}

	if managedCount == 0 {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(r.cfg.RelaunchDelay()):
		}

		return r.launchChrome(ctx)
	}

	return nil
}

func (r Runner) launchChrome(ctx context.Context) error {
	args := []string{
		"--user-data-dir=" + r.cfg.ProfilePath,
		"--disable-extensions-except=" + r.cfg.ExtensionPath,
		"--load-extension=" + r.cfg.ExtensionPath,
	}
	args = append(args, r.cfg.ChromeArgs...)

	r.logger.Printf("launching chrome with args=%q", args)
	cmd := exec.CommandContext(ctx, r.cfg.ChromePath, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("launch chrome: %w", err)
	}

	return nil
}

func (r Runner) isManagedChrome(commandLine string) bool {
	normalized := normalizeCommandLine(commandLine)
	return strings.Contains(normalized, normalizeCommandLine("--user-data-dir="+r.cfg.ProfilePath)) &&
		strings.Contains(normalized, normalizeCommandLine("--disable-extensions-except="+r.cfg.ExtensionPath)) &&
		strings.Contains(normalized, normalizeCommandLine("--load-extension="+r.cfg.ExtensionPath))
}

func normalizeCommandLine(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, `"`, "")
	value = strings.ReplaceAll(value, `'`, "")
	value = strings.ReplaceAll(value, `\\`, `\`)
	return value
}

//go:build windows

package main

import (
	"context"
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"forlittle/windows-agent/internal/config"
	"forlittle/windows-agent/internal/ipc"
	"forlittle/windows-agent/internal/timecontrol"

	"golang.org/x/sys/windows/svc"
)

const serviceName = "ForLittleTimeControl"

func main() {
	configPath := flag.String("config", `C:\ProgramData\ForLittle\TimeControl\config.json`, "path to Time Control config")
	console := flag.Bool("console", false, "run interactively for diagnostics")
	flag.Parse()
	logger, closeLog := newLogger(*configPath, *console)
	defer closeLog()
	if *console {
		runProgram(context.Background(), *configPath, logger)
		return
	}
	interactive, err := svc.IsAnInteractiveSession()
	if err != nil {
		logger.Fatal(err)
	}
	if interactive {
		logger.Printf("interactive session detected; use -console for diagnostics or start through SCM")
		return
	}
	if err := svc.Run(serviceName, serviceProgram{configPath: *configPath, logger: logger}); err != nil {
		logger.Fatal(err)
	}
}

func newLogger(configPath string, console bool) (*log.Logger, func()) {
	writer := io.Writer(os.Stdout)
	dataDir := filepath.Dir(configPath)
	if cfg, err := config.LoadTimeControl(configPath); err == nil && cfg.DataDir != "" {
		dataDir = cfg.DataDir
	}
	if err := os.MkdirAll(dataDir, 0o700); err == nil {
		if file, err := os.OpenFile(filepath.Join(dataDir, "service.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600); err == nil {
			if console {
				writer = io.MultiWriter(os.Stdout, file)
			} else {
				writer = file
			}
			return log.New(writer, "forlittle-time-control: ", log.LstdFlags|log.LUTC), func() { _ = file.Close() }
		}
	}
	return log.New(writer, "forlittle-time-control: ", log.LstdFlags|log.LUTC), func() {}
}

type serviceProgram struct {
	configPath string
	logger     *log.Logger
}

func (p serviceProgram) Execute(_ []string, requests <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	status <- svc.Status{State: svc.StartPending}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); runProgram(ctx, p.configPath, p.logger) }()
	status <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	for {
		select {
		case request := <-requests:
			switch request.Cmd {
			case svc.Interrogate:
				status <- request.CurrentStatus
			case svc.Stop, svc.Shutdown:
				status <- svc.Status{State: svc.StopPending}
				cancel()
				<-done
				return false, 0
			}
		case <-done:
			return false, 1
		}
	}
}

func runProgram(ctx context.Context, configPath string, logger *log.Logger) {
	cfg, err := config.LoadTimeControl(configPath)
	if err != nil {
		logger.Printf("invalid configuration: %v", err)
		return
	}
	hub := ipc.NewHub()
	service := timecontrol.NewService(cfg, hub, logger)
	pipe := ipc.PipeServer{Hub: hub, Initial: service.CurrentMessage, OnAgentMessage: service.HandleAgentMessage}
	go func() {
		if err := pipe.Serve(ctx); err != nil && ctx.Err() == nil {
			logger.Printf("named pipe stopped: %v", err)
		}
	}()
	if err := service.Run(ctx); err != nil && ctx.Err() == nil {
		logger.Printf("service stopped: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
}

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"forlittle/windows-agent/internal/agent"
	"forlittle/windows-agent/internal/config"
)

func main() {
	configPath := flag.String("config", "config.json", "path to agent config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	runner := agent.New(cfg, log.New(os.Stdout, "forlittle-agent: ", log.LstdFlags))
	if err := runner.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "agent stopped: %v\n", err)
		os.Exit(1)
	}
}

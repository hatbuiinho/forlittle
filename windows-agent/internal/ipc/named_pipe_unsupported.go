//go:build !windows

package ipc

import (
	"context"
	"errors"

	"forlittle/windows-agent/internal/timecontrol"
)

const DefaultPipeName = `\\.\pipe\ForLittleTimeControl`

type PipeServer struct {
	Hub            *Hub
	Initial        func() timecontrol.StateMessage
	Name           string
	OnAgentMessage func(timecontrol.AgentMessage)
}

func (PipeServer) Serve(context.Context) error {
	return errors.New("named pipes are only available on Windows")
}

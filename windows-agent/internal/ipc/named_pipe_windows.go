//go:build windows

package ipc

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"

	"forlittle/windows-agent/internal/timecontrol"

	"github.com/Microsoft/go-winio"
)

const DefaultPipeName = `\\.\pipe\ForLittleTimeControl`

type PipeServer struct {
	Hub            *Hub
	Initial        func() timecontrol.StateMessage
	Name           string
	OnAgentMessage func(timecontrol.AgentMessage)
	OnError        func(error)
}

func (s PipeServer) Serve(ctx context.Context) error {
	name := s.Name
	if name == "" {
		name = DefaultPipeName
	}
	listener, err := winio.ListenPipe(name, &winio.PipeConfig{SecurityDescriptor: "D:P(A;;GA;;;SY)(A;;GA;;;BA)(A;;GRGW;;;IU)"})
	if err != nil {
		return err
	}
	defer listener.Close()
	go func() { <-ctx.Done(); _ = listener.Close() }()
	for {
		connection, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		go s.serveConnection(ctx, connection)
	}
}

func (s PipeServer) serveConnection(ctx context.Context, connection net.Conn) {
	defer connection.Close()
	messages, unsubscribe := s.Hub.Subscribe()
	defer unsubscribe()
	if s.Initial != nil && !writeMessage(connection, s.Initial()) {
		s.report(fmt.Errorf("write initial agent state: failed"))
		return
	}

	done := make(chan error, 1)
	go func() { done <- readAgentMessages(connection, s.OnAgentMessage) }()
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-done:
			if err != nil {
				s.report(fmt.Errorf("read agent messages: %w", err))
			}
			return
		case message, ok := <-messages:
			if !ok || !writeMessage(connection, message) {
				if ok {
					s.report(fmt.Errorf("write agent state: failed"))
				}
				return
			}
		}
	}
}

func (s PipeServer) report(err error) {
	if s.OnError != nil {
		s.OnError(err)
	}
}

func readAgentMessages(connection net.Conn, callback func(timecontrol.AgentMessage)) error {
	scanner := bufio.NewScanner(connection)
	scanner.Buffer(make([]byte, 256), 4096)
	for scanner.Scan() {
		if callback == nil {
			continue
		}
		var message timecontrol.AgentMessage
		if json.Unmarshal(scanner.Bytes(), &message) == nil {
			callback(message)
		}
	}
	return scanner.Err()
}

func writeMessage(connection net.Conn, message timecontrol.StateMessage) bool {
	data, err := json.Marshal(message)
	if err != nil {
		return false
	}
	_, err = connection.Write(append(data, '\n'))
	return err == nil
}

package timecontrol

import (
	"bytes"
	"testing"
	"time"
)

func TestCommandBrokerDeliversOnlyToTargetMachine(t *testing.T) {
	broker := NewCommandBroker()
	target, cancelTarget := broker.Subscribe("PC-01")
	defer cancelTarget()
	other, cancelOther := broker.Subscribe("PC-02")
	defer cancelOther()

	broker.Notify("PC-01", []byte(`{"type":"COMMAND_AVAILABLE"}`))
	select {
	case message := <-target:
		if !bytes.Contains(message, []byte("COMMAND_AVAILABLE")) {
			t.Fatalf("unexpected message: %s", message)
		}
	case <-time.After(time.Second):
		t.Fatal("target machine did not receive notification")
	}

	select {
	case <-other:
		t.Fatal("other machine received a notification")
	case <-time.After(20 * time.Millisecond):
	}
}

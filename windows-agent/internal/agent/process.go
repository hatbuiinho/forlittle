package agent

type ChromeProcess struct {
	PID       int
	ParentPID int

	CommandLine string
}

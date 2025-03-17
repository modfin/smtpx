package smtpx

const (
	Name    = "Brevx"
	Version = "0.0.1"
)

const CommandVerbMaxLength = 16
const CommandLineMaxLength = 1024

const (
	// Server has just been created
	ServerStateNew = iota
	// Server has just been stopped
	ServerStateStopped
	// Server has been started and is running
	ServerStateRunning
	// Server could not start due to an error
	ServerStateStartError
)

const (
	defaultMaxClients = 100
	defaultTimeout    = 30
	defaultInterface  = ":2525"
	defaultMaxSize    = 10_485_760 // int64(10 << 20) // 10 Megabytes

	defaultMaxRecipients           = 100 //  RFC5321LimitRecipients
	defaultMaxUnrecognizedCommands = 5
)

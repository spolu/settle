package cli

// CmdName represents a command name.
type CmdName string

// Command is the interface for a cli command.
type Command interface {
	// Name returns the command name.
	Name() CmdName

	// Help prints out the help message for the command.
	Help()

	// Parse the arguments passed to the command.
	Parse([]string) error

	// Execute the command or return a human-friendly error.
	Execute([]string) error
}

// Registrar is used to register command generators within the module.
var Registrar = map[CmdName](func() Command){}

// Cli represents a cli instance.
type Cli struct {
}

package command

import (
	"context"

	"github.com/spolu/settle/cli"
	"github.com/spolu/settle/lib/out"
)

const (
	// CmdNmHelp is the command name.
	CmdNmHelp cli.CmdName = "help"
)

func init() {
	cli.Registrar[CmdNmHelp] = NewHelp
}

// Help a user up to a certain amount of a given asset they issued.
type Help struct {
	Command cli.Command
}

// NewHelp constructs and initializes the command.
func NewHelp() cli.Command {
	return &Help{}
}

// Name returns the command name.
func (c *Help) Name() cli.CmdName {
	return CmdNmHelp
}

// Help prints out the help message for the command.
func (c *Help) Help(
	ctx context.Context,
) {
	out.Normf("\nUsage: ")
	out.Boldf("settle <command> [<args> ...]\n")
	out.Normf("\n")
	out.Normf("  Decentralized trust graph for value exchange on the Internet.\n")
	out.Normf("\n")
	out.Normf("Commands:\n")

	out.Boldf("  help <command>\n")
	out.Normf("    Show help for a command.\n")
	out.Valuf("    settle help trust\n")
	out.Normf("\n")

	out.Boldf("  mint <asset>\n")
	out.Normf("    Creates a new asset.\n")
	out.Valuf("    settle mint USD.2\n")
	out.Normf("\n")

	out.Boldf("  trust <user> <asset> <amount>\n")
	out.Normf("    Trust a user up to a certain amount of an asset.\n")
	out.Valuf("    settle trust von.neumann@ias.edu EUR.2 200\n")
	out.Normf("\n")

	out.Boldf("  pay <asset> <amount> to <user>\n")
	out.Normf("    Pay a user.\n")
	out.Valuf("    settle pay GBP.2 20 to von.neumann@ias.edu\n")
	out.Normf("\n")

	out.Boldf("  list <object>\n")
	out.Normf("    List balances, assets, trustlines.\n")
	out.Valuf("    settle list balances\n")
	out.Normf("\n")

	out.Boldf("  login\n")
	out.Normf("    Log into a mint.\n")
	out.Valuf("    settle login\n")
	out.Normf("\n")

	out.Boldf("  register\n")
	out.Normf("    Register on publicy available mints.\n")
	out.Valuf("    settle login\n")
	out.Normf("\n")

	out.Boldf("  logout\n")
	out.Normf("    Log the current user out.\n")
	out.Valuf("    settle logout\n")
	out.Normf("\n")
}

// Parse parses the arguments passed to the command.
func (c *Help) Parse(
	ctx context.Context,
	args []string,
) error {
	if len(args) == 0 {
		c.Command = NewHelp()
	} else {
		if r, ok := cli.Registrar[cli.CmdName(args[0])]; !ok {
			c.Command = NewHelp()
		} else {
			c.Command = r()
		}
	}
	return nil
}

// Execute the command or return a human-friendly error.
func (c *Help) Execute(
	ctx context.Context,
) error {
	c.Command.Help(ctx)
	return nil
}

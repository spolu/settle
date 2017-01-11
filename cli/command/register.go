package command

import (
	"bufio"
	"context"
	"os"
	"strconv"
	"strings"

	"github.com/spolu/settle/cli"
	"github.com/spolu/settle/lib/env"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/out"
)

const (
	// CmdNmRegister is the command name.
	CmdNmRegister cli.CmdName = "register"
)

func init() {
	cli.Registrar[CmdNmRegister] = NewRegister
}

// Register a user up to a certain amount of a given asset they issued.
type Register struct {
}

// NewRegister constructs and initializes the command.
func NewRegister() cli.Command {
	return &Register{}
}

// Name returns the command name.
func (c *Register) Name() cli.CmdName {
	return CmdNmRegister
}

// Help prints out the help message for the command.
func (c *Register) Help(
	ctx context.Context,
) {
	out.Normf("\nUsage: ")
	out.Boldf("settle register\n")
	out.Normf("\n")
	out.Normf("  Registering lets you create an account on a list of publicly available mints\n")
	out.Normf("  directly from the settle command line.")
	out.Normf("\n\n")
	out.Normf("  List of available mints:\n")
	for i, r := range PublicMints {
		out.Normf("    (%d) ", i)
		out.Boldf("%s", r.Name)
		out.Normf(" [")
		out.Valuf("%s", r.Host)
		out.Normf("] ")
		out.Normf("%s", r.Description)
	}
	out.Normf("\n\n")
}

// Parse parses the arguments passed to the command.
func (c *Register) Parse(
	ctx context.Context,
	args []string,
) error {
	return nil
}

// Execute the command or return a human-friendly error.
func (c *Register) Execute(
	ctx context.Context,
) error {

	out.Normf("  List of available mints:\n")
	for i, r := range PublicMints {
		out.Normf("    (%d) ", i)
		out.Boldf("%s", r.Name)
		out.Normf(" [")
		out.Valuf("%s", r.Host)
		out.Normf("] ")
		out.Normf("%s", r.Description)
	}
	out.Normf("\n\n")

	reader := bufio.NewReader(os.Stdin)

	out.Normf("Mint selection [0]: ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	i := int64(0)
	if choice != "" {
		var err error
		i, err = strconv.ParseInt(choice, 10, 8)
		if err != nil || i < 0 || i >= int64(len(PublicMints)) {
			return errors.Trace(errors.Newf("Invalid choice: %s", choice))
		}
	}
	register := PublicMints[i]

	out.Normf("       Username []: ")
	username, _ := reader.ReadString('\n')

	out.Normf("          Email []: ")
	email, _ := reader.ReadString('\n')

	// Register the user.
	_, err := RegisterUser(ctx,
		register,
		strings.TrimSpace(username),
		strings.TrimSpace(email),
	)
	if err != nil {
		return errors.Trace(err)
	}

	out.Normf("\n")
	out.Normf("Success! Check your inbox for an email containing instructions to retrieve your\n")
	out.Normf("credentials. Then, run: ")
	if env.Get(ctx).Environment == env.QA {
		out.Boldf("settle -env=qa login\n")
	} else {
		out.Boldf("settle login\n")
	}

	return nil
}

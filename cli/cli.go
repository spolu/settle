package cli

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/spolu/settle/lib/env"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/out"
)

// CmdName represents a command name.
type CmdName string

// ContextKey is the type of the key used with context to contextual data.
type ContextKey string

// Command is the interface for a cli command.
type Command interface {
	// Name returns the command name.
	Name() CmdName

	// Help prints out the help message for the command.
	Help(context.Context)

	// Parse the arguments passed to the command.
	Parse(context.Context, []string) error

	// Execute the command or return a human-friendly error.
	Execute(context.Context) error
}

// Registrar is used to register command generators within the module.
var Registrar = map[CmdName](func() Command){}

// Cli represents a cli instance.
type Cli struct {
	Ctx   context.Context
	Flags map[string]string
	Args  []string
}

// flagFilterRegexp filters out flags from arguments.
var flagFilterRegexp = regexp.MustCompile("^-+")

// New initializes a new Cli by parsing the passed arguments.
func New(
	argv []string,
) (*Cli, error) {
	ctx := context.Background()

	args := []string{}
	flags := map[string]string{}

	for _, a := range argv {
		if flagFilterRegexp.MatchString(a) {
			a = strings.Trim(a, "-")
			s := strings.Split(a, "=")
			if len(s) == 2 {
				flags[s[0]] = s[1]
			}
		} else {
			args = append(args, strings.TrimSpace(a))
		}
	}

	cliEnv := env.Env{
		Environment: env.Production,
		Config:      map[env.ConfigKey]string{},
	}

	// Environment flag.
	if e, ok := flags["env"]; ok && e == "qa" {
		cliEnv.Environment = env.QA
	}
	ctx = env.With(ctx, &cliEnv)

	creds, err := CurrentUser(ctx)
	if err != nil {
		return nil, errors.Trace(err)
	}

	ctx = WithCredentials(ctx, creds)

	user := "n/a"
	if creds != nil {
		user = fmt.Sprintf("%s@%s", creds.Username, creds.Host)
	}
	out.Statf("[Initializing] env=%s user=%s\n", cliEnv.Environment, user)

	return &Cli{
		Ctx:   ctx,
		Args:  args,
		Flags: flags,
	}, nil
}

// Run the cli.
func (c *Cli) Run() error {
	if len(c.Args) == 0 {
		c.Args = append(c.Args, "help")
	}

	var command Command
	cmd, args := c.Args[0], c.Args[1:]
	if r, ok := Registrar[CmdName(cmd)]; !ok {
		command = Registrar[CmdName("help")]()
	} else {
		command = r()
	}

	err := command.Parse(c.Ctx, args)
	if err != nil {
		command.Help(c.Ctx)
		return errors.Trace(err)
	}

	err = command.Execute(c.Ctx)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

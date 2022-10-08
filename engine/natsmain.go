package engine

import (
	"flag"
	"fmt"
	"os"

	"github.com/nats-io/nats-server/v2/server"
)

// StartNATSServer xxx
func StartNATSServer() {

	exe := "nats-server"

	// Create a FlagSet and sets the usage
	fs := flag.NewFlagSet(exe, flag.ExitOnError)

	natsconf := ConfigValue("natsconf")
	if natsconf == "" {
		natsconf = "natsalone.conf"
	}
	// Configure the options from the flags/config file
	conf := ConfigFilePath(natsconf)
	args := []string{"-c", conf}

	opts, err := server.ConfigureOptions(fs, args,
		server.PrintServerAndExit,
		fs.Usage,
		server.PrintTLSHelpAndDie)
	if err != nil {
		server.PrintAndDie(fmt.Sprintf("%s: %s", exe, err))
	} else if opts.CheckConfig {
		fmt.Fprintf(os.Stderr, "%s: configuration file %s is valid\n", exe, opts.ConfigFile)
		os.Exit(0)
	}

	// Create the server with appropriate options.
	s, err := server.NewServer(opts)
	if err != nil {
		server.PrintAndDie(fmt.Sprintf("%s: %s", exe, err))
	}

	// Configure the logger based on the flags
	s.ConfigureLogger()

	// Start things up. Block here until done.
	if err := server.Run(s); err != nil {
		server.PrintAndDie(err.Error())
	}
	s.WaitForShutdown()
}

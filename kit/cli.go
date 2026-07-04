package kit

import (
	"os"
)

// RunCLICommand is the shared tail of the palette command-line mains: it
// executes fn(args), prints the result (or the error) to stdout, and exits
// non-zero on failure so scripts can detect it.
func RunCLICommand(args []string, fn func([]string) (map[string]string, error)) {
	apiout, err := fn(args)
	if err != nil {
		os.Stdout.WriteString("Error: " + err.Error() + "\n")
		LogError(err)
		os.Exit(1)
	}
	os.Stdout.WriteString(HumanReadableAPIOutput(apiout))
}

package kit

import (
	"encoding/json"

	// "github.com/nats-io/nats-server/v2/server"
	// "github.com/nats-io/nuid"
)

/*
// StartNATSServer xxx
func StartNATSServer() {

	_ = MyNUID() // to make sure nuid.json is initialized

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
*/

var myNUID = ""

// MyNUID xxx
func MyNUID() string {
	if myNUID == "" {
		myNUID = GetNUID()
	}
	return myNUID
}

// GetNUID xxx
func GetNUID() string {
	bytes, err := TheHost.GetConfigFileData("nuid.json")
	if err != nil {
		LogError(err)
		return "FakeNUID"
	}
	var f any
	err = json.Unmarshal(bytes, &f)
	if err != nil {
		LogError(err)
		return "FakeNUID"
	}
	toplevel := f.(map[string]any)
	t, ok := toplevel["nuid"]
	nuid, ok2 := t.(string)
	if !ok || !ok2 {
		LogWarn("No nuid in nuid.json")
		return "FakeNUID"
	}
	return nuid
	/*
	file, err := os.OpenFile(nuidpath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("InitLogs: Unable to open %s err=%s", nuidpath, err)
		return "UnableToOpenNUIDFile"
	}
	nuid := nuid.Next()
	file.WriteString("{\n\t\"nuid\": \"" + nuid + "\"\n}\n")
	file.Close()
	log.Printf("GetNUID: generated nuid.json for %s\n", nuid)
	return nuid
	*/
}

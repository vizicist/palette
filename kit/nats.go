package kit

import (
	"fmt"
	"net/url"
	"time"

	"github.com/joho/godotenv"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

var (
	natsLeafServer      *server.Server = nil
	natsConn        *nats.Conn     = nil
	natsIsConnected bool           = false
)

func NatsInit() {

	err := NatsStartLeafServer()
	if err != nil {
		LogError(err)
		return
	}
	LogIfError(err)

	if natsIsConnected {
		// Already connected
		LogError(fmt.Errorf("NatsInit: Already connected!"))
		return
	}

	url := "127.0.0.1:4222"

	// Connect Options.
	opts := []nats.Option{nats.Name("Palette NATS Subscriber")}
	opts = setupConnOptions(opts)

	/*
		// Use UserCredentials
		var userCreds = "" // User Credentials File
		if userCreds != "" {
			opts = append(opts, nats.UserCredentials(userCreds))
		}
	*/

	// Connect to NATS
	nc, err := nats.Connect(url, opts...)
	if err != nil {
		natsIsConnected = false
		LogError(fmt.Errorf("nats.Connect failed, fullurl=%s err=%s", url, err))
		return
	}
	natsIsConnected = true
	natsConn = nc

	LogInfo("Successful connect to NATS", "url", url)

	date := time.Now().Format(PaletteTimeLayout)
	msg := fmt.Sprintf("Successful connection from hostname=%s date=%s", Hostname(), date)
	NatsPublishFromEngine("connect.info", msg)

	subscribeTo := fmt.Sprintf("to_palette.%s.>", Hostname())
	LogInfo("Subscribing to NATS", "subscribeTo", subscribeTo)
	err = NatsSubscribe(subscribeTo, natsRequestHandler)
	LogIfError(err)
}

func natsRequestHandler(msg *nats.Msg) {
	data := string(msg.Data)
	LogInfo("NatsHandler", "subject", msg.Subject, "data", data)
	result, err := ExecuteApiFromJson(data)
	var response string
	if err != nil {
		LogError(fmt.Errorf("natsRequestHandler unable to interpret"), "data", data)
		response = ErrorResponse(err)
	} else {
		response = ResultResponse(result)
	}
	bytes := []byte(response)
	// Send the response.
	err = msg.Respond(bytes)
	LogIfError(err)
}

// Request is used for APIs - it blocks waiting for a response and returns the response
func NatsRequest(subj, data string, timeout time.Duration) (retdata string, err error) {
	if !natsIsConnected {
		return "", fmt.Errorf("NatsRequest: called when NATS is not Connected")
	}

	LogOfType("nats", "NatsRequest", "subject", subj, "data", data)
	if natsConn == nil {
		return "", fmt.Errorf("Viznats.Request: no NATS connection")
	}
	bytes := []byte(data)
	msg, err := natsConn.Request(subj, bytes, timeout)
	if err == nats.ErrTimeout {
		return "", fmt.Errorf("timeout, nothing is subscribed to subj=%s", subj)
	} else if err != nil {
		return "", fmt.Errorf("error: subj=%s err=%s", subj, err)
	}
	return string(msg.Data), nil
}

// Publish xxx
func NatsPublish(subj string, msg string) error {

	if !natsIsConnected {
		return fmt.Errorf("NatsPublish: called when NATS is not Connected")
	}

	nc := natsConn
	if !natsIsConnected || nc == nil {
		return fmt.Errorf("Viznats.Publish: no NATS connection, subject=%s", subj)
	}
	bytes := []byte(msg)

	LogInfo("Nats.Publish", "subject", subj, "msg", msg)

	err := natsConn.Publish(subj, bytes)
	LogIfError(err)
	nc.Flush()

	if err := nc.LastError(); err != nil {
		return err
	}
	return nil
}

// Subscribe xxx
func NatsSubscribe(subj string, callback nats.MsgHandler) error {

	if !natsIsConnected {
		return fmt.Errorf("NatsSubscribe: called when NATS is not Connected")
	}

	LogInfo("NatsSubscribe", "subject", subj)

	nc := natsConn
	if nc == nil {
		return fmt.Errorf("NatsSubscribe: subject=%s, no connection to NATS server", subj)
	}
	_, err := nc.Subscribe(subj, callback)
	LogIfError(err)
	nc.Flush()

	return nc.LastError()
}

func NatsClose() {
	if natsConn == nil || !natsIsConnected {
		LogError(fmt.Errorf("NatsClose called when natsConn is nil or unconnected"))
		return
	}
	natsConn.Close()
	natsIsConnected = false
	LogError(fmt.Errorf("NatsCLose called"))
}

func setupConnOptions(opts []nats.Option) []nats.Option {
	totalWait := 48 * time.Hour
	reconnectDelay := 5 * time.Second

	opts = append(opts, nats.ReconnectWait(reconnectDelay))
	opts = append(opts, nats.MaxReconnects(int(totalWait/reconnectDelay)))
	opts = append(opts, nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
		natsIsConnected = false
		LogWarn("nats.Disconnected",
			"err", err,
			"waitminutes", totalWait.Minutes())
	}))
	opts = append(opts, nats.ReconnectHandler(func(nc *nats.Conn) {
		natsIsConnected = true
		LogWarn("nats.Reconnected", "connecturl", nc.ConnectedUrl())
	}))
	opts = append(opts, nats.ClosedHandler(func(nc *nats.Conn) {
		natsIsConnected = false
		LogWarn("nats.ClosedHandler",
			"lasterror", nc.LastError())

	}))
	return opts
}

// NatsPublishFromEngine sends an asynchronous message via NATS
func NatsPublishFromEngine(subject string, msg string) {
	if !natsIsConnected {
		// silent, but perhaps you could log it every once in a while
		return
	}
	fullsubject := fmt.Sprintf("from_palette.%s.%s", Hostname(), subject)
	err := NatsPublish(fullsubject, msg)
	LogIfError(err)
}

func NatsDisconnect() {
	natsConn.Close()
	natsIsConnected = false
}

/*
func printVersion() {
	LogInfo("PrintVersion")
}
func printUsage() {
	LogInfo("PrintUsage")
}
func printTLSHelp() {
	LogInfo("PrintTLSHelp")
}
*/

func NatsStartLeafServer() error {

	if natsLeafServer != nil {
		return fmt.Errorf("NatsStartLeafServer: NATS leaf Server already started")
	}

	path := ConfigFilePath(".env")
	myenv, err := godotenv.Read(path)
	if err != nil {
		return fmt.Errorf("Error reading .env for NATS leaf server")
	}
	hubStr, ok := myenv["NATS_HUB_URL"]
	if !ok {
		return fmt.Errorf("No NATS_HUB_URL value, use 'palette env set' to set")
	}

	hubUrl, err := url.Parse(hubStr)
	if err != nil {
		return fmt.Errorf("Unable to parse url value - %s", hubUrl)
	}

	leafOptions := &server.Options{
		ServerName: "leaf-server",
		Port:       4222, // Port for local clients to connect to the leaf node
		LeafNode: server.LeafNodeOpts{
			Remotes: []*server.RemoteLeafOpts{
				{
					URLs: []*url.URL{
						hubUrl, // Connect to hub's leafnode port
					},
				},
			},
			TLSConfig: nil, // Optional TLS configuration if needed
		},
	}

	// Create the server with appropriate options.
	s, err := server.NewServer(leafOptions)
	if err != nil {
		return err
	}

	s.ConfigureLogger()

	// Start the server up in the background
	if err := server.Run(s); err != nil {
		return err
	}

	natsLeafServer = s

	return nil
}

func NatsWaitForShutdown() {
	natsLeafServer.WaitForShutdown()
}

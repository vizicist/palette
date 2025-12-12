package kit

import (
	"context"
	"fmt"
	"net/url"
	"time"

	json "github.com/goccy/go-json"

	"github.com/joho/godotenv"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

var (
	natsLeafServer  *server.Server = nil
	natsConn        *nats.Conn     = nil
	natsIsConnected bool           = false
)

func NatsConnectLocalAndSubscribe() {

	if natsIsConnected {
		// Already connected
		LogError(fmt.Errorf("NatsInit: Already connected"))
		return
	}

	url := "127.0.0.1:4222"

	// Connect Options.
	opts := []nats.Option{nats.Name("Palette NATS Subscriber")}
	opts = setupConnOptions(opts)

	// Connect to NATS
	LogInfo("Calling nats.Connect A", "url", url)
	nc, err := nats.Connect(url, opts...)
	if err != nil {
		natsIsConnected = false
		LogError(fmt.Errorf("nats.Connect failed, fullurl=%s err=%s", url, err))
		return
	}
	natsIsConnected = true
	natsConn = nc

	subscribeTo := fmt.Sprintf("to_palette.%s.>", Hostname())
	err = NatsSubscribe(subscribeTo, natsRequestHandler)
	if err != nil {
		LogError(err)
	} else {
		LogInfo("Connected and subscribing to NATS", "subscribeTo", subscribeTo)
		NatsPublishFromEngine("connect.info", map[string]any{
			"hostname": Hostname(),
		})
	}

}

func NatsConnectRemote() error {

	if natsIsConnected {
		// Already connected
		return fmt.Errorf("NatsInit: Already connected")
	}

	url, err := NatsEnvValue("NATS_HUB_CLIENT_URL")
	if err != nil {
		return err
	}

	// Connect Options.
	opts := []nats.Option{nats.Name("Palette NATS Subscriber")}
	opts = setupConnOptions(opts)

	LogInfo("Calling nats.Connect B", "url", url)

	// Connect to NATS
	nc, err := nats.Connect(url, opts...)
	if err != nil {
		natsIsConnected = false
		return fmt.Errorf("nats.Connect failed, url=%s err=%s", url, err)
	}
	natsIsConnected = true
	natsConn = nc

	LogInfo("Successful connect to remote NATS", "url", url)
	return nil
}

func NatsDump(streamName string, f func(tm time.Time, subj string, data string)) error {
	return NatsDumpTimeRange(streamName, nil, nil, f)
}

// NatsDumpTimeRange dumps messages from a stream within an optional time range.
// If startTime is nil, starts from the beginning. If endTime is nil, continues to the end.
func NatsDumpTimeRange(streamName string, startTime *time.Time, endTime *time.Time, f func(tm time.Time, subj string, data string)) error {

	if !natsIsConnected {
		return fmt.Errorf("NatsSummary: not Connected")
	}

	// Create a JetStream management context
	js, err := natsConn.JetStream()
	if err != nil {
		LogError(fmt.Errorf("error creating JetStream management context: %v", err))
	}

	// Get stream info to validate the stream exists
	_, err = js.StreamInfo(streamName)
	if err != nil {
		return err
	}

	// Build subscription options
	opts := []nats.SubOpt{nats.BindStream(streamName)}

	if startTime != nil {
		// Efficiently start from the specified time
		opts = append(opts, nats.StartTime(*startTime))
	} else {
		// Start from message 0
		opts = append(opts, nats.DeliverAll())
	}

	// Create an ephemeral pull subscription
	sub, err := js.PullSubscribe("", "", opts...)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {
		// Fetch messages in batches of 10
		msgs, err := sub.Fetch(10, nats.Context(ctx))
		if err != nil {
			if err == context.DeadlineExceeded {
				fmt.Println("No more messages to fetch.")
				break
			}
			LogError(fmt.Errorf("error fetching messages: %v", err))
		}

		for _, msg := range msgs {
			md, err := msg.Metadata()
			if err != nil {
				LogError(fmt.Errorf("error fetching message metadata: %v", err))
				break
			}

			// If endTime is specified and we've passed it, stop
			if endTime != nil && md.Timestamp.After(*endTime) {
				return nil
			}

			f(md.Timestamp, msg.Subject, string(msg.Data))
			err = msg.Ack() // Acknowledge the message
			if err != nil {
				LogError(fmt.Errorf("error in msg.Ack(): %v", err))
				break
			}
		}
	}
	return nil
}

func NatsStreams() ([]string, error) {

	if !natsIsConnected {
		return nil, fmt.Errorf("NatsSummary: not Connected")
	}

	// Create a JetStream management context
	jsm, err := natsConn.JetStream()
	if err != nil {
		return nil, fmt.Errorf("error creating JetStream management context: %v", err)
	}

	// List all streams
	streams := jsm.StreamNames()
	s := []string{}
	for stream := range streams {
		s = append(s, stream)
	}
	return s, nil
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
func NatsPublish(subj string, data map[string]any) error {

	nc := natsConn
	if !natsIsConnected || nc == nil {
		return fmt.Errorf("NatsPublish: no NATS connection, subject=%s", subj)
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// bytes := []byte("foobar")

	LogInfo("NatsPublish", "subject", subj, "data", string(bytes))

	err = natsConn.Publish(subj, bytes)
	LogIfError(err)
	nc.Flush()

	return nc.LastError()
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
func NatsPublishFromEngine(subject string, data map[string]any) {
	if !natsIsConnected {
		// silent, but perhaps you could log it every once in a while
		LogError(fmt.Errorf("NatsPublishFromEngine: called when NATS is not Connected"))
		return
	}
	fullsubject := fmt.Sprintf("from_palette.%s.%s", Hostname(), subject)
	err := NatsPublish(fullsubject, data)
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

func NatsEnvValue(key string) (string, error) {
	path := ConfigFilePath(".env")
	myenv, err := godotenv.Read(path)
	if err != nil {
		return "", fmt.Errorf("error reading .env for NATS_*_URL values")
	}
	s, ok := myenv[key]
	if !ok {
		return "", fmt.Errorf("no %s value, use 'palette env set' to set", key)
	}
	return s, nil
}

func NatsStartLeafServer() error {

	if natsLeafServer != nil {
		return fmt.Errorf("NatsStartLeafServer: NATS leaf Server already started")
	}

	hubStr, err := NatsEnvValue("NATS_HUB_LEAF_URL")
	if err != nil {
		return err
	}

	huburl, err := url.Parse(hubStr)
	if err != nil {
		return fmt.Errorf("unable to parse url value - %s", huburl)
	}

	leafName := Hostname() + "-leaf-server"

	leafOptions := &server.Options{
		ServerName: leafName,
		Port:       4222, // Port for local clients to connect to the leaf node
		LeafNode: server.LeafNodeOpts{
			Remotes: []*server.RemoteLeafOpts{
				{
					URLs: []*url.URL{
						huburl, // Connect to hub's leafnode port
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

	LogInfo("Calling nats.server.Run", "huburl", huburl)
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

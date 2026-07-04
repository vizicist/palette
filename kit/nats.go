package kit

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	json "github.com/goccy/go-json"

	"github.com/nats-io/nats.go"
)

// maskURLPassword replaces the password in a URL with X's
func maskURLPassword(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	if u.User != nil {
		if _, hasPassword := u.User.Password(); hasPassword {
			u.User = url.UserPassword(u.User.Username(), strings.Repeat("X", 8))
		}
	}
	return u.String()
}

// The NATS connection is written by connect/disconnect paths and by the
// NATS client's own callback goroutines (disconnect/reconnect/closed), and
// read from HTTP/API/scheduler goroutines — so all access goes through
// these mutex-guarded helpers.
var (
	natsMutex       sync.Mutex
	natsConn        *nats.Conn = nil
	natsIsConnected bool       = false
)

// natsConnection returns the current connection, or nil if not connected.
func natsConnection() *nats.Conn {
	natsMutex.Lock()
	defer natsMutex.Unlock()
	if !natsIsConnected {
		return nil
	}
	return natsConn
}

// setNatsConnection installs (or, with nil, clears) the connection.
func setNatsConnection(nc *nats.Conn) {
	natsMutex.Lock()
	natsConn = nc
	natsIsConnected = nc != nil
	natsMutex.Unlock()
}

// takeNatsConnection clears the connection and returns what was there.
func takeNatsConnection() *nats.Conn {
	natsMutex.Lock()
	defer natsMutex.Unlock()
	nc := natsConn
	natsConn = nil
	natsIsConnected = false
	return nc
}

// setNatsConnected flips just the connected flag; used by the NATS
// disconnect/reconnect callbacks, where the connection object persists.
func setNatsConnected(connected bool) {
	natsMutex.Lock()
	natsIsConnected = connected
	natsMutex.Unlock()
}

// NatsIsConnected reports whether a NATS connection is currently usable.
func NatsIsConnected() bool {
	return natsConnection() != nil
}

func StartEmbeddedNATSAndConnectEngine() {

	if NatsIsConnected() {
		// Already connected
		LogError(fmt.Errorf("StartEmbeddedNATSAndConnectEngine: Already connected"))
		return
	}

	if err := StartEmbeddedLocalNATSServer(); err != nil {
		LogError(err)
		return
	}
	url := EmbeddedNATSURL()

	// Connect Options.
	opts := []nats.Option{nats.Name("Palette Engine Local NATS Subscriber")}
	opts = setupConnOptions(opts)

	// Connect to the embedded local server. The server owns the leaf
	// connection to the hub, keeping palette.local.> traffic local-only.
	LogInfo("Connecting to embedded local NATS", "url", maskURLPassword(url))
	nc, err := nats.Connect(url, opts...)
	if err != nil {
		setNatsConnection(nil)
		LogError(fmt.Errorf("nats.Connect to embedded local server failed, url=%s err=%w", maskURLPassword(url), err))
		return
	}
	setNatsConnection(nc)

	subscribeTo := fmt.Sprintf("to_palette.%s.>", Hostname())
	err = SubscribeEngineAPIOverNATS(subscribeTo)
	if err != nil {
		LogError(err)
	} else {
		LogInfo("Connected to embedded local NATS and subscribed", "subscribeTo", subscribeTo)
		NatsPublishFromEngine("connect.info", map[string]any{
			"hostname": Hostname(),
		})
	}

}

func SubscribeEngineAPIOverNATS(subject string) error {
	return NatsSubscribe(subject, natsEngineAPIHandler)
}

func NatsConnectLocal() error {
	url, err := NatsEnvValue("NATS_URL")
	if err != nil {
		return err
	}
	return natsConnect(url)
}

func NatsConnectRemote() error {
	url, err := NatsEnvValue("NATS_URL")
	if err != nil {
		return err
	}
	return natsConnect(url)
}

func natsConnect(url string) error {
	if NatsIsConnected() {
		return fmt.Errorf("natsConnect: Already connected")
	}

	opts := []nats.Option{nats.Name("Palette NATS Subscriber")}
	opts = setupConnOptions(opts)

	LogInfo("Connecting to NATS", "url", maskURLPassword(url))

	nc, err := nats.Connect(url, opts...)
	if err != nil {
		setNatsConnection(nil)
		return fmt.Errorf("nats.Connect failed, url=%s err=%w", maskURLPassword(url), err)
	}
	setNatsConnection(nc)

	LogInfo("Connected to NATS", "url", maskURLPassword(url))
	return nil
}

func NatsDump(streamName string, f func(tm time.Time, subj string, data string)) error {
	return NatsDumpTimeRange(streamName, nil, nil, f)
}

// NatsDumpTimeRange dumps messages from a stream within an optional time range.
// If startTime is nil, starts from the beginning. If endTime is nil, continues to the end.
func NatsDumpTimeRange(streamName string, startTime *time.Time, endTime *time.Time, f func(tm time.Time, subj string, data string)) error {

	nc := natsConnection()
	if nc == nil {
		return fmt.Errorf("NatsDumpTimeRange: not Connected")
	}

	// Create a JetStream management context
	js, err := nc.JetStream()
	if err != nil {
		return fmt.Errorf("error creating JetStream management context: %w", err)
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
			// A persistent fetch error would otherwise spin this loop
			// forever, logging the same failure; fail fast instead.
			return fmt.Errorf("error fetching messages: %w", err)
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

	nc := natsConnection()
	if nc == nil {
		return nil, fmt.Errorf("NatsStreams: not Connected")
	}

	// Create a JetStream management context
	jsm, err := nc.JetStream()
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

func natsEngineAPIHandler(msg *nats.Msg) {
	data := string(msg.Data)
	LogInfo("natsEngineAPIHandler", "subject", msg.Subject, "data", data)
	result, err := ExecuteAPIFromJSON(data)
	var response string
	if err != nil {
		LogError(fmt.Errorf("natsEngineAPIHandler unable to interpret"), "data", data)
		response = ErrorResponse(err)
	} else {
		response = ResultResponse(result)
	}
	bytes := []byte(response)
	// Send the response.
	err = msg.Respond(bytes)
	LogIfError(err)
}

// NatsRequest is used for APIs - it blocks waiting for a response and returns the response
func NatsRequest(subj, data string, timeout time.Duration) (retdata string, err error) {
	nc := natsConnection()
	if nc == nil {
		return "", fmt.Errorf("NatsRequest: called when NATS is not Connected")
	}

	LogOfType("nats", "NatsRequest", "subject", subj, "data", data)
	bytes := []byte(data)
	msg, err := nc.Request(subj, bytes, timeout)
	if err == nats.ErrTimeout {
		return "", fmt.Errorf("timeout, nothing is subscribed to subj=%s", subj)
	} else if err != nil {
		return "", fmt.Errorf("error: subj=%s err=%w", subj, err)
	}
	return string(msg.Data), nil
}

// NatsPublish xxx
func NatsPublish(subj string, data map[string]any) error {

	nc := natsConnection()
	if nc == nil {
		return fmt.Errorf("NatsPublish: no NATS connection, subject=%s", subj)
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	LogInfo("NatsPublish", "subject", subj, "data", string(bytes))

	err = nc.Publish(subj, bytes)
	LogIfError(err)
	nc.Flush()

	return nc.LastError()
}

func NatsPublishJSON(subj string, data any) error {
	nc := natsConnection()
	if nc == nil {
		return fmt.Errorf("NatsPublishJSON: no NATS connection, subject=%s", subj)
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	err = nc.Publish(subj, bytes)
	LogIfError(err)
	return nc.LastError()
}

// NatsSubscribe subscribes to the given subject using the provided callback.
func NatsSubscribe(subj string, callback nats.MsgHandler) error {

	nc := natsConnection()
	if nc == nil {
		return fmt.Errorf("NatsSubscribe: called when NATS is not Connected, subject=%s", subj)
	}

	LogOfType("nats", "NatsSubscribe", "subject", subj)
	_, err := nc.Subscribe(subj, callback)
	LogIfError(err)
	nc.Flush()

	return nc.LastError()
}

func NatsClose() {
	nc := takeNatsConnection()
	if nc == nil {
		LogError(fmt.Errorf("NatsClose called when natsConn is nil or unconnected"))
		return
	}
	nc.Close()
	LogInfo("NatsClose called")
}

func setupConnOptions(opts []nats.Option) []nats.Option {
	totalWait := 48 * time.Hour
	reconnectDelay := 5 * time.Second

	opts = append(opts, nats.ReconnectWait(reconnectDelay))
	opts = append(opts, nats.MaxReconnects(int(totalWait/reconnectDelay)))
	opts = append(opts, nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
		setNatsConnected(false)
		LogWarn("nats.Disconnected",
			"err", err,
			"waitminutes", totalWait.Minutes())
	}))
	opts = append(opts, nats.ReconnectHandler(func(nc *nats.Conn) {
		setNatsConnected(true)
		LogWarn("nats.Reconnected", "connecturl", nc.ConnectedUrl())
	}))
	opts = append(opts, nats.ClosedHandler(func(nc *nats.Conn) {
		setNatsConnected(false)
		LogWarn("nats.ClosedHandler",
			"lasterror", nc.LastError())

	}))
	return opts
}

// NatsPublishFromEngine sends an asynchronous message via NATS
func NatsPublishFromEngine(subject string, data map[string]any) {
	if !NatsIsConnected() {
		// silent, but perhaps you could log it every once in a while
		LogError(fmt.Errorf("NatsPublishFromEngine: called when NATS is not Connected"))
		return
	}
	fullsubject := fmt.Sprintf("from_palette.%s.%s", Hostname(), subject)
	err := NatsPublish(fullsubject, data)
	LogIfError(err)
}

func NatsDisconnect() {
	if nc := takeNatsConnection(); nc != nil {
		nc.Close()
	}
}

func NatsEnvValue(key string) (string, error) {
	// Prefer the env file (.palette/.env), falling back to the OS environment
	// variable. NATS_URL also accepts NATS_HUB_CLIENT_URL as an alias.
	s := EnvLookup(key)
	if s == "" && key == "NATS_URL" {
		s = EnvLookup("NATS_HUB_CLIENT_URL")
	}
	if s == "" {
		return "", fmt.Errorf("no %s value, use 'palette env set' to set", key)
	}
	return s, nil
}

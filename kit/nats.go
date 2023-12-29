package kit

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/nats-io/nats.go"
)

// VizNats xxx
type VizNats struct {
	natsConn    *nats.Conn
	enabled     bool
	isConnected bool
	doReconnect bool
	attempts    int
}

// TheNats is the only one
var TheNats *VizNats

var time0 = time.Now()

// PublishFromEngine sends an asynchronous message via NATS
func PublishFromEngine(subject string, msg string) {
	if !TheNats.enabled {
		// silent, but perhaps you could log it every once in a while
		return
	}
	fullsubject := fmt.Sprintf("from_palette.%s.%s", Hostname(), subject)
	err := TheNats.Publish(fullsubject, msg)
	LogIfError(err)
}

// PublishCursorEvent xxx
func PublishCursorEvent(ce CursorEvent) {
	if !TheNats.enabled {
		return // silent
	}
	dt := time.Since(time0)
	regionvalue := ""
	if ce.Tag != "" {
		regionvalue = "\"tag\": \"" + ce.Tag + "\", "
	}
	params := "{ " +
		// "\"tag\": \"" + ce.Tag + "\", " +
		"\"cid\": \"" + fmt.Sprintf("%d", ce.Gid) + "\", " +
		regionvalue +
		"\"ddu\": \"" + ce.Ddu + "\", " +
		"\"millisecs\": \"" + fmt.Sprintf("%d", dt.Milliseconds()) + "\", " +
		"\"x\": \"" + fmt.Sprintf("%f", ce.Pos.X) + "\", " +
		"\"y\": \"" + fmt.Sprintf("%f", ce.Pos.Y) + "\", " +
		"\"z\": \"" + fmt.Sprintf("%f", ce.Pos.Z) + "\", " +
		"\"area\": \"" + fmt.Sprintf("%f", ce.Area) + "\" }"

	PublishFromEngine("event.cursor", params)
}

// PublishMIDIDeviceEvent xxx
func PublishMIDIDeviceEvent(me MidiEvent) {
	if !TheNats.enabled {
		return // silent
	}
	dt := time.Since(time0)
	// NOTE: we ignore the Timestamp on the MIDIDeviceEvent
	// and use our own, so the timestamps are consistent with
	// the ones on Cursor events
	params := "{ " +
		"\"nuid\": \"" + MyNUID() + "\", " +
		"\"event\": \"" + "midi" + "\", " +
		// "\"timestamp\": \"" + fmt.Sprintf("%d", me.Timestamp) + "\", " +
		"\"millisecs\": \"" + fmt.Sprintf("%d", dt.Milliseconds()) + "\", " +
		"\"bytes\": \"" + fmt.Sprintf("%v", me.Msg.Bytes()) + "\" }"

	PublishFromEngine("event.midi", params)
}

// PublishSpriteEvent xxx
func PublishSpriteEvent(x, y, z float32) {
	if !TheNats.enabled {
		return // silent
	}
	params := "{ " +
		"\"nuid\": \"" + MyNUID() + "\", " +
		"\"x\": \"" + fmt.Sprintf("%f", x) + "\", " +
		"\"y\": \"" + fmt.Sprintf("%f", y) + "\", " +
		"\"z\": \"" + fmt.Sprintf("%f", z) + "\" }"

	PublishFromEngine("event.sprite", params)
}

// NewNats xxx
func NewNats() *VizNats {
	return &VizNats{
		natsConn:    nil,
		enabled:     true,
		isConnected: false,
		doReconnect: false,
		attempts:    0,
	}
}

func (vn *VizNats) Disconnect() {
	vn.natsConn.Close()
	vn.natsConn = nil
}

// Connect xxx
func (vn *VizNats) Connect() error {

	if !TheNats.enabled {
		return fmt.Errorf("VizNats.Connect: called when NATS not enabled")
	}
	if vn.natsConn != nil {
		// Already connected
		LogInfo("VisNats.Connect: Already connected!")
		return nil
	}
	if !vn.doReconnect && vn.attempts > 0 {
		err := fmt.Errorf("VisNats.Connect: doReconnect is false, not attempting another connect")
		LogError(err)
		return err
	}
	vn.attempts++
	LogInfo("VisNats.Connect: about to try to connect", "attempt", vn.attempts)
	user := os.Getenv("NATS_USER")
	password := os.Getenv("NATS_PASSWORD")
	url := os.Getenv("NATS_URL")
	if url == "" {
		url = LocalAddress
	}
	fullurl := fmt.Sprintf("%s:%s@%s", user, password, url)

	// Connect Options.
	opts := []nats.Option{nats.Name("Palette hostwin Subscriber")}
	opts = setupConnOptions(opts)

	// Use UserCredentials
	var userCreds = "" // User Credentials File
	if userCreds != "" {
		opts = append(opts, nats.UserCredentials(userCreds))
	}

	reconnects := -1 // Keep reconnecting forever
	if ! vn.doReconnect {
		LogInfo("VizNats.Connect: will not attempt to reconnect")
		reconnects = 0
	}
	opts = append(opts, nats.MaxReconnects(reconnects))

	// Connect to NATS
	nc, err := nats.Connect(fullurl, opts...)
	if err != nil {
		vn.natsConn = nil
		return fmt.Errorf("nats.Connect failed, user=%s url=%s err=%s", user, url, err)
	}
	vn.natsConn = nc
	LogInfo("Successful connect to NATS")

	date := time.Now().Format(PaletteTimeLayout)
	msg := fmt.Sprintf("Successful connection from hostname=%s date=%s", Hostname(), date)
	PublishFromEngine("connect.info", msg)
	return nil
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
func (vn *VizNats) Request(subj, data string, timeout time.Duration) (retdata string, err error) {
	if !TheNats.enabled {
		return "", fmt.Errorf("VizNats.Request: called when NATS not enabled")
	}
	LogOfType("nats", "VizNats.Request", "subject", subj, "data", data)
	nc := vn.natsConn
	if nc == nil {
		return "", fmt.Errorf("Viznats.Request: no NATS connection")
	}
	bytes := []byte(data)
	msg, err := nc.Request(subj, bytes, timeout)
	if err == nats.ErrTimeout {
		return "", fmt.Errorf("timeout, nothing is subscribed to subj=%s", subj)
	} else if err != nil {
		return "", fmt.Errorf("error: subj=%s err=%s", subj, err)
	}
	return string(msg.Data), nil
}

// Publish xxx
func (vn *VizNats) Publish(subj string, msg string) error {

	if !TheNats.enabled {
		return fmt.Errorf("VizNats.Publish: called when NATS not enabled")
	}

	nc := vn.natsConn
	if nc == nil {
		return fmt.Errorf("Viznats.Publish: no NATS connection, subject=%s", subj)
	}
	bytes := []byte(msg)

	LogInfo("Nats.Publish", "subject", subj, "msg", msg)

	err := nc.Publish(subj, bytes)
	LogIfError(err)
	nc.Flush()

	if err := nc.LastError(); err != nil {
		return err
	}
	return nil
}

// Subscribe xxx
func (vn *VizNats) Subscribe(subj string, callback nats.MsgHandler) error {

	if !TheNats.enabled {
		return fmt.Errorf("VizNats.Subscribe: called when NATS not enabled")
	}

	LogInfo("VizNats.Subscribe", "subject", subj)

	nc := vn.natsConn
	if nc == nil {
		return fmt.Errorf("VizNats.Subscribe: subject=%s, no connection to NATS server", subj)
	}
	_, err := nc.Subscribe(subj, callback)
	LogIfError(err)
	nc.Flush()

	return nc.LastError()
}

func (vn *VizNats) Close() {
	if !vn.enabled {
		LogError(fmt.Errorf("VisNats.Close: called with NATS not enabled"))
		return
	}
	if vn.doReconnect && vn.attempts > 0 {
		LogInfo("VizNats.CLose called, should NOT attempt another connection, NATS is being disabled")
		vn.enabled = false
		return
	}
	if vn.natsConn != nil {
		vn.natsConn.Close()
		vn.natsConn = nil
		LogError(fmt.Errorf("VizNats.CLose called"))
	} else {
		LogError(fmt.Errorf("VizNats.CLose called when natsConn is nil"))
	}
}

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
	bytes, err := GetConfigFileData("nuid.json")
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
			LogWarn("InitLogs: Unable to open", "nuidpath", nuidpath, "err", err)
			return "UnableToOpenNUIDFile"
		}
		nuid := nuid.Next()
		file.WriteString("{\n\t\"nuid\": \"" + nuid + "\"\n}\n")
		file.Close()
		LogInfo("GetNUID: generated nuid.json", "nuid", nuid)
		return nuid
	*/
}

func setupConnOptions(opts []nats.Option) []nats.Option {
	totalWait := 10 * time.Minute
	reconnectDelay := time.Second

	opts = append(opts, nats.ReconnectWait(reconnectDelay))
	opts = append(opts, nats.MaxReconnects(int(totalWait/reconnectDelay)))
	opts = append(opts, nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
		TheNats.natsConn = nil
		TheNats.enabled = TheNats.doReconnect
		LogWarn("nats.Disconnected",
			"err", err,
			"waitminutes", totalWait.Minutes(),
			"doReconnect", TheNats.doReconnect)
	}))
	opts = append(opts, nats.ReconnectHandler(func(nc *nats.Conn) {
		LogWarn("nats.Reconnected", "connecturl", nc.ConnectedUrl())
	}))
	opts = append(opts, nats.ClosedHandler(func(nc *nats.Conn) {
		TheNats.natsConn = nil
		TheNats.enabled = TheNats.doReconnect
		LogWarn("nats.ClosedHandler",
			"lasterror", nc.LastError(),
			"doReconnect", TheNats.doReconnect)

	}))
	return opts
}

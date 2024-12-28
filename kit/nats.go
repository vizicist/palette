package kit

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/nats-io/nats.go"
	// "github.com/nats-io/nats-server/v2/server"
)

// VizNats xxx
type VizNats struct {
	natsConn    *nats.Conn
	enabled     bool
	isConnected bool
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
		enabled:     false,
		isConnected: false,
	}
}

func (vn *VizNats) Disconnect() {
	vn.natsConn.Close()
	vn.isConnected = false
}

// Init xxx
func (vn *VizNats) Init() error {

	if !TheNats.enabled {
		return fmt.Errorf("VizNats.Init: called when NATS not enabled")
	}

	LogInfo("VizNats.Init", "natsConn", vn.natsConn, "isconnected", vn.isConnected, "enabled", vn.enabled)

	if vn.isConnected {
		// Already connected
		LogInfo("VisNats.Init: Already connected!")
		return nil
	}

	user := os.Getenv("NATS_USER")
	password := os.Getenv("NATS_PASSWORD")
	url := os.Getenv("NATS_URL")

	if url == "" {
		url = LocalAddress
	}
	fullurl := fmt.Sprintf("%s:%s@%s", user, password, url)

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
	nc, err := nats.Connect(fullurl, opts...)
	if err != nil {
		vn.isConnected = false
		return fmt.Errorf("nats.Connect failed, user=%s url=%s err=%s", user, url, err)
	}
	vn.isConnected = true
	vn.natsConn = nc

	LogInfo("Successful connect to NATS","user",user,"url",url)

	date := time.Now().Format(PaletteTimeLayout)
	msg := fmt.Sprintf("Successful connection from hostname=%s date=%s", Hostname(), date)
	PublishFromEngine("connect.info", msg)

	subscribeTo := fmt.Sprintf("to_palette.%s.>", Hostname())
	LogInfo("Subscribing to NATS", "subscribeTo", subscribeTo)
	err = TheNats.Subscribe(subscribeTo, natsRequestHandler)
	LogIfError(err)

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
	if !TheNats.isConnected {
		return "", fmt.Errorf("VizNats.Request: called when NATS is not Connected")
	}

	LogOfType("nats", "VizNats.Request", "subject", subj, "data", data)
	if !vn.isConnected || vn.natsConn == nil {
		return "", fmt.Errorf("Viznats.Request: no NATS connection")
	}
	bytes := []byte(data)
	msg, err := vn.natsConn.Request(subj, bytes, timeout)
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
	if !TheNats.isConnected {
		return fmt.Errorf("VizNats.Publish: called when NATS is not Connected")
	}

	nc := vn.natsConn
	if !vn.isConnected || nc == nil {
		return fmt.Errorf("Viznats.Publish: no NATS connection, subject=%s", subj)
	}
	bytes := []byte(msg)

	LogInfo("Nats.Publish", "subject", subj, "msg", msg)

	err := vn.natsConn.Publish(subj, bytes)
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
	if !TheNats.isConnected {
		return fmt.Errorf("VizNats.Subscribe: called when NATS is not Connected")
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
		LogError(fmt.Errorf("VizNats.Close: called with NATS not enabled"))
		return
	}
	if vn.natsConn == nil || !vn.isConnected {
		LogError(fmt.Errorf("VizNats.Close called when natsConn is nil or unconnected"))
		return
	}
	vn.natsConn.Close()
	vn.isConnected = false
	LogError(fmt.Errorf("VizNats.CLose called"))
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
	totalWait := 48 * time.Hour
	reconnectDelay := 5 * time.Second

	opts = append(opts, nats.ReconnectWait(reconnectDelay))
	opts = append(opts, nats.MaxReconnects(int(totalWait/reconnectDelay)))
	opts = append(opts, nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
		TheNats.isConnected = false
		LogWarn("nats.Disconnected",
			"err", err,
			"waitminutes", totalWait.Minutes())
	}))
	opts = append(opts, nats.ReconnectHandler(func(nc *nats.Conn) {
		TheNats.isConnected = true
		LogWarn("nats.Reconnected", "connecturl", nc.ConnectedUrl())
	}))
	opts = append(opts, nats.ClosedHandler(func(nc *nats.Conn) {
		TheNats.isConnected = false
		LogWarn("nats.ClosedHandler",
			"lasterror", nc.LastError())

	}))
	return opts
}

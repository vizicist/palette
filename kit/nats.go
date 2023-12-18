package kit

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/nats-io/nats.go"
)

// VizNats xxx
type VizNats struct {
	natsConn *nats.Conn
}

// TheNats is the only one
var TheNats *VizNats

var time0 = time.Now()

func NatsApi(cmd string) (result string, err error) {
	err = TheNats.Connect()
	if err != nil {
		LogIfError(err)
		return "", err
	}
	timeout := 3 * time.Second
	retdata, err := TheNats.Request("to_palette.api", cmd, timeout)
	LogIfError(err)
	return retdata, err
}

// Publish sends an asynchronous message via NATS
func Publish(subject string, msg string) {
	err := TheNats.Publish(subject, msg)
	LogIfError(err)
}

// PublishCursorEvent xxx
func PublishCursorEvent(ce CursorEvent) {
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

	subject := "fromengine.event.cursor"
	Publish(subject, params)
}

// PublishMIDIDeviceEvent xxx
func PublishMIDIDeviceEvent(me MidiEvent) {
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

	subject := "fromengine.event.midi"
	Publish(subject, params)
}

// PublishSpriteEvent xxx
func PublishSpriteEvent(x, y, z float32) {
	params := "{ " +
		"\"nuid\": \"" + MyNUID() + "\", " +
		"\"x\": \"" + fmt.Sprintf("%f", x) + "\", " +
		"\"y\": \"" + fmt.Sprintf("%f", y) + "\", " +
		"\"z\": \"" + fmt.Sprintf("%f", z) + "\" }"

	subject := "fromengine.event.sprite"
	log.Printf("Publishing %s %s\n", subject, params)

	Publish(subject, params)
}

/*
// StartVizNats xxx
func StartVizNats() {
	err := TheNats.Connect()
	if err != nil {
		log.Printf("VizNats.Connect: err=%s\n", err)
		TheNats.natsConn = nil
	}
}
*/

// NewVizNats xxx
func NewNats() *VizNats {
	return &VizNats{
		natsConn: nil,
	}
}

func (vn *VizNats) Disconnect() {
	vn.natsConn.Close()
	vn.natsConn = nil
}

// Connect xxx
func (vn *VizNats) Connect() error {

	if vn.natsConn != nil {
		// Already connected
		return nil
	}
	user := os.Getenv("NATS_USER")
	password := os.Getenv("NATS_PASSWORD")
	url := os.Getenv("NATS_URL")
	if url == "" {
		url = LocalAddress
	}
	fullurl := fmt.Sprintf("%s:%s@%s", user, password, url)

	var userCreds = "" // User Credentials File

	// Connect Options.
	opts := []nats.Option{nats.Name("Palette hostwin Subscriber")}
	opts = setupConnOptions(opts)

	// Use UserCredentials
	if userCreds != "" {
		opts = append(opts, nats.UserCredentials(userCreds))
	}

	// Keep reconnecting forever
	opts = append(opts, nats.MaxReconnects(-1))

	// Connect to NATS
	nc, err := nats.Connect(fullurl, opts...)
	if err != nil {
		return fmt.Errorf("nats.Connect failed, user=%s err=%s", user, err)
	}
	vn.natsConn = nc
	LogInfo("Successful connect to NATS")

	msg := fmt.Sprintf("Successful connection from hostname = %s", Hostname())
	return vn.Publish("palette.info", msg)
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
	if IsLogging("nats") {
		log.Printf("VizNats.Request: %s %s\n", subj, data)
	}
	nc := vn.natsConn
	if nc == nil {
		return "", fmt.Errorf("unable to communicate with NATS")
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

	nc := vn.natsConn
	if nc == nil {
		return fmt.Errorf("Publish: subject=%s, no connection to nats-server", subj)
	}
	bytes := []byte(msg)

	if IsLogging("nats") {
		log.Printf("Nats.Publish: %s %s\n", subj, msg)
	}

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

	// if IsLogging("nats") {
	LogInfo("VizNats.Subscribe: %s\n", subj)
	// }

	nc := vn.natsConn
	if nc == nil {
		return fmt.Errorf("Subscribe: subject=%s, no connection to nats-server", subj)
	}
	_, err := nc.Subscribe(subj, callback)
	LogIfError(err)
	nc.Flush()

	return nc.LastError()
}

func (vn *VizNats) Close() {
	if vn.natsConn != nil {
		vn.natsConn.Close()
		vn.natsConn = nil
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

func setupConnOptions(opts []nats.Option) []nats.Option {
	totalWait := 10 * time.Minute
	reconnectDelay := time.Second

	opts = append(opts, nats.ReconnectWait(reconnectDelay))
	opts = append(opts, nats.MaxReconnects(int(totalWait/reconnectDelay)))
	opts = append(opts, nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
		log.Printf("Disconnected due to:%s, will attempt reconnects for %.0fm", err, totalWait.Minutes())
	}))
	opts = append(opts, nats.ReconnectHandler(func(nc *nats.Conn) {
		log.Printf("Reconnected [%s]", nc.ConnectedUrl())
	}))
	opts = append(opts, nats.ClosedHandler(func(nc *nats.Conn) {
		log.Printf("nats.ClosedHandler, Exiting: %v", nc.LastError())
		TheNats.natsConn = nil
	}))
	return opts
}

func handleDiscover(msg *nats.Msg) {
	response := MyNUID()
	if IsLogging("api") {
		log.Printf("handleDiscover: data=%s reply=%s response=%s\n", string(msg.Data), msg.Reply, response)
	}
	err := msg.Respond([]byte(response))
	LogIfError(err)
}

var _ = handleDiscover // to avoid unused error from go-staticcheck

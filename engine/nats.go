package engine

import (
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

// PaletteAPISubject xxx
var PaletteAPISubject = "palette.api"

// PaletteEventSubject xxx
var PaletteEventSubject = "palette.event"

var time0 = time.Now()

// PublishCursorDeviceEvent xxx
func PublishCursorDeviceEvent(ce CursorDeviceEvent) error {
	dt := time.Since(time0)
	regionvalue := ""
	if ce.Region != "" {
		regionvalue = "\"region\": \"" + ce.Region + "\", "
	}
	event := "cursor_" + ce.Ddu
	params := "{ " +
		"\"nuid\": \"" + ce.NUID + "\", " +
		"\"cid\": \"" + ce.CID + "\", " +
		regionvalue +
		"\"event\": \"" + event + "\", " +
		"\"millisecs\": \"" + fmt.Sprintf("%d", dt.Milliseconds()) + "\", " +
		"\"x\": \"" + fmt.Sprintf("%f", ce.X) + "\", " +
		"\"y\": \"" + fmt.Sprintf("%f", ce.Y) + "\", " +
		"\"z\": \"" + fmt.Sprintf("%f", ce.Z) + "\", " +
		"\"area\": \"" + fmt.Sprintf("%f", ce.Area) + "\" }"

	err := theVizNats.Publish(PaletteEventSubject, params)
	if err != nil {
		return err
	}
	return nil
}

// PublishMIDIDeviceEvent xxx
func PublishMIDIDeviceEvent(me MIDIDeviceEvent) error {
	dt := time.Since(time0)
	// NOTE: we ignore the Timestamp on the MIDIDeviceEvent
	// and use our own, so the timestamps are consistent with
	// the ones on Cursor events
	params := "{ " +
		"\"nuid\": \"" + MyNUID() + "\", " +
		"\"event\": \"" + "midi" + "\", " +
		// "\"timestamp\": \"" + fmt.Sprintf("%d", me.Timestamp) + "\", " +
		"\"millisecs\": \"" + fmt.Sprintf("%d", dt.Milliseconds()) + "\", " +
		"\"status\": \"" + fmt.Sprintf("%d", me.Status) + "\", " +
		"\"data1\": \"" + fmt.Sprintf("%d", me.Data1) + "\", " +
		"\"data2\": \"" + fmt.Sprintf("%d", me.Data2) + "\" }"

	err := theVizNats.Publish(PaletteEventSubject, params)
	if err != nil {
		return err
	}
	return nil
}

// PublishSpriteEvent xxx
func PublishSpriteEvent(x, y, z float32) error {
	params := "{ " +
		"\"nuid\": \"" + MyNUID() + "\", " +
		"\"x\": \"" + fmt.Sprintf("%f", x) + "\", " +
		"\"y\": \"" + fmt.Sprintf("%f", y) + "\", " +
		"\"z\": \"" + fmt.Sprintf("%f", z) + "\" }"

	err := theVizNats.Publish(PaletteEventSubject, params)
	if err != nil {
		return err
	}
	return nil
}

// PublishAliveEvent xxx
func PublishAliveEvent(secs float64, cursorCount int) error {
	params := "{ " +
		"\"nuid\": \"" + MyNUID() + "\", " +
		"\"event\": \"" + "alive" + "\", " +
		"\"seconds\": \"" + fmt.Sprintf("%f", secs) + "\", " +
		"\"cursorcount\": \"" + fmt.Sprintf("%d", cursorCount) +
		"\" }"

	err := theVizNats.Publish(PaletteEventSubject, params)
	if err != nil {
		return err
	}
	return nil
}

// EngineAPI xxx
func EngineAPI(api, params string) (retmap map[string]string, err error) {
	timeout := 3 * time.Second
	args := "{ " +
		"\"nuid\": \"" + MyNUID() + "\", " +
		"\"api\": \"" + api + "\", " +
		"\"params\": \"" + jsonEscape(params) +
		"\" }"
	result, err := theVizNats.Request("palette.api", args, timeout)
	if err != nil {
		log.Printf("EngineAPI: protocol error err=%s\n", err)
		return nil, err
	}
	retmap, err = StringMap(result)
	if err != nil {
		log.Printf("EngineAPI: protocol stringmap err=%s\n", err)
		return nil, err
	}
	apierr, hasapierr := retmap["error"]
	if hasapierr {
		log.Printf("EngineAPI: api error err=%s\n", apierr)
		return nil, fmt.Errorf("%s", apierr)
	}
	return retmap, err
}

// VizNats xxx
type VizNats struct {
	natsConn *nats.Conn
}

// TheVizNats is the only one
var theVizNats *VizNats

// InitVizNats xxx
func InitNATS() {
	if theVizNats != nil {
		log.Printf("InitNATS: called twice!?\n")
		return
	}
	theVizNats = NewVizNats()
}

func ConnectToNATSServer() {
	err := theVizNats.Connect()
	if err != nil {
		log.Printf("VizNats.Connect: err=%s\n", err)
		theVizNats.natsConn = nil
	}
}

// NewVizNats xxx
func NewVizNats() *VizNats {
	return &VizNats{
		natsConn: nil,
	}
}

// Connect xxx
func (vn *VizNats) Connect() error {

	var urls = nats.DefaultURL // The nats server URLs (separated by comma)
	var userCreds = ""         // User Credentials File

	// Connect Options.
	opts := []nats.Option{nats.Name("VizNat Subscriber")}
	opts = setupConnOptions(opts)

	// Use UserCredentials
	if userCreds != "" {
		opts = append(opts, nats.UserCredentials(userCreds))
	}

	// Keep reconnecting forever
	opts = append(opts, nats.MaxReconnects(-1))

	// Connect to NATS
	nc, err := nats.Connect(urls, opts...)
	if err == nil {
		vn.natsConn = nc
	}
	return err // nil or not
}

// Request is used for APIs - it blocks waiting for a response and returns the response
func (vn *VizNats) Request(subj, data string, timeout time.Duration) (retdata string, err error) {
	if Debug.NATS {
		log.Printf("VizNats.Request: %s %s\n", subj, data)
	}
	nc := vn.natsConn
	bytes := []byte(data)
	msg, err := nc.Request(subj, bytes, timeout)
	if err != nil {
		return "", fmt.Errorf("Request: subj=%s err=%s", subj, err)
	}
	return string(msg.Data), nil
}

// Publish xxx
func (vn *VizNats) Publish(subj string, msg string) error {

	if Debug.NATS {
		log.Printf("VizNats.Publish: %s %s\n", subj, msg)
	}

	nc := vn.natsConn
	if nc == nil {
		return fmt.Errorf("Publish: subject=%s, no connection to nats-server", subj)
	}
	bytes := []byte(msg)

	nc.Publish(subj, bytes)
	nc.Flush()

	if err := nc.LastError(); err != nil {
		return err
	}
	return nil
}

// Subscribe xxx
func SubscribeNATS(subj string, callback nats.MsgHandler) error {

	if Debug.NATS {
		log.Printf("VizNats.Subscribe: %s\n", subj)
	}
	nc := theVizNats.natsConn
	if nc == nil {
		return fmt.Errorf("SubscribeNATS: subject=%s, no connection to nats-server", subj)
	}
	nc.Subscribe(subj, callback)
	nc.Flush()

	if err := nc.LastError(); err != nil {
		return err
	}
	return nil
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
		theVizNats.natsConn = nil
	}))
	return opts
}

func handleDiscover(msg *nats.Msg) {
	response := MyNUID()
	if Debug.API {
		log.Printf("handleDiscover: data=%s reply=%s response=%s\n", string(msg.Data), msg.Reply, response)
	}
	msg.Respond([]byte(response))
}

var _ = handleDiscover // to avoid unused error from go-staticcheck

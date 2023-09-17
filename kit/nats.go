package kit

import (
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

// VizNats xxx
type VizNats struct {
	natsConn *nats.Conn
}

// TheNats is the only one
var TheNats *VizNats

// PaletteAPISubject xxx
var PaletteAPISubject = "palette.api"

// PaletteEventSubject xxx
var PaletteEventSubject = "palette.event"

var time0 = time.Now()

// PublishCursorDeviceEvent xxx
func PublishCursorDeviceEvent(ce CursorEvent) error {
	dt := time.Since(time0)
	regionvalue := ""
	if ce.Tag != "" {
		regionvalue = "\"tag\": \"" + ce.Tag + "\", "
	}
	event := "cursor_" + ce.Ddu
	params := "{ " +
		"\"tag\": \"" + ce.Tag + "\", " +
		"\"cid\": \"" + fmt.Sprintf("%d", ce.Gid) + "\", " +
		regionvalue +
		"\"event\": \"" + event + "\", " +
		"\"millisecs\": \"" + fmt.Sprintf("%d", dt.Milliseconds()) + "\", " +
		"\"x\": \"" + fmt.Sprintf("%f", ce.Pos.X) + "\", " +
		"\"y\": \"" + fmt.Sprintf("%f", ce.Pos.Y) + "\", " +
		"\"z\": \"" + fmt.Sprintf("%f", ce.Pos.Z) + "\", " +
		"\"area\": \"" + fmt.Sprintf("%f", ce.Area) + "\" }"

	if IsLogging("nats") {
		log.Printf("Publishing %s %s\n", PaletteEventSubject, params)
	}
	err := TheNats.Publish(PaletteEventSubject, params)
	if err != nil {
		return err
	}
	return nil
}

// PublishMIDIDeviceEvent xxx
func PublishMIDIDeviceEvent(me MidiEvent) error {
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

	if IsLogging("midi") {
		log.Printf("Publishing %s %s\n", PaletteEventSubject, params)
	}
	err := TheNats.Publish(PaletteEventSubject, params)
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

	log.Printf("Publishing %s %s\n", PaletteEventSubject, params)

	err := TheNats.Publish(PaletteEventSubject, params)
	if err != nil {
		return err
	}
	return nil
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
func NewVizNats() *VizNats {
	return &VizNats{
		natsConn: nil,
	}
}

// Connect xxx
func (vn *VizNats) Connect(user string, password string) error {

	fullurl := fmt.Sprintf("%s:%s@timthompson.com",user,password)

	// var urls = nats.DefaultURL // The nats server URLs (separated by comma)
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
	nc, err := nats.Connect(fullurl, opts...)
	if err != nil {
		return fmt.Errorf("nats.Connect failed, user=%s err=%s",user,err)
	} else {
		LogInfo("nats.Connect succeeded","user",user)
		vn.natsConn = nc
	}
	return err // nil or not
}

// Request is used for APIs - it blocks waiting for a response and returns the response
func (vn *VizNats) Request(subj, data string, timeout time.Duration) (retdata string, err error) {
	if IsLogging("nats") {
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

	if IsLogging("nats") {
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
func (vn *VizNats) Subscribe(subj string, callback nats.MsgHandler) error {

	if IsLogging("nats") {
		log.Printf("VizNats.Subscribe: %s\n", subj)
	}
	nc := vn.natsConn
	if nc == nil {
		return fmt.Errorf("Subscribe: subject=%s, no connection to nats-server", subj)
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
		TheNats.natsConn = nil
	}))
	return opts
}

func handleDiscover(msg *nats.Msg) {
	response := MyNUID()
	if IsLogging("api") {
		log.Printf("handleDiscover: data=%s reply=%s response=%s\n", string(msg.Data), msg.Reply, response)
	}
	msg.Respond([]byte(response))
}

var _ = handleDiscover // to avoid unused error from go-staticcheck

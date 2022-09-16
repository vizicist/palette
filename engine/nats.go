package engine

import (
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

// PaletteAPISubject xxx
var PaletteAPISubject = "palette.api"

// PaletteOutputEventSubject xxx
var PaletteOutputEventSubject = "palette.output.event"

// PaletteInputEventSubject xxx
var PaletteInputEventSubject = "palette.input.event"

// PaletteNote messages are sent from the engine to plugins for output notes.
// and also from plugins to the engine in order to play them (somehow avoiding recursion)
var PaletteNoteSubject = "palette.note"

var time0 = time.Now()

type paletteNATS struct {
	natsConn *nats.Conn
}

var natsSingleton *paletteNATS

// NATS xxx
func NATS() *paletteNATS {
	if natsSingleton != nil {
		return natsSingleton
	}
	natsSingleton = &paletteNATS{
		natsConn: nil,
	}
	err := natsSingleton.Connect()
	if err != nil {
		log.Printf("NATS: err=%s\n", err)
		natsSingleton.natsConn = nil
	}
	return natsSingleton
}

// PublishCursorDeviceEvent xxx
func PublishCursorDeviceEvent(subj string, ce CursorDeviceEvent) error {
	dt := time.Since(time0)
	params := JsonObject(
		// "nuid", ce.NUID,
		"source", ce.Source,
		"region", ce.Region,
		"event", "cursor_"+ce.Ddu,
		"millisecs", fmt.Sprintf("%d", dt.Milliseconds()),
		"x", fmt.Sprintf("%f", ce.X),
		"y", fmt.Sprintf("%f", ce.Y),
		"z", fmt.Sprintf("%f", ce.Z),
		"area", fmt.Sprintf("%f", ce.Area),
	)

	err := NATSPublish(subj, params)
	if err != nil {
		return err
	}
	return nil
}

// PublishMIDIDeviceEvent xxx
func PublishMIDIDeviceEvent(subj string, me MIDIDeviceEvent) error {
	dt := time.Since(time0)
	// NOTE: we ignore the Timestamp on the MIDIDeviceEvent
	// and use our own, so the timestamps are consistent with
	// the ones on Cursor events
	params := JsonObject(
		// "nuid", MyNUID(),
		"event", "midi",
		// "timestamp", fmt.Sprintf("%d", me.Timestamp),
		"millisecs", fmt.Sprintf("%d", dt.Milliseconds()),
		"status", fmt.Sprintf("%d", me.Status),
		"data1", fmt.Sprintf("%d", me.Data1),
		"data2", fmt.Sprintf("%d", me.Data2),
	)

	err := NATSPublish(subj, params)
	if err != nil {
		return err
	}
	return nil
}

// PublishSpriteEvent xxx
func PublishSpriteEvent(subj string, x, y, z float32) error {
	params := JsonObject(
		// "nuid", MyNUID(),
		"event", "sprite",
		"x", fmt.Sprintf("%f", x),
		"y", fmt.Sprintf("%f", y),
		"z", fmt.Sprintf("%f", z),
	)

	err := NATSPublish(subj, params)
	if err != nil {
		return err
	}
	return nil
}

// PublishAliveEvent xxx
func PublishAliveEvent(subj string, secs float64, cursorCount int) error {
	params := JsonObject(
		// "nuid", MyNUID(),
		"event", "alive",
		"seconds", fmt.Sprintf("%f", secs),
		"cursorcount", fmt.Sprintf("%d", cursorCount),
	)
	err := NATSPublish(subj, params)
	if err != nil {
		return err
	}
	return nil
}

// PublishNoteEvent xxx
func PublishNoteEvent(subj string, note *Note, source string) error {
	params := JsonObject(
		"source", source,
		"event", "note",
		"note", jsonEscape(note.String()),
		"synth", note.Sound,
		"clicks", fmt.Sprintf("%d", CurrentClick()),
		// "clicks", fmt.Sprintf("%d", note.Clicks),
	)
	err := NATSPublish(subj, params)
	if err != nil {
		return err
	}
	return nil
}

// EngineAPI result is json with either a "result" or "error" value.
// The err return value is an internal error, not from the API.
func EngineAPI(api, params string) (result string, err error) {
	// Long timeout to better handle engine debugging
	timeout := 60 * time.Second
	args := JsonObject(
		// "nuid", MyNUID(),
		"api", api,
		"params", jsonEscape(params),
	)
	return NATSRequest(PaletteAPISubject, args, timeout)
}

// Connect xxx
func (vn *paletteNATS) Connect() error {

	var urls = nats.DefaultURL // The nats server URLs (separated by comma)
	var userCreds = ""         // User Credentials File

	// Connect Options.
	opts := []nats.Option{nats.Name("Palette Subscriber")}
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
func NATSRequest(subj, data string, timeout time.Duration) (retdata string, err error) {
	vn := NATS()
	if Debug.NATS {
		log.Printf("NATS.Request: %s %s\n", subj, data)
	}
	nc := vn.natsConn
	bytes := []byte(data)
	msg, err := nc.Request(subj, bytes, timeout)
	if err != nil {
		if err == nats.ErrInvalidConnection {
			err = fmt.Errorf("palette engine is either not responding or running")
		} else {
			err = fmt.Errorf("request: subj=%s err=%s", subj, err)
		}
		return "", err
	}
	return string(msg.Data), nil
}

// NATSPublish xxx
func NATSPublish(subj string, msg string) error {

	vn := NATS()
	if Debug.NATS {
		log.Printf("NATSPublish: %s %s\n", subj, msg)
	}

	nc := vn.natsConn
	if nc == nil {
		return fmt.Errorf("NATSPublish: subject=%s, no connection to nats-server", subj)
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
		log.Printf("NATS.Subscribe: %s\n", subj)
	}
	nc := NATS().natsConn
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
		NATS().natsConn = nil
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

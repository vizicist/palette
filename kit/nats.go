package kit

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

// VizNats xxx
type VizNats struct {
	natsConn   *nats.Conn
	natsServer *server.Server
}

// TheNats is the only one
var TheNats *VizNats

// PaletteAPISubject xxx
var PaletteAPISubject = "palette.api"

var time0 = time.Now()

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
func NewVizNats() *VizNats {
	return &VizNats{
		natsConn: nil,
	}
}

// Connect xxx
func (vn *VizNats) Connect(user string, password string, url string) error {

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
	} else {
		vn.natsConn = nc
		err = vn.Publish("palette.info", "nats.Connect has succeeded")
		LogIfError(err)
	}
	return err // nil or not
}

// Request is used for APIs - it blocks waiting for a response and returns the response
func (vn *VizNats) Request(subj, data string, timeout time.Duration) (retdata string, err error) {
	if IsLogging("nats") {
		log.Printf("VizNats.Request: %s %s\n", subj, data)
	}
	nc := vn.natsConn
	if nc == nil {
		return "", fmt.Errorf("VizNats.Request: no natsConn")
	}
	bytes := []byte(data)
	msg, err := nc.Request(subj, bytes, timeout)
	if err != nil {
		return "", fmt.Errorf("Request: subj=%s err=%s", subj, err)
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

	return nc.LastError()
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

// StartNATSServer xxx
func (viznats *VizNats) StartServer() error{

	_ = MyNUID() // to make sure nuid.json is initialized

	fs := flag.NewFlagSet("VizNats.StartServer", flag.ExitOnError)

	natsconf, err := GetParam("engine.natsconf")
	if err != nil || natsconf == "" {
		natsconf = "natsalone.conf"
	}

	LogInfo("Starting NATS Server", "natsconf", natsconf)

	// Configure the options from the flags/config file
	conf := TheHost.ConfigFilePath(natsconf)
	args := []string{"-c", conf}

	opts, err := server.ConfigureOptions(fs, args,
		server.PrintServerAndExit,
		fs.Usage,
		server.PrintTLSHelpAndDie)
	if err != nil {
		return err
	}
	LogInfo("VizNats.StartServer config file is valid", "config", opts.ConfigFile)

	// Create the server with appropriate options.
	s, err := server.NewServer(opts)
	if err != nil {
		return err
	}

	// Configure the logger based on the flags
	s.ConfigureLogger()

	// Start the server up in the background
	if err := server.Run(s); err != nil {
		LogError(err)
	}
	viznats.natsServer = s
	return nil
}

func (viznats *VizNats) WaitForShutdown() {
	viznats.natsServer.WaitForShutdown()
}

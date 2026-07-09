package kit

import (
	"fmt"
	"sync"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
)

const (
	embeddedNATSHost          = "127.0.0.1"
	embeddedNATSPort          = 4222
	embeddedNATSWebsocketPort = 9222
)

var embeddedNATS struct {
	mutex  sync.Mutex
	server *natsserver.Server
}

func StartEmbeddedLocalNATSServer() error {
	embeddedNATS.mutex.Lock()
	defer embeddedNATS.mutex.Unlock()

	if embeddedNATS.server != nil && embeddedNATS.server.Running() {
		return nil
	}

	opts := &natsserver.Options{
		ServerName: fmt.Sprintf("palette-%s-local", Hostname()),
		Host:       embeddedNATSHost,
		Port:       embeddedNATSPort,
		NoSigs:     true,
		Websocket: natsserver.WebsocketOpts{
			Host:  embeddedNATSHost,
			Port:  embeddedNATSWebsocketPort,
			NoTLS: true,
			AllowedOrigins: []string{
				"http://127.0.0.1:3330",
				"http://localhost:3330",
			},
		},
	}

	server, err := natsserver.NewServer(opts)
	if err != nil {
		return fmt.Errorf("unable to create embedded NATS server: %w", err)
	}
	server.SetLogger(paletteNATSLogger{}, false, false)
	go server.Start()
	if !server.ReadyForConnections(5 * time.Second) {
		server.Shutdown()
		return fmt.Errorf("embedded NATS server did not become ready")
	}

	embeddedNATS.server = server
	LogInfo("Started embedded NATS server",
		"url", EmbeddedNATSURL(),
		"websocket", EmbeddedNATSWebsocketURL())
	return nil
}

type paletteNATSLogger struct{}

func (paletteNATSLogger) Noticef(format string, v ...any) {
	LogOfType("nats", fmt.Sprintf(format, v...))
}

func (paletteNATSLogger) Warnf(format string, v ...any) {
	LogWarn("embedded NATS: " + fmt.Sprintf(format, v...))
}

func (paletteNATSLogger) Fatalf(format string, v ...any) {
	LogError(fmt.Errorf("embedded NATS fatal: "+format, v...))
}

func (paletteNATSLogger) Errorf(format string, v ...any) {
	LogError(fmt.Errorf("embedded NATS: "+format, v...))
}

func (paletteNATSLogger) Debugf(format string, v ...any) {
	LogOfType("nats", fmt.Sprintf(format, v...))
}

func (paletteNATSLogger) Tracef(format string, v ...any) {
	LogOfType("nats", fmt.Sprintf(format, v...))
}

func StopEmbeddedNATSServer() {
	embeddedNATS.mutex.Lock()
	server := embeddedNATS.server
	embeddedNATS.server = nil
	embeddedNATS.mutex.Unlock()
	if server != nil {
		server.Shutdown()
		server.WaitForShutdown()
	}
}

func EmbeddedNATSRunning() bool {
	embeddedNATS.mutex.Lock()
	defer embeddedNATS.mutex.Unlock()
	return embeddedNATS.server != nil && embeddedNATS.server.Running()
}

func EmbeddedNATSURL() string {
	return fmt.Sprintf("nats://%s:%d", embeddedNATSHost, embeddedNATSPort)
}

func EmbeddedNATSWebsocketURL() string {
	return fmt.Sprintf("ws://%s:%d", embeddedNATSHost, embeddedNATSWebsocketPort)
}

package kit

import (
	"fmt"
	"net"
	"net/url"
	"sync"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
)

const (
	embeddedNATSHost          = "127.0.0.1"
	embeddedNATSPort          = 4222
	embeddedNATSWebsocketPort = 9222
	embeddedNATSLocalSubject  = "palette.local.>"
)

var embeddedNATS struct {
	mutex          sync.Mutex
	server         *natsserver.Server
	leafURL        string
	leafConfigured bool
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

	if leafURL, err := EmbeddedNATSLeafURL(); err == nil && leafURL != "" {
		remoteURL, parseErr := url.Parse(leafURL)
		if parseErr != nil {
			LogWarn("StartEmbeddedLocalNATSServer: unable to parse leaf remote URL", "err", parseErr, "url", maskURLPassword(leafURL))
		} else {
			opts.LeafNode.ReconnectInterval = 5 * time.Second
			opts.LeafNode.Remotes = []*natsserver.RemoteLeafOpts{
				{
					URLs:        []*url.URL{remoteURL},
					DenyImports: []string{embeddedNATSLocalSubject},
					DenyExports: []string{embeddedNATSLocalSubject},
				},
			}
			embeddedNATS.leafURL = leafURL
			embeddedNATS.leafConfigured = true
		}
	} else {
		LogWarn("StartEmbeddedLocalNATSServer: no NATS_URL configured; local NATS will run without a leaf remote", "err", err)
		embeddedNATS.leafURL = ""
		embeddedNATS.leafConfigured = false
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
		"websocket", EmbeddedNATSWebsocketURL(),
		"leafConfigured", embeddedNATS.leafConfigured,
		"leafURL", maskURLPassword(embeddedNATS.leafURL))
	return nil
}

func EmbeddedNATSLeafURL() (string, error) {
	if leafURL, err := NatsEnvValue("NATS_LEAF_URL"); err == nil && leafURL != "" {
		return leafURL, nil
	}
	hubURL, err := NatsEnvValue("NATS_URL")
	if err != nil {
		return "", err
	}
	return deriveNATSLeafURL(hubURL)
}

func deriveNATSLeafURL(hubURL string) (string, error) {
	u, err := url.Parse(hubURL)
	if err != nil {
		return "", err
	}
	hostname := u.Hostname()
	if hostname == "" {
		return "", fmt.Errorf("NATS_URL has no hostname")
	}
	u.Host = net.JoinHostPort(hostname, "7422")
	return u.String(), nil
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

func EmbeddedNATSLeafConfigured() bool {
	embeddedNATS.mutex.Lock()
	defer embeddedNATS.mutex.Unlock()
	return embeddedNATS.leafConfigured
}

func EmbeddedNATSLeafConnections() int {
	embeddedNATS.mutex.Lock()
	server := embeddedNATS.server
	embeddedNATS.mutex.Unlock()
	if server == nil {
		return 0
	}
	return server.NumLeafNodes()
}

func EmbeddedNATSURL() string {
	return fmt.Sprintf("nats://%s:%d", embeddedNATSHost, embeddedNATSPort)
}

func EmbeddedNATSWebsocketURL() string {
	return fmt.Sprintf("ws://%s:%d", embeddedNATSHost, embeddedNATSWebsocketPort)
}

package main

/*
 * This program monitors NATS traffic from Palettes
  * and writes it to a file whose name contains the date.
 */

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/nats-io/nats.go"
)

var PaletteTimeLayout = "2006-01-02-15-04-05"

func main() {

	user := os.Getenv("NATS_USER")
	password := os.Getenv("NATS_PASSWORD")
	url := os.Getenv("NATS_URL")
	if url == "" {
		log.Printf("No value for NATS_URL!?")
		os.Exit(1)
	}
	if password == "" {
		log.Printf("No value for NATS_PASSWORD!?")
		os.Exit(1)
	}
	if user == "" {
		log.Printf("No value for NATS_USER!?")
		os.Exit(1)
	}
	fullurl := fmt.Sprintf("%s:%s@%s", user, password, url)

	// Connect Options.
	opts := []nats.Option{nats.Name("Palette hostwin Subscriber")}
	opts = setupConnOptions(opts)

	// Connect to NATS
	nc, err := nats.Connect(fullurl, opts...)
	if err != nil {
		log.Printf("nats.Connect failed, user=%s url=%s err=%s", user, url, err)
		os.Exit(1)
	}

	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	// Create a file to write the traffic
	// file, err := os.Create("palette_nats_traffic.log")

	date := time.Now().Format(PaletteTimeLayout)
	filename := "nats_traffic_" + date + ".log"
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)

	msg := fmt.Sprintf("Started, writing to %s", filename)
	addToLog(file, "nats_traffic.log", msg)

	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Subscribe to all subjects with a wildcard
	sub, err := nc.Subscribe(">", func(m *nats.Msg) {
		// Write the subject and data of each message to the file
		addToLog(file, m.Subject, string(m.Data))
	})
	if err != nil {
		log.Fatal(err)
	}
	defer sub.Unsubscribe()

	// Wait for messages
	select {}
}

func addToLog(file *os.File, subject string, data string) {
	date := time.Now().Format(PaletteTimeLayout)
	line := fmt.Sprintf("%s ; %s ; %s\n", date, subject, data)

	// Print once to the file, and again to stdout
	fmt.Fprint(file, line)
	fmt.Print(line)
}

func setupConnOptions(opts []nats.Option) []nats.Option {

	totalWait := 48 * time.Hour // 2 days
	reconnectDelay := 10 * time.Second

	opts = append(opts, nats.ReconnectWait(reconnectDelay))
	opts = append(opts, nats.MaxReconnects(int(totalWait/reconnectDelay)))
	opts = append(opts, nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
		log.Printf("NATS Disconnect Err = " + err.Error() + "\n")
	}))
	opts = append(opts, nats.ReconnectHandler(func(nc *nats.Conn) {
		log.Printf("NATS Reconnected\n")
	}))
	opts = append(opts, nats.ClosedHandler(func(nc *nats.Conn) {
		log.Printf("nats ClosedHandler\n")
	}))
	return opts
}

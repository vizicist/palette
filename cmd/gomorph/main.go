// Command gomorph reads cursor events from Sensel Morph device(s) and sends
// them as OSC /cursor messages. It is a pure-Go (no LibSensel/cgo) reimplementation
// of github.com/vizicist/gomorph, so it runs on Windows, macOS and Raspberry Pi.
//
//	gomorph                 # open all Morphs, print cursors, send OSC to 127.0.0.1:4444
//	gomorph -list           # just list the Morphs found
//	gomorph -quiet          # send OSC but don't print cursor lines
//	gomorph -serial ABC123  # only use the Morph with this serial number
//	gomorph -listen         # act as an OSC server and print received /cursor messages
package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/hypebeast/go-osc/osc"
	"github.com/vizicist/palette/cmd/gomorph/morph"
)

// Client is the OSC client used to send cursor messages.
var Client *osc.Client

// Verbose enables extra logging.
var Verbose bool

// Quiet suppresses printing of cursor messages.
var Quiet bool

func main() {
	verbosePtr := flag.Bool("verbose", false, "Be Verbose")
	quietPtr := flag.Bool("quiet", false, "Don't print cursor messages")
	listPtr := flag.Bool("list", false, "List Morphs")
	listenPtr := flag.Bool("listen", false, "listen for OSC events")
	ipPtr := flag.String("ip", "127.0.0.1", "IP address to send/listen on")
	portPtr := flag.Int("port", 4444, "OSC UDP port to send/listen on")
	serialPtr := flag.String("serial", "*", "Morph serialnum to use")
	devicePtr := flag.String("device", "", "Open this serial device directly (e.g. COM7 or /dev/cu.usbmodemXXXX), bypassing enumeration")

	flag.Parse()

	Verbose = *verbosePtr
	Quiet = *quietPtr
	morph.DebugMorph = Verbose

	if *listenPtr {
		doListen(*ipPtr, *portPtr)
		return
	}

	fmt.Printf("MAIN: start\n")
	var morphs []morph.OneMorph
	var err error
	if *devicePtr != "" {
		morphs, err = morph.InitPort(*devicePtr)
	} else {
		morphs, err = morph.Init(*serialPtr)
	}
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}

	for _, m := range morphs {
		fmt.Printf("Opened: Morph idx=%d serial=%s firmware=%d.%d.%d width=%.1fmm height=%.1fmm\n",
			m.Idx, m.SerialNum, m.FwVersionMajor, m.FwVersionMinor, m.FwVersionBuild, m.Width, m.Height)
	}

	if *listPtr {
		return
	}

	Client = osc.NewClient(*ipPtr, *portPtr)
	if err := morph.Start(morphs, handleMorph, 1.0); err != nil {
		fmt.Printf("Error: %s\n", err)
	}
}

func handleMorph(e morph.CursorDeviceEvent) {
	if !Quiet {
		fmt.Printf("Morph: cursor %s %s %f %f %f\n", e.Ddu, e.CID, e.X, e.Y, e.Z)
	}
	msg := osc.NewMessage("/cursor")
	msg.Append(e.Ddu)
	msg.Append(e.CID)
	msg.Append(float32(e.X))
	msg.Append(float32(e.Y))
	msg.Append(float32(e.Z))
	if err := Client.Send(msg); err != nil && Verbose {
		fmt.Printf("OSC send error: %s\n", err)
	}
}

func doListen(ip string, port int) {
	addr := fmt.Sprintf("%s:%d", ip, port)
	server := &osc.Server{}
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		fmt.Println("Couldn't listen: ", err)
		return
	}
	defer conn.Close()

	fmt.Printf("Listening for OSC at %s\n", addr)
	fmt.Printf("Press \"q\" to exit\n")

	go func() {
		for {
			packet, err := server.ReceivePacket(conn)
			if err != nil {
				fmt.Println("Error in ReceivePacket: " + err.Error())
				os.Exit(1)
			}
			if packet == nil {
				continue
			}
			switch pkt := packet.(type) {
			default:
				fmt.Println("Unknown packet type!")
			case *osc.Message:
				fmt.Printf("OSC Message: ")
				osc.PrintMessage(pkt)
			case *osc.Bundle:
				fmt.Println("OSC Bundle:")
				for i, message := range pkt.Messages {
					fmt.Printf("OSC Bundle Message #%d: ", i+1)
					osc.PrintMessage(message)
				}
			}
		}
	}()

	reader := bufio.NewReader(os.Stdin)
	for {
		c, err := reader.ReadByte()
		if err != nil {
			os.Exit(0)
		}
		if c == 'q' {
			os.Exit(0)
		}
	}
}

package main

import (
	"fmt"
	"os"
	"strings"
	"flag"

	twitch "github.com/gempir/go-twitch-irc/v3"
	"github.com/vizicist/palette/engine"
)

func main() {

	engine.InitLog("chat")

	engine.LogInfo("CHAT START")

	flag.Parse()

	go StartTwitch()

	vals, err := engine.EngineRemoteApi("engine.status")
	if err != nil {
		engine.LogError(err)
	} else {
		engine.LogInfo("api output", "vals", vals)
	}

	select {}
}

func StartTwitch() {

	clientUserName := os.Getenv("TWITCH_USER")
	clientAuthenticationToken := os.Getenv("TWITCH_TOKEN")
	client := twitch.NewClient(clientUserName, clientAuthenticationToken)

	client.OnConnect(func() {
		engine.LogInfo("Twitch OnConnect","clientUserName",clientUserName)
		// client.Say("photonsalon", fmt.Sprintf("OnConnect user=%s", clientUserName))
	})
	client.OnWhisperMessage(func(message twitch.WhisperMessage) {
		engine.LogInfo("OnWhisperMessage")
	})
	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		// engine.LogInfo("OnPrivateMessage", "raw", message.Raw)
		msg := strings.ToLower(message.Message)
		id := message.Tags["id"]
		engine.LogInfo("OnPrivateMessage", "msg", msg)
		words := strings.Split(msg, " ")
		if len(words) == 0 {
			client.Reply("photonsalon", id, "No command given?")
		} else {
			switch words[0] {

			case "randomize":
				vals, err := engine.EngineRemoteApi("quadpro.loadrand")
				var reply string
				if err != nil {
					reply = fmt.Sprintf("err=%s", err.Error())
				} else {
					result := vals["result"]
					reply = fmt.Sprintf("Preset = %s", result)
				}
				client.Reply("photonsalon", id, reply)

			case "status":
				vals, err := engine.EngineRemoteApi("engine.status")
				var reply string
				if err != nil {
					reply = fmt.Sprintf("err=%s", err.Error())
				} else {
					reply = fmt.Sprintf("vals=%v", vals)
				}
				client.Reply("photonsalon", id, reply)
			case "ping":
				engine.LogInfo("ping message", "msg", msg)
			}
		}

	})
	client.OnClearChatMessage(func(message twitch.ClearChatMessage) {
		engine.LogInfo("OnClearChatMessage")
	})
	client.OnClearMessage(func(message twitch.ClearMessage) {
		engine.LogInfo("OnClearMessage")
	})
	client.OnRoomStateMessage(func(message twitch.RoomStateMessage) {
		// engine.LogInfo("OnRoomStateMessage", "raw", message.Raw)
	})
	client.OnUserNoticeMessage(func(message twitch.UserNoticeMessage) {
		engine.LogInfo("OnUserNoticeMessage")
	})
	client.OnUserStateMessage(func(message twitch.UserStateMessage) {
		// engine.LogInfo("OnUserStateMessage", "raw", message.Raw)
	})
	client.OnGlobalUserStateMessage(func(message twitch.GlobalUserStateMessage) {
		// engine.LogInfo("OnGlobalUserStateMessage", "raw", message.Raw)
	})
	client.OnNoticeMessage(func(message twitch.NoticeMessage) {
		engine.LogInfo("OnNoticeMessage", "message", message.Message)
	})
	client.OnUserJoinMessage(func(message twitch.UserJoinMessage) {
		engine.LogInfo("OnUserJoingMessage")
	})
	client.OnUserPartMessage(func(message twitch.UserPartMessage) {
		engine.LogInfo("OnUserPartgMessage")
	})
	client.OnSelfJoinMessage(func(message twitch.UserJoinMessage) {
		// engine.LogInfo("onSelfJoinMessage", "raw", message.Raw)
	})
	client.OnSelfPartMessage(func(message twitch.UserPartMessage) {
		engine.LogInfo("OnSelfPartMessage")
	})
	client.OnReconnectMessage(func(message twitch.ReconnectMessage) {
		engine.LogInfo("OnReconnectMessage")
	})
	client.OnNamesMessage(func(message twitch.NamesMessage) {
		// engine.LogInfo("OnNamesMessage")
	})
	client.OnPingMessage(func(message twitch.PingMessage) {
		// engine.LogInfo("OnPingMessage")
	})
	client.OnPongMessage(func(message twitch.PongMessage) {
		// engine.LogInfo("OnPongMessage")
	})
	client.OnUnsetMessage(func(message twitch.RawMessage) {
		// engine.LogInfo("OnUnsetMessage", "raw", message.Raw)
	})
	client.OnPingSent(func() {
		// engine.LogInfo("OnPingSent")
	})

	client.Join("photonsalon")

	err := client.Connect()
	engine.LogInfo("MAIN 5")
	if err != nil {
		engine.LogInfo("MAIN 6")
		panic(err)
	}
	engine.LogInfo("MAIN 7!!!")
	select {}
	// unreachable
}

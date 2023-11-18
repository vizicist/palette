package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	twitch "github.com/gempir/go-twitch-irc/v3"
	"github.com/vizicist/palette/engine"
)

func main() {

	engine.InitLog("chat")

	flag.Parse()

	err := StartTwitch()
	if err != nil {
		engine.LogError(err)
	}
	engine.LogInfo("Chat is exiting")
}

func StartTwitch() error {

	clientUserName := os.Getenv("TWITCH_USER")
	if clientUserName == "" {
		return fmt.Errorf("StartTwitch: TWITCH_USER not set")
	}
	clientAuthenticationToken := os.Getenv("TWITCH_TOKEN")
	if clientAuthenticationToken == "" {
		return fmt.Errorf("StartTwitch: TWITCH_TOKEN not set")
	}
	client := twitch.NewClient(clientUserName, clientAuthenticationToken)

	client.OnConnect(func() {
		engine.LogInfo("Twitch OnConnect", "clientUserName", clientUserName)
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
		for i := range words {
			words[i] = strings.ToLower(words[i])
		}
		if len(words) == 0 {
			client.Reply("photonsalon", id, "No command given?")
		} else {
			switch words[0] {

			case "randomize":
				category := "quad"
				if len(words) > 1 {
					category = words[1]
				}
				engine.LogInfo("randomize message", "category", category)
				vals, err := engine.EngineRemoteApi("quadpro.loadrand", "category", category)
				var reply string
				if err != nil {
					reply = fmt.Sprintf("err=%s", err.Error())
				} else {
					result := vals["result"]
					reply = fmt.Sprintf("Preset = %s", result)
				}
				client.Reply("photonsalon", id, reply)

			case "list":
				category := "quad"
				if len(words) > 1 {
					category = words[1]
				}
				engine.LogInfo("list message", "category", category)
				vals, err := engine.EngineRemoteApi("saved.list", "category", category)
				var reply string
				if err != nil {
					reply = fmt.Sprintf("err=%s", err.Error())
				} else {
					reply = vals["result"]
				}
				limit := 200
				if len(reply) > limit {
					reply = reply[:limit] + "..."
				}
				engine.LogInfo("list message reply", "reply", reply)
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
	if err != nil {
		return fmt.Errorf("unable to connect to twitch, clientUserName=%s clientAuthenticationToken=%s err=%s", clientUserName, clientAuthenticationToken, err.Error())
	}
	select {}
	// unreachable
}

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	twitch "github.com/gempir/go-twitch-irc/v3"
	"github.com/vizicist/palette/kit"
)

func main() {

	kit.InitLog("chat")

	flag.Parse()

	err := StartTwitch()
	if err != nil {
		kit.LogError(err)
	}
	kit.LogInfo("Chat is exiting")
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
		kit.LogInfo("Twitch OnConnect", "clientUserName", clientUserName)
		// client.Say("photonsalon", fmt.Sprintf("OnConnect user=%s", clientUserName))
	})
	client.OnWhisperMessage(func(message twitch.WhisperMessage) {
		kit.LogInfo("OnWhisperMessage")
	})
	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		// kit.LogInfo("OnPrivateMessage", "raw", message.Raw)
		msg := strings.ToLower(message.Message)
		id := message.Tags["id"]
		kit.LogInfo("OnPrivateMessage", "msg", msg)
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
				kit.LogInfo("randomize message", "category", category)
				vals, err := kit.EngineRemoteApi("quadpro.loadrand", "category", category)
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
				kit.LogInfo("list message", "category", category)
				vals, err := kit.EngineRemoteApi("saved.list", "category", category)
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
				kit.LogInfo("list message reply", "reply", reply)
				client.Reply("photonsalon", id, reply)

			case "status":
				vals, err := kit.EngineRemoteApi("global.status")
				var reply string
				if err != nil {
					reply = fmt.Sprintf("err=%s", err.Error())
				} else {
					reply = fmt.Sprintf("vals=%v", vals)
				}
				client.Reply("photonsalon", id, reply)
			case "ping":
				kit.LogInfo("ping message", "msg", msg)
			}
		}

	})
	client.OnClearChatMessage(func(message twitch.ClearChatMessage) {
		kit.LogInfo("OnClearChatMessage")
	})
	client.OnClearMessage(func(message twitch.ClearMessage) {
		kit.LogInfo("OnClearMessage")
	})
	client.OnRoomStateMessage(func(message twitch.RoomStateMessage) {
		// kit.LogInfo("OnRoomStateMessage", "raw", message.Raw)
	})
	client.OnUserNoticeMessage(func(message twitch.UserNoticeMessage) {
		kit.LogInfo("OnUserNoticeMessage")
	})
	client.OnUserStateMessage(func(message twitch.UserStateMessage) {
		// kit.LogInfo("OnUserStateMessage", "raw", message.Raw)
	})
	client.OnGlobalUserStateMessage(func(message twitch.GlobalUserStateMessage) {
		// kit.LogInfo("OnGlobalUserStateMessage", "raw", message.Raw)
	})
	client.OnNoticeMessage(func(message twitch.NoticeMessage) {
		kit.LogInfo("OnNoticeMessage", "message", message.Message)
	})
	client.OnUserJoinMessage(func(message twitch.UserJoinMessage) {
		kit.LogInfo("OnUserJoingMessage")
	})
	client.OnUserPartMessage(func(message twitch.UserPartMessage) {
		kit.LogInfo("OnUserPartgMessage")
	})
	client.OnSelfJoinMessage(func(message twitch.UserJoinMessage) {
		// kit.LogInfo("onSelfJoinMessage", "raw", message.Raw)
	})
	client.OnSelfPartMessage(func(message twitch.UserPartMessage) {
		kit.LogInfo("OnSelfPartMessage")
	})
	client.OnReconnectMessage(func(message twitch.ReconnectMessage) {
		kit.LogInfo("OnReconnectMessage")
	})
	client.OnNamesMessage(func(message twitch.NamesMessage) {
		// kit.LogInfo("OnNamesMessage")
	})
	client.OnPingMessage(func(message twitch.PingMessage) {
		// kit.LogInfo("OnPingMessage")
	})
	client.OnPongMessage(func(message twitch.PongMessage) {
		// kit.LogInfo("OnPongMessage")
	})
	client.OnUnsetMessage(func(message twitch.RawMessage) {
		// kit.LogInfo("OnUnsetMessage", "raw", message.Raw)
	})
	client.OnPingSent(func() {
		// kit.LogInfo("OnPingSent")
	})

	client.Join("photonsalon")

	err := client.Connect()
	if err != nil {
		return fmt.Errorf("unable to connect to twitch, clientUserName=%s clientAuthenticationToken=%s err=%s", clientUserName, clientAuthenticationToken, err.Error())
	}
	select {}
	// unreachable
}

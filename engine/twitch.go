package engine

import (
	"fmt"
	"strings"
	"time"

	twitch "github.com/gempir/go-twitch-irc/v3"
)

const (
	// clientUsername            = "justinfan123123"
	// clientAuthenticationToken = "oauth:123123123"
	clientUsername            = "nosuchtim"
	clientAuthenticationToken = "oauth:9dudgfmilvgy76hgtsag361rcmpzfl"
)

func StartTwitch() {
	client := twitch.NewClient(clientUsername, clientAuthenticationToken)
	// client := twitch.NewAnonymousClient()

	client.OnConnect(func() {
		LogInfo("ONCONNECT!!")
		client.Say("photonsalon", fmt.Sprintf("Hello World OnConnect time=%s!", time.Now().String()))
	})
	client.OnWhisperMessage(func(message twitch.WhisperMessage) {
		LogInfo("OnWhisperMessage")
	})
	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		LogInfo("OnPrivateMessage", "raw", message.Raw)
		msg := strings.ToLower(message.Message)
		if strings.HasPrefix(msg, "p ") {
			client.Say("photonsalon", fmt.Sprintf("got p msg time=%s!", time.Now().String()))
		}
		if strings.Contains(msg, "ping") {
			LogInfo("ping message", "msg", msg)
		}
	})
	client.OnClearChatMessage(func(message twitch.ClearChatMessage) {
		LogInfo("OnClearChatMessage")
	})
	client.OnClearMessage(func(message twitch.ClearMessage) {
		LogInfo("OnClearMessage")
	})
	client.OnRoomStateMessage(func(message twitch.RoomStateMessage) {
		LogInfo("OnRoomStateMessage", "raw", message.Raw)
	})
	client.OnUserNoticeMessage(func(message twitch.UserNoticeMessage) {
		LogInfo("OnUserNoticeMessage")
	})
	client.OnUserStateMessage(func(message twitch.UserStateMessage) {
		LogInfo("OnUserStateMessage", "raw", message.Raw)
	})
	client.OnGlobalUserStateMessage(func(message twitch.GlobalUserStateMessage) {
		LogInfo("OnGlobalUserStateMessage", "raw", message.Raw)
	})
	client.OnNoticeMessage(func(message twitch.NoticeMessage) {
		LogInfo("OnNoticeMessage", "message", message.Message)
	})
	client.OnUserJoinMessage(func(message twitch.UserJoinMessage) {
		LogInfo("OnUserJoingMessage")
	})
	client.OnUserPartMessage(func(message twitch.UserPartMessage) {
		LogInfo("OnUserPartgMessage")
	})
	client.OnSelfJoinMessage(func(message twitch.UserJoinMessage) {
		LogInfo("onSelfJoinMessage", "raw", message.Raw)
	})
	client.OnSelfPartMessage(func(message twitch.UserPartMessage) {
		LogInfo("OnSelfPartMessage")
	})
	client.OnReconnectMessage(func(message twitch.ReconnectMessage) {
		LogInfo("OnReconnectMessage")
	})
	client.OnNamesMessage(func(message twitch.NamesMessage) {
		LogInfo("OnNamesMessage")
	})
	client.OnPingMessage(func(message twitch.PingMessage) {
		LogInfo("OnPingMessage")
	})
	client.OnPongMessage(func(message twitch.PongMessage) {
		LogInfo("OnPongMessage")
	})
	client.OnUnsetMessage(func(message twitch.RawMessage) {
		LogInfo("OnUnsetMessage", "raw", message.Raw)
	})
	client.OnPingSent(func() {
		LogInfo("OnPingSent")
	})

	client.Join("photonsalon")

	err := client.Connect()
	LogInfo("MAIN 5")
	if err != nil {
		LogInfo("MAIN 6")
		panic(err)
	}
	LogInfo("MAIN 7!!!")
	select {}
	// unreachable
}

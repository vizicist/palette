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
		Info("ONCONNECT!!")
		client.Say("photonsalon", fmt.Sprintf("Hello World OnConnect time=%s!", time.Now().String()))
	})
	client.OnWhisperMessage(func(message twitch.WhisperMessage) {
		Info("OnWhisperMessage")
	})
	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		Info("OnPrivateMessage", "raw", message.Raw)
		msg := strings.ToLower(message.Message)
		if strings.HasPrefix(msg, "p ") {
			client.Say("photonsalon", fmt.Sprintf("got p msg time=%s!", time.Now().String()))
		}
		if strings.Contains(msg, "ping") {
			Info("ping message", "msg", msg)
		}
	})
	client.OnClearChatMessage(func(message twitch.ClearChatMessage) {
		Info("OnClearChatMessage")
	})
	client.OnClearMessage(func(message twitch.ClearMessage) {
		Info("OnClearMessage")
	})
	client.OnRoomStateMessage(func(message twitch.RoomStateMessage) {
		Info("OnRoomStateMessage", "raw", message.Raw)
	})
	client.OnUserNoticeMessage(func(message twitch.UserNoticeMessage) {
		Info("OnUserNoticeMessage")
	})
	client.OnUserStateMessage(func(message twitch.UserStateMessage) {
		Info("OnUserStateMessage", "raw", message.Raw)
	})
	client.OnGlobalUserStateMessage(func(message twitch.GlobalUserStateMessage) {
		Info("OnGlobalUserStateMessage", "raw", message.Raw)
	})
	client.OnNoticeMessage(func(message twitch.NoticeMessage) {
		Info("OnNoticeMessage", "message", message.Message)
	})
	client.OnUserJoinMessage(func(message twitch.UserJoinMessage) {
		Info("OnUserJoingMessage")
	})
	client.OnUserPartMessage(func(message twitch.UserPartMessage) {
		Info("OnUserPartgMessage")
	})
	client.OnSelfJoinMessage(func(message twitch.UserJoinMessage) {
		Info("onSelfJoinMessage", "raw", message.Raw)
	})
	client.OnSelfPartMessage(func(message twitch.UserPartMessage) {
		Info("OnSelfPartMessage")
	})
	client.OnReconnectMessage(func(message twitch.ReconnectMessage) {
		Info("OnReconnectMessage")
	})
	client.OnNamesMessage(func(message twitch.NamesMessage) {
		Info("OnNamesMessage")
	})
	client.OnPingMessage(func(message twitch.PingMessage) {
		Info("OnPingMessage")
	})
	client.OnPongMessage(func(message twitch.PongMessage) {
		Info("OnPongMessage")
	})
	client.OnUnsetMessage(func(message twitch.RawMessage) {
		Info("OnUnsetMessage", "raw", message.Raw)
	})
	client.OnPingSent(func() {
		Info("OnPingSent")
	})

	client.Join("photonsalon")

	err := client.Connect()
	Info("MAIN 5")
	if err != nil {
		Info("MAIN 6")
		panic(err)
	}
	Info("MAIN 7!!!")
	select {}
	// unreachable
}

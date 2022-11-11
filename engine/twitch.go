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
		Log.Debugf("ONCONNECT!!")
		client.Say("photonsalon", fmt.Sprintf("Hello World OnConnect time=%s!", time.Now().String()))
	})
	client.OnWhisperMessage(func(message twitch.WhisperMessage) {
		Log.Debugf("OnWhisperMessage")
	})
	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		Log.Debugf("OnPrivateMessage raw=%s\n", message.Raw)
		msg := strings.ToLower(message.Message)
		if strings.HasPrefix(msg, "p ") {
			client.Say("photonsalon", fmt.Sprintf("got p msg time=%s!", time.Now().String()))
		}
		if strings.Contains(msg, "ping") {
			Log.Debugf(message.User.Name, "PONG", message.Message)
		}
	})
	client.OnClearChatMessage(func(message twitch.ClearChatMessage) {
		Log.Debugf("OnClearChatMessage")
	})
	client.OnClearMessage(func(message twitch.ClearMessage) {
		Log.Debugf("OnClearMessage")
	})
	client.OnRoomStateMessage(func(message twitch.RoomStateMessage) {
		Log.Debugf("OnRoomStateMessage raw=%s\n", message.Raw)
	})
	client.OnUserNoticeMessage(func(message twitch.UserNoticeMessage) {
		Log.Debugf("OnUserNoticeMessage")
	})
	client.OnUserStateMessage(func(message twitch.UserStateMessage) {
		Log.Debugf("OnUserStateMessage raw=%s", message.Raw)
	})
	client.OnGlobalUserStateMessage(func(message twitch.GlobalUserStateMessage) {
		Log.Debugf("OnGlobalUserStateMessage raw=%s\n", message.Raw)
	})
	client.OnNoticeMessage(func(message twitch.NoticeMessage) {
		Log.Debugf("OnNoticeMessage message=%s", message.Message)
	})
	client.OnUserJoinMessage(func(message twitch.UserJoinMessage) {
		Log.Debugf("OnUserJoingMessage")
	})
	client.OnUserPartMessage(func(message twitch.UserPartMessage) {
		Log.Debugf("OnUserPartgMessage")
	})
	client.OnSelfJoinMessage(func(message twitch.UserJoinMessage) {
		Log.Debugf("onSelfJoinMessage raw=%s\n", message.Raw)
	})
	client.OnSelfPartMessage(func(message twitch.UserPartMessage) {
		Log.Debugf("OnSelfPartMessage")
	})
	client.OnReconnectMessage(func(message twitch.ReconnectMessage) {
		Log.Debugf("OnReconnectMessage")
	})
	client.OnNamesMessage(func(message twitch.NamesMessage) {
		Log.Debugf("OnNamesMessage")
	})
	client.OnPingMessage(func(message twitch.PingMessage) {
		Log.Debugf("OnPingMessage")
	})
	client.OnPongMessage(func(message twitch.PongMessage) {
		Log.Debugf("OnPongMessage")
	})
	client.OnUnsetMessage(func(message twitch.RawMessage) {
		Log.Debugf("OnUnsetMessage raw=%s\n", message.Raw)
	})
	client.OnPingSent(func() {
		Log.Debugf("OnPingSent")
	})

	client.Join("photonsalon")

	err := client.Connect()
	Log.Debugf("MAIN 5")
	if err != nil {
		Log.Debugf("MAIN 6")
		panic(err)
	}
	Log.Debugf("MAIN 7!!!")
	select {}
	// unreachable
}

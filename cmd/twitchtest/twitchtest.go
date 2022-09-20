package main

import (
	"fmt"
	"log"
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

func main() {
	client := twitch.NewClient(clientUsername, clientAuthenticationToken)
	// client := twitch.NewAnonymousClient()

	client.OnConnect(func() {
		log.Println("ONCONNECT!!")
		client.Say("photonsalon", fmt.Sprintf("Hello World OnConnect time=%s!", time.Now().String()))
	})
	client.OnWhisperMessage(func(message twitch.WhisperMessage) {
		log.Println("OnWhisperMessage")
	})
	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		log.Printf("OnPrivateMessage raw=%s\n", message.Raw)
		msg := strings.ToLower(message.Message)
		if strings.HasPrefix(msg, "p ") {
			client.Say("photonsalon", fmt.Sprintf("got p msg time=%s!", time.Now().String()))
		}
		if strings.Contains(msg, "ping") {
			log.Println(message.User.Name, "PONG", message.Message)
		}
	})
	client.OnClearChatMessage(func(message twitch.ClearChatMessage) {
		log.Println("OnClearChatMessage")
	})
	client.OnClearMessage(func(message twitch.ClearMessage) {
		log.Println("OnClearMessage")
	})
	client.OnRoomStateMessage(func(message twitch.RoomStateMessage) {
		log.Printf("OnRoomStateMessage raw=%s\n", message.Raw)
	})
	client.OnUserNoticeMessage(func(message twitch.UserNoticeMessage) {
		log.Println("OnUserNoticeMessage")
	})
	client.OnUserStateMessage(func(message twitch.UserStateMessage) {
		log.Printf("OnUserStateMessage raw=%s", message.Raw)
	})
	client.OnGlobalUserStateMessage(func(message twitch.GlobalUserStateMessage) {
		log.Printf("OnGlobalUserStateMessage raw=%s\n", message.Raw)
	})
	client.OnNoticeMessage(func(message twitch.NoticeMessage) {
		log.Printf("OnNoticeMessage message=%s", message.Message)
	})
	client.OnUserJoinMessage(func(message twitch.UserJoinMessage) {
		log.Println("OnUserJoingMessage")
	})
	client.OnUserPartMessage(func(message twitch.UserPartMessage) {
		log.Println("OnUserPartgMessage")
	})
	client.OnSelfJoinMessage(func(message twitch.UserJoinMessage) {
		log.Printf("onSelfJoinMessage raw=%s\n", message.Raw)
	})
	client.OnSelfPartMessage(func(message twitch.UserPartMessage) {
		log.Println("OnSelfPartMessage")
	})
	client.OnReconnectMessage(func(message twitch.ReconnectMessage) {
		log.Println("OnReconnectMessage")
	})
	client.OnNamesMessage(func(message twitch.NamesMessage) {
		log.Println("OnNamesMessage")
	})
	client.OnPingMessage(func(message twitch.PingMessage) {
		log.Println("OnPingMessage")
	})
	client.OnPongMessage(func(message twitch.PongMessage) {
		log.Println("OnPongMessage")
	})
	client.OnUnsetMessage(func(message twitch.RawMessage) {
		log.Printf("OnUnsetMessage raw=%s\n", message.Raw)
	})
	client.OnPingSent(func() {
		log.Println("OnPingSent")
	})

	client.Join("photonsalon")

	err := client.Connect()
	log.Println("MAIN 5")
	if err != nil {
		log.Println("MAIN 6")
		panic(err)
	}
	log.Println("MAIN 7!!!")
	select {}
	// unreachable
}

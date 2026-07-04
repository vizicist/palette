package main

import (
	"fmt"
	"log"

	"github.com/andreykaipov/goobs"
	"github.com/vizicist/palette/kit"
)

func main() {
	// OBS_PASSWORD is read from the env file (.palette/.env), falling back to
	// the OS environment variable. If unset anywhere we warn and still attempt
	// the connection with an empty password (works when OBS auth is disabled).
	password := kit.EnvLookup("OBS_PASSWORD")
	if password == "" {
		log.Println("warning: OBS_PASSWORD not set in env file or environment, connecting with empty password")
	}

	client, err := goobs.New("localhost:4455", goobs.WithPassword(password))
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = client.Disconnect()
	}()

	version, _ := client.General.GetVersion()
	fmt.Printf("OBS Studio version: %s\n", version.ObsVersion)
	fmt.Printf("Websocket server version: %s\n", version.ObsWebSocketVersion)

	resp, _ := client.Scenes.GetSceneList()
	for _, v := range resp.Scenes {
		fmt.Printf("%2d %s\n", v.SceneIndex, v.SceneName)
	}
}

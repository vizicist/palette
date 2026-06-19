package main

import (
	"fmt"
	"log"

	"github.com/andreykaipov/goobs"
	"github.com/joho/godotenv"
	"github.com/vizicist/palette/kit"
)

func main() {
	myenv, err := godotenv.Read(kit.EnvFilePath())
	if err != nil {
		log.Fatalf("cannot read OBS env file: %v", err)
	}
	password := myenv["OBS_PASSWORD"]
	if password == "" {
		log.Fatal("OBS_PASSWORD not set in OBS env file")
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

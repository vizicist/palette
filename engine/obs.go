package engine

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/andreykaipov/goobs"
)

func ObsProcessInfo() *ProcessInfo {

	fullpath, err := GetParam("engine.obspath")
	LogIfError(err)
	if fullpath != "" && !FileExists(fullpath) {
		LogWarn("No OBS found, looking for", "path", fullpath)
		return nil
	}
	exe := filepath.Base(fullpath)
	pi := NewProcessInfo(exe, fullpath, "", ObsActivate)
	pi.DirPath = filepath.Dir(fullpath)
	pi.Arg = "--disable-shutdown-check"
	return pi
}

// ObsActivate is called in a goroutine, so it can block.
func ObsActivate() {
	time.Sleep(3 * time.Second)
	stream, err := GetParamBool("engine.obsstream")
	LogIfError(err)
	if err == nil && stream {
		LogInfo("ObsActivate calling streamstart")
		ObsCommand("streamstart")
	}
}

func ObsCommand(cmd string) error {

	client, err := goobs.New("localhost:4455", goobs.WithPassword("mantic0re"))
	if err != nil {
		return err
	}
	defer client.Disconnect()

	switch cmd {
	case "status":
		version, err := client.General.GetVersion()
		if err != nil {
			return err
		}
		recordStatus, err := client.Record.GetRecordStatus()
		if err != nil {
			return err
		}
		streamStatus, err := client.Stream.GetStreamStatus()
		if err != nil {
			return err
		}
		fmt.Printf("Streaming active: %v\n", streamStatus.OutputActive)
		fmt.Printf("Recording active: %v\n", recordStatus.OutputActive)
		fmt.Printf("OBS Studio version: %s\n", version.ObsVersion)
		// fmt.Printf("Websocket server version: %s\n", version.ObsWebSocketVersion)

	case "recordstart":
		_, err := client.Record.StartRecord()
		return err
	case "recordstop":
		_, err := client.Record.StopRecord()
		return err

	case "streamstart":
		_, err := client.Stream.StartStream()
		return err
	case "streamstop":
		_, err := client.Stream.StopStream()
		return err

	default:
		return fmt.Errorf("unknown obs command: %s", cmd)
	}

	/*
		resp, _ := client.Scenes.GetSceneList()
		for _, v := range resp.Scenes {
			fmt.Printf("%2d %s\n", v.SceneIndex, v.SceneName)
		}
	*/
	return nil
}

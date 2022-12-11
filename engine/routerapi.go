package engine

import (
	"fmt"
)

func extractAgent(argsmap map[string]string) string {
	playerName, playerok := argsmap["player"]
	if !playerok {
		playerName = "*"
	} else {
		delete(argsmap, "player")
	}
	return playerName
}

func (r *Router) executePlayerAPI(api string, argsmap map[string]string) (result string, err error) {

	return "", fmt.Errorf("executePlayerAPI needs work")
	/*
		playerName := extractPlayer(argsmap)

		switch api {

		case "event":
			return "", r.HandleInputEvent(playerName, argsmap)

		case "set":
			name, ok := argsmap["name"]
			if !ok {
				return "", fmt.Errorf("executePlayerAPI: missing name argument")
			}
			value, ok := argsmap["value"]
			if !ok {
				return "", fmt.Errorf("executePlayerAPI: missing value argument")
			}
			r.SetPlayerParamValue(playerName, name, value)
			return "", r.saveCurrentSnaps(playerName)

		case "setparams":
			for name, value := range argsmap {
				r.SetPlayerParamValue(playerName, name, value)
			}
			return "", r.saveCurrentSnaps(playerName)

		case "get":
			name, ok := argsmap["name"]
			if !ok {
				return "", fmt.Errorf("executePlayerAPI: missing name argument")
			}
			if playerName == "*" {
				return "", fmt.Errorf("executePlayerAPI: get can't handle *")
			}
			player, err := r.PlayerManager.GetPlayer(playerName)
			if err != nil {
				return "", err
			}
			return player.params.paramValueAsString(name)

		default:
			// The player-specific APIs above are handled
			// here in the Router context, but for everything else,
			// we punt down to the player's player.
			// player can be A, B, C, D, or *
			r.PlayerManager.ApplyToAllPlayers(func(player *Player) {
				_, err := player.ExecuteAPI(api, argsmap, "")
				if err != nil {
					LogError(err)
				}
			})
			return "", nil
		}
	*/
}

/*
func (r *Router) SetPlayerParamValue(playerName string, name string, value string) {
	ApplyToPlayersNamed(playerName, func(player *Player) {
		err := player.SetOneParamValue(name, value)
		if err != nil {
			LogError(err)
			// But don't fail completely, this might be for
			// parameters that no longer exist, and a hard failure would
			// cause more problems.
		}
	})
}
*/

func (r *Router) saveQuadPreset(presetName string) error {

	Warn("Router.saveQuadPreset needs work")
	return nil
	/*
		preset, err := LoadPreset(presetName)
		if err != nil {
			return err
		}
		// wantCategory is sound, visual, effect, snap, or quad
		path := preset.WriteableFilePath()
		s := "{\n    \"params\": {\n"

		sep := ""
		Info("saveQuadPreset", "preset", presetName)

			for _, ctx := range r.taskManager.agentsContext {
				Info("starting", "player", player.playerName)
				// Print the parameter values sorted by name
				fullNames := player.params.values
				sortedNames := make([]string, 0, len(fullNames))
				for k := range fullNames {
					sortedNames = append(sortedNames, k)
				}
				sort.Strings(sortedNames)

				for _, fullName := range sortedNames {
					valstring, err := player.params.paramValueAsString(fullName)
					if err != nil {
						LogError(err)
						continue
					}
					s += fmt.Sprintf("%s        \"%s-%s\":\"%s\"", sep, player.playerName, fullName, valstring)
					sep = ",\n"
				}
			}
			s += "\n    }\n}"
			data := []byte(s)
			return os.WriteFile(path, data, 0644)
	*/
}

/*
func (r *Router) executeProcessAPI(api string, apiargs map[string]string) (result string, err error) {
	switch api {

	case "start":
		process, ok := apiargs["process"]
		if !ok {
			err = fmt.Errorf("executeProcessAPI: missing process argument")
		} else {
			err = StartRunning(process)
		}

	case "stop":
		process, ok := apiargs["process"]
		if !ok {
			err = fmt.Errorf("executeProcessAPI: missing process argument")
		} else {
			err = StopRunning(process)
		}

	default:
		err = fmt.Errorf("executeProcessAPI: unknown api %s", api)
	}

	if err != nil {
		return "", err
	} else {
		return result, nil
	}
}
*/

/*
func (r *Router) saveCurrentSnaps(playerName string) error {
	ApplyToPlayersNamed(playerName, func(player *Player) {
		err := player.saveCurrentSnap()
		if err != nil {
			Warn("saveCurrentSnaps", "err", err)
		}
	})
	return nil
}
*/

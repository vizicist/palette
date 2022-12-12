package engine

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

	LogWarn("Router.saveQuadPreset needs work")
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

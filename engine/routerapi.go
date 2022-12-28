package engine

/*
func (r *Router) SetLayerParamValue(layerName string, name string, value string) {
	ApplyToLayersNamed(layerName, func(layer *Layer) {
		err := layer.SetOneParamValue(name, value)
		if err != nil {
			LogError(err)
			// But don't fail completely, this might be for
			// parameters that no longer exist, and a hard failure would
			// cause more problems.
		}
	})
}
*/


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

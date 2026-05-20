package kit

import (
	"fmt"
	"strconv"
)

func ExecuteCursorAPI(api string, apiargs map[string]string) (string, error) {
	switch api {
	case "event":
		return cursorEventAPI(api, apiargs)
	default:
		return "", fmt.Errorf("ExecuteCursorAPI: unrecognized api=%s", api)
	}
}

func cursorEventAPI(api string, apiargs map[string]string) (string, error) {
	if theCursorManager == nil {
		return "", fmt.Errorf("cursor.event: cursor manager is not initialized")
	}

	ddu, err := needStringArg("ddu", api, apiargs)
	if err != nil {
		return "", err
	}
	switch ddu {
	case "down", "drag", "up", "clear":
	default:
		return "", fmt.Errorf("cursor.event: bad ddu=%s", ddu)
	}

	source := optionalStringArg("source", apiargs, "remote")
	gid := 0
	if gidStr := optionalStringArg("gid", apiargs, ""); gidStr != "" {
		gid, err = strconv.Atoi(gidStr)
		if err != nil {
			return "", fmt.Errorf("cursor.event: bad gid value: %w", err)
		}
	}
	if gid == 0 {
		gid = theCursorManager.UniqueGID()
	}

	x, y, z := 0.0, 0.0, 0.0
	if ddu != "clear" {
		x, err = needFloatArg("x", api, apiargs)
		if err != nil {
			return "", err
		}
		y, err = needFloatArg("y", api, apiargs)
		if err != nil {
			return "", err
		}
		z, err = needFloatArg("z", api, apiargs)
		if err != nil {
			return "", err
		}
		x = boundValueZeroToOne(x)
		y = boundValueZeroToOne(y)
		z = boundValueZeroToOne(z)
	}

	ce := NewCursorEvent(gid, source, ddu, CursorPos{X: x, Y: y, Z: z})
	if areaStr := optionalStringArg("area", apiargs, ""); areaStr != "" {
		area, err := strconv.ParseFloat(areaStr, 64)
		if err != nil {
			return "", fmt.Errorf("cursor.event: bad area value: %w", err)
		}
		ce.Area = area
	}

	theCursorManager.ExecuteCursorEvent(ce)
	return JSONObject("gid", fmt.Sprintf("%d", gid)), nil
}

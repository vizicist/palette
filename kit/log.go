package kit

import (
	"sync"
	"strings"
	"fmt"
)

var LogEnabled = map[string]bool{
	"*":              false,
	"advance":        false,
	"api":            false,
	"attract":        false,
	"bidule":         false,
	"config":         false,
	"cursor":         false,
	"drawing":        false,
	"erae":           false,
	"exec":           false,
	"ffgl":           false,
	"freeframe":      false,
	"gensound":       false,
	"genvisual":      false,
	"gesture":        false,
	"go":             false,
	"goroutine":      false,
	"info":           false,
	"patch":          false,
	"process":        false,
	"keykit":         false,
	"layerapi":       false,
	"listeners":      false,
	"load":           false,
	"loop":           false,
	"midi":           false,
	"midicontroller": false,
	"midiports":      false,
	"mmtt":           false,
	"morph":          false,
	"mouse":          false,
	"nats":           false,
	"note":           false,
	"notify":         false,
	"osc":            false,
	"params":         false,
	"phrase":         false,
	"plugin":         false,
	"quant":          false,
	"saved":          false,
	"realtime":       false,
	"resolume":       false,
	"remote":         false,
	"router":         false,
	"scale":          false,
	"scheduler":      false,
	"task":           false,
	"transpose":      false,
	"value":          false,
}

var LogMutex sync.Mutex

func SetLogTypeEnabled(dtype string, b bool) {

	LogMutex.Lock()
	defer LogMutex.Unlock()

	d := strings.ToLower(dtype)
	_, ok := LogEnabled[d]
	if !ok {
		LogIfError(fmt.Errorf("SetLogTypeEnabled: logtype not recognized"), "logtype", d)
		return
	}
	LogEnabled[d] = b
}

func IsLogging(logtype string) bool {

	LogMutex.Lock()
	defer LogMutex.Unlock()

	b, ok := LogEnabled[logtype]
	if !ok {
		LogIfError(fmt.Errorf("IsLogging: logtype not recognized"), "logtype", logtype)
		return false
	}
	return b

}

func SetLogTypes(logtypes string) {

	LogMutex.Lock()
	for logtype := range LogEnabled {
		LogEnabled[logtype] = false
	}
	LogMutex.Unlock()

	if logtypes != "" {
		darr := strings.Split(logtypes, ",")
		for _, d := range darr {
			if d != "" {
				d := strings.ToLower(d)
				LogInfo("Turning logging ON for", "logtype", d)
				LogMutex.Lock()
				_, ok := LogEnabled[d]
				if !ok {
					LogIfError(fmt.Errorf("ResetLogTypes: logtype not recognized"), "logtype", d)
				} else {
					LogEnabled[d] = true
				}
				LogMutex.Unlock()
			}
		}
	}
}

func LogIfError(err error, keysAndValues ...any) {
	if err == nil {
		return
	}
	TheHost.LogError(err, keysAndValues...)
}

func LogWarn(msg string, keysAndValues ...any) {
	TheHost.LogWarn(msg,keysAndValues...)
}

// func LogOfType(logtypes string, msg string, keysAndValues ...any) {
// 	LogOfType(logtypes,msg,keysAndValues...)
// }

func LogInfo(msg string, keysAndValues ...any) {
	TheHost.LogInfo(msg,keysAndValues...)
}
func LogError(err error, keysAndValues ...any) {
	TheHost.LogError(err,keysAndValues...)
}

func LogOfType(logtypes string, msg string, keysAndValues ...any) {
	if (len(keysAndValues) % 2) != 0 {
		LogWarn("LogOfType function given bad number of arguments")
	}
	for _, logtype := range strings.Split(logtypes, ",") {
		isEnabled := IsLogging(logtype)
		if isEnabled {
			keysAndValues = append(keysAndValues, "logtype")
			keysAndValues = append(keysAndValues, logtype)
			keysAndValues = AppendExtraValues(keysAndValues)
			LogInfo(msg, keysAndValues...)
		}
	}
}
func AppendExtraValues(keysAndValues []any) []any {
	keysAndValues = append(keysAndValues, "click")
	keysAndValues = append(keysAndValues, int64(CurrentClick()))
	if IsLogging("goroutine") {
		keysAndValues = append(keysAndValues, "goroutine")
		keysAndValues = append(keysAndValues, fmt.Sprintf("%d", GoroutineID()))
	}
	return keysAndValues
}


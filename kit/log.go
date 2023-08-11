package kit

func LogIfError(err error, keysAndValues ...any) {
	if err == nil {
		return
	}
	TheHost.LogError(err, keysAndValues...)
}

func LogWarn(msg string, keysAndValues ...any) {
	TheHost.LogWarn(msg,keysAndValues...)
}

func LogOfType(logtypes string, msg string, keysAndValues ...any) {
	TheHost.LogOfType(logtypes,msg,keysAndValues...)
}

func LogInfo(msg string, keysAndValues ...any) {
	TheHost.LogInfo(msg,keysAndValues...)
}
func LogError(err error, keysAndValues ...any) {
	TheHost.LogError(err,keysAndValues...)
}
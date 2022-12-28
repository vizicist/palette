package engine

import (
	"fmt"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var TheLog *zap.SugaredLogger
var Time0Nanoseconds = time.Now().UnixNano()
var Time0 = time.Time{}
var FirstTime = true

func Uptime() float64 {
	now := time.Now()
	if FirstTime {
		FirstTime = false
		Time0 = now
	}
	return now.Sub(Time0).Seconds()
}

func myTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	// nanos := t.UnixNano() - Time0Nanoseconds
	// sec := float64(nanos) / float64(time.Second)
	// For some reason a %04.4f format doesn't seem to work,
	// so I do it manuall.
	// leftpart := int(math.Trunc(sec))
	// rf := sec - float64(leftpart)
	// rightpart := int(rf * 1000)
	// s := fmt.Sprintf("%06d,%04d.%04d", CurrentClick(), leftpart, rightpart)
	s := fmt.Sprintf("%.6f", Uptime())
	enc.AppendString(s)
}

func fileLogger(path string) *zap.Logger {

	stacktraceKey := ""
	if ConfigBoolWithDefault("stacktrace", false) {
		stacktraceKey = "stacktrace"
	}

	config := zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "",
		NameKey:        "name",
		TimeKey:        "uptime",
		CallerKey:      "", // "caller",
		FunctionKey:    "", // "function",
		StacktraceKey:  stacktraceKey,
		LineEnding:     "\n",
		EncodeTime:     myTimeEncoder,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// config.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(config)
	logFile, _ := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	writer := zapcore.AddSync(logFile)
	defaultLogLevel := zapcore.DebugLevel
	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, writer, defaultLogLevel),
	)

	// field := zapcore.Field{Type: zapcore.Int64Type, Integer: int64(CurrentClick()), Key: "click"}
	// core = core.With([]zapcore.Field{field})

	// logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.WarnLevel))
	return logger
}

func InitLog(logname string) {
	logpath := LogFilePath(logname + ".log")
	logger := fileLogger(logpath)
	TheLog = logger.Sugar()
	defer logger.Sync() // flushes buffer, if any
	LogInfo("InitLog ===========================", "logname", logname)
}

func IsLogging(logtype string) bool {
	_, ok := LogEnabled[logtype]
	return ok

}

// LogError will accept a nil value and do nothing
func LogError(err error, keysAndValues ...any) {

	if err == nil {
		return
	}

	if (len(keysAndValues) % 2) != 0 {
		LogWarn("LogOfType function given bad number of arguments")
	}
	keysAndValues = append(keysAndValues, "err")
	keysAndValues = append(keysAndValues, err)
	caller := "LogError"
	LogWarn(caller, keysAndValues...)
}

func appendExtraValues(keysAndValues []any) []any {
	keysAndValues = append(keysAndValues, "click")
	keysAndValues = append(keysAndValues, int64(CurrentClick()))
	keysAndValues = append(keysAndValues, "goroutine")
	keysAndValues = append(keysAndValues, fmt.Sprintf("%d", GoroutineID()))
	return keysAndValues
}

func DebugLogOfType(logtype string, msg string, keysAndValues ...any) {
	if (len(keysAndValues) % 2) != 0 {
		LogWarn("LogOfType function given bad number of arguments")
	}
	isEnabled, ok := LogEnabled[logtype]
	keysAndValues = append(keysAndValues, "logtype")
	keysAndValues = append(keysAndValues, logtype)
	if !ok {
		LogWarn("logtype not recognized", keysAndValues...)
	} else if isEnabled {
		keysAndValues = appendExtraValues(keysAndValues)
		TheLog.Infow(msg, keysAndValues...)
	}
}

func LogWarn(msg string, keysAndValues ...any) {
	if (len(keysAndValues) % 2) != 0 {
		LogWarn("Warn function given bad number of arguments")
	} else {
		keysAndValues = appendExtraValues(keysAndValues)
		keysAndValues = append(keysAndValues, "loglevel")
		keysAndValues = append(keysAndValues, "warn")
		TheLog.Warnw(msg, keysAndValues...)
	}
}

func LogInfo(msg string, keysAndValues ...any) {
	if (len(keysAndValues) % 2) != 0 {
		LogWarn("LogOfType function given bad number of arguments")
	}
	keysAndValues = appendExtraValues(keysAndValues)
	keysAndValues = append(keysAndValues, "loglevel")
	keysAndValues = append(keysAndValues, "info")
	TheLog.Infow(msg, keysAndValues...)
}

var LogEnabled = map[string]bool{
	"*":         false,
	"info":      false,
	"advance":   false,
	"api":       false,
	"config":    false,
	"drawing":   false,
	"cursor":    false,
	"erae":      false,
	"exec":      false,
	"freeframe": false,
	"gensound":  false,
	"genvisual": false,
	"go":        false,
	"layer":     false,
	"loop":      false,
	"midi":      false,
	"midiports": false,
	"mmtt":      false,
	"morph":     false,
	"phrase":    false,
	"layerapi":  false,
	"mouse":     false,
	"nats":      false,
	"notify":    false,
	"osc":       false,
	"plugin":    false,
	"preset":    false,
	"resolume":  false,
	"task":      false,
	"realtime":  false,
	"remote":    false,
	"router":    false,
	"scale":     false,
	"scheduler": false,
	"transpose": false,
	"value":     false,
}

func SetLogTypeEnabled(dtype string, b bool) {
	d := strings.ToLower(dtype)
	LogEnabled[d] = b
}

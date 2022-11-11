package engine

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.SugaredLogger

// sugar.Infof("Failed to fetch URL: %s", url)
func fileLogger(path string) *zap.Logger {
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(config)
	logFile, _ := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	writer := zapcore.AddSync(logFile)
	defaultLogLevel := zapcore.DebugLevel
	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, writer, defaultLogLevel),
	)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	return logger
}

func InitLog(logname string) {
	logpath := LogFilePath(logname + ".log")
	logger := fileLogger(logpath)
	Log = logger.Sugar()
	defer logger.Sync() // flushes buffer, if any
	Log.Debugf("Hello from zap logger")
}

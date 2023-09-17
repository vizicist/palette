package hostwin

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/vizicist/palette/kit"
)

var TheLog *zap.SugaredLogger

func myTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	// nanos := t.UnixNano() - Time0Nanoseconds
	// sec := float64(nanos) / float64(time.Second)
	// For some reason a %04.4f format doesn't seem to work,
	// so I do it manuall.
	// leftpart := int(math.Trunc(sec))
	// rf := sec - float64(leftpart)
	// rightpart := int(rf * 1000)
	// s := fmt.Sprintf("%06d,%04d.%04d", CurrentClick(), leftpart, rightpart)
	s := fmt.Sprintf("%.6f", kit.Uptime())
	enc.AppendString(s)
}


func zapEncoderConfig() zapcore.EncoderConfig {

	stacktraceKey := ""
	// stacktraceKey = "stacktrace" // use this if you want to get stack traces

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
	return config
}

func fileLogger(path string) *zap.Logger {

	config := zapEncoderConfig()
	fileEncoder := zapcore.NewJSONEncoder(config)
	logFile, _ := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	writer := zapcore.AddSync(logFile)
	defaultLogLevel := zapcore.DebugLevel
	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, writer, defaultLogLevel),
	)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.WarnLevel))
	return logger
}

func stdoutLogger() *zap.Logger {

	config := zapEncoderConfig()
	fileEncoder := zapcore.NewJSONEncoder(config)
	logFile := os.Stdout
	writer := zapcore.AddSync(logFile)
	defaultLogLevel := zapcore.DebugLevel
	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, writer, defaultLogLevel),
	)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.WarnLevel))
	return logger
}

// InitLog creates a logger to a log file, or stdout if logname is "".
func InitLog(logname string) {
	var logger *zap.Logger
	if logname == "" {
		logger = stdoutLogger()
	} else {
		logger = fileLogger(LogFilePath(logname + ".log"))
	}
	TheLog = logger.Sugar()
	defer LogIfError(logger.Sync()) // flushes buffer, if any
	date := time.Now().Format("2006-01-02 15:04:05")
	LogInfo("InitLog ==============================", "date", date, "logname", logname)
}

func SummarizeLog(fname string) error {
	file, err := os.Open(fname)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(file)
	nloaded := 0
	userMode := true
	startdate := ""
	uptimesecs := float64(0.0)
	line := ""
	for scanner.Scan() {
		line = scanner.Text()
		var values map[string]any
		if line[0] != '{' {
			continue
		}
		if err := json.Unmarshal([]byte(line), &values); err != nil {
			fmt.Printf("Error parsing line: %s\n", line)
			continue
		}
		msg, ok := values["msg"].(string)
		if !ok {
			continue
		}

		if strings.HasPrefix(msg, "InitLog") {
			startdate, ok = values["date"].(string)
			if !ok {
				startdate = ""
			}
			fmt.Printf("%s :: Starting Engine\n", startdate)
			nloaded = 0
			userMode = true
		} else if strings.HasPrefix(msg, "setAttractMode") {
			// This catches both plain setAttractMode and setAttractMode already
			turnAttractOn, ok := values["onoff"].(bool)
			if !ok {
				turnAttractOn = false
			}
			uptime, ok := values["uptime"].(string)
			if !ok {
				uptime = ""
			}
			uptimesecs, err = strconv.ParseFloat(uptime, 32)
			if err != nil {
				LogIfError(err)
				uptimesecs = 0.0
			}
			if turnAttractOn {
				if !userMode {
					// fmt.Printf("Already in attractMode? not resetting nloaded\n")
				} else {
					// Turning on attract mode means we've just finished a user session
					realstart := StartPlusUptime(startdate, uptimesecs)
					// fmt.Printf("User session: startdate=%s startsecs=%f nloaded=%d\n", startdate, modestart, nloaded)
					if nloaded > 0 {
						fmt.Printf("%s :: User session nloaded=%d\n", realstart, nloaded)
						nloaded = 0
					}
					userMode = false
				}
			} else {
				if userMode {
					// fmt.Printf("Already in userMode? not resetting nloaded\n")
				} else {
					// Turning off attract mode means we've just finished an attract session
					realstart := StartPlusUptime(startdate, uptimesecs)
					if nloaded > 0 {
						fmt.Printf("%s :: Attract session nloaded=%d\n", realstart, nloaded)
						nloaded = 0
					}
					userMode = true
				}
			}
		} else if strings.HasPrefix(msg, "QuadPro.Load") {
			nloaded++
		}
	}

	if nloaded > 0 {
		realstart := StartPlusUptime(startdate, uptimesecs)
		if !userMode {
			fmt.Printf("%s :: Attract session nloaded=%d\n", realstart, nloaded)
		} else {
			fmt.Printf("%s :: User session nloaded=%d\n", realstart, nloaded)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func StartPlusUptime(startdate string, uptime float64) string {
	layout := "2006-01-02 15:04:05"
	tt, err := time.Parse(layout, startdate)
	LogIfError(err)
	dur := time.Duration(uptime * float64(time.Second))
	realstart := tt.Add(dur)
	return realstart.Format(layout)
}

func LogFilePath(nm string) string {
	logdir := filepath.Join(LocalPaletteDir(), "logs")
	return filepath.Join(logdir, nm)
}

func LogIfError(err error, keysAndValues ...any) {
	if err == nil {
		return
	}
	LogError(err, keysAndValues...)
}

// LogIfError will accept a nil value and do nothing
func LogError(err error, keysAndValues ...any) {

	if err == nil {
		LogWarn("LogError given a nil error value")
		return
	}

	if (len(keysAndValues) % 2) != 0 {
		LogWarn("LogError function given bad number of arguments")
	}
	keysAndValues = append(keysAndValues, "err")
	keysAndValues = append(keysAndValues, err)
	caller := "LogError"
	LogWarn(caller, keysAndValues...)
}

func LogWarn(msg string, keysAndValues ...any) {
	if (len(keysAndValues) % 2) != 0 {
		LogWarn("Warn function given bad number of arguments")
	} else {
		keysAndValues = kit.AppendExtraValues(keysAndValues)
		keysAndValues = append(keysAndValues, "loglevel")
		keysAndValues = append(keysAndValues, "warn")
		TheLog.Warnw(msg, keysAndValues...)
	}
}

func LogInfo(msg string, keysAndValues ...any) {
	if (len(keysAndValues) % 2) != 0 {
		LogWarn("LogInfo function given bad number of arguments")
	}
	keysAndValues = kit.AppendExtraValues(keysAndValues)
	keysAndValues = append(keysAndValues, "loglevel")
	keysAndValues = append(keysAndValues, "info")
	TheLog.Infow(msg, keysAndValues...)
}


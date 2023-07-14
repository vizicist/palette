package engine

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var TheLog *zap.SugaredLogger
var Time0 = time.Time{}
var FirstTime = true
var LogMutex sync.Mutex

// Uptime returns the number of seconds since the program started.
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

// LogFilePath uses $PALETTE_LOGDIR if set
func LogFilePath(nm string) string {
	// $PALETTE_LOGDIR can override, but normally we create one in the Data directory with the hostname.
	// This allows the log files to be checked into github.
	logdir := os.Getenv("PALETTE_LOGDIR")
	if logdir == "" {
		logdir = filepath.Join(PaletteDataPath(), "logs", Hostname())
	}
	if _, err := os.Stat(logdir); os.IsNotExist(err) {
		err = os.MkdirAll(logdir, os.FileMode(0777))
		LogIfError(err) // not fatal?
	}
	return filepath.Join(logdir, nm)
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
		LogWarn("LogIfError function given bad number of arguments")
	}
	keysAndValues = append(keysAndValues, "err")
	keysAndValues = append(keysAndValues, err)
	caller := "LogIfError"
	LogWarn(caller, keysAndValues...)
}

func appendExtraValues(keysAndValues []any) []any {
	keysAndValues = append(keysAndValues, "click")
	keysAndValues = append(keysAndValues, int64(CurrentClick()))
	if IsLogging("goroutine") {
		keysAndValues = append(keysAndValues, "goroutine")
		keysAndValues = append(keysAndValues, fmt.Sprintf("%d", GoroutineID()))
	}
	return keysAndValues
}

func LogOfType(logtypes string, msg string, keysAndValues ...any) {
	if (len(keysAndValues) % 2) != 0 {
		LogWarn("LogOfType function given bad number of arguments")
	}
	for _, logtype := range strings.Split(logtypes, ",") {
		isEnabled := IsLogging(logtype)
		keysAndValues = append(keysAndValues, "logtype")
		keysAndValues = append(keysAndValues, logtype)
		if isEnabled {
			keysAndValues = appendExtraValues(keysAndValues)
			TheLog.Infow(msg, keysAndValues...)
		}
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
		LogWarn("LogInfo function given bad number of arguments")
	}
	keysAndValues = appendExtraValues(keysAndValues)
	keysAndValues = append(keysAndValues, "loglevel")
	keysAndValues = append(keysAndValues, "info")
	TheLog.Infow(msg, keysAndValues...)
}

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

func InitLogTypes() {
	logtypes := os.Getenv("PALETTE_LOGTYPES")
	SetLogTypes(logtypes)
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

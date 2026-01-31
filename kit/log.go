package kit

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	json "github.com/goccy/go-json"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var TheLog *zap.SugaredLogger
var Time0 = time.Time{}
var FirstTime = true
var LogMutex sync.Mutex

// var PaletteTimeLayout = "2006-01-02-15-04-05.000"
var PaletteTimeLayout = "2006-01-02T15:04:05Z07:00" // RFC3339 Format

// Uptime returns the number of seconds since the program started.
func Uptime() float64 {
	now := time.Now()
	if FirstTime {
		FirstTime = false
		Time0 = now
	}
	return now.Sub(Time0).Seconds()
}

func zapEncoderConfig() zapcore.EncoderConfig {

	stacktraceKey := ""
	// stacktraceKey = "stacktrace" // use this if you want to get stack traces

	config := zapcore.EncoderConfig{
		MessageKey:    "msg",
		LevelKey:      "",
		NameKey:       "name",
		TimeKey:       "", // uptime removed - absolute "time" field added via appendExtraValues
		CallerKey:     "", // "caller",
		FunctionKey:   "", // "function",
		StacktraceKey: stacktraceKey,
		LineEnding:    "\n",
		EncodeLevel:   zapcore.LowercaseLevelEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:  zapcore.ShortCallerEncoder,
	}
	return config
}

func fileLogger(path string) *zap.Logger {

	config := zapEncoderConfig()
	fileEncoder := zapcore.NewJSONEncoder(config)
	logFile, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	var writer zapcore.WriteSyncer
	if err != nil {
		// Can't open log file, fall back to no-op writer to avoid errors
		// This happens when log directory doesn't exist or isn't writable
		writer = noSyncWriter{io.Discard}
	} else {
		writer = zapcore.AddSync(logFile)
	}

	defaultLogLevel := zapcore.DebugLevel
	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, writer, defaultLogLevel),
	)
	// Use noSyncWriter for ErrorOutput to prevent "write error" messages
	// when zap tries to fsync stderr (which fails on terminals)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.WarnLevel),
		zap.ErrorOutput(noSyncWriter{os.Stderr}))
	return logger
}

// noSyncWriter wraps an io.Writer and ignores Sync() calls.
// This prevents "write error: invalid argument" messages when zap tries
// to fsync stdout/stderr (which fails on terminals).
type noSyncWriter struct {
	io.Writer
}

func (w noSyncWriter) Sync() error {
	return nil
}

func stdoutLogger() *zap.Logger {

	config := zapEncoderConfig()
	fileEncoder := zapcore.NewJSONEncoder(config)
	// Wrap stdout in noSyncWriter to prevent fsync errors on terminals
	writer := noSyncWriter{os.Stdout}
	defaultLogLevel := zapcore.ErrorLevel
	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, writer, defaultLogLevel),
	)
	// Use noSyncWriter for ErrorOutput to prevent "write error" messages
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.WarnLevel),
		zap.ErrorOutput(noSyncWriter{os.Stderr}))
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
	LogInfo("InitLog ==============================", "logname", logname)
}

func LogFatal(err error) {
	TheLog.Fatal(err)
}

func StartPlusUptime(startdate string, uptime float64) string {
	tt, err := time.Parse(PaletteTimeLayout, startdate)
	LogIfError(err)
	dur := time.Duration(uptime * float64(time.Second))
	realstart := tt.Add(dur)
	return realstart.Format(PaletteTimeLayout)
}

// LogFilePath uses $PALETTE_LOGDIR if set
func LogFilePath(nm string) string {
	// $PALETTE_LOGDIR can override, but normally we create one in the Data directory with the hostname.
	// This allows the log files to be checked into github.
	logdir := os.Getenv("PALETTE_LOGDIR")
	if logdir == "" {
		logdir = filepath.Join(PaletteDataPath(), "logs")
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
	if err == nil || TheLog == nil {
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
	LogRaw("error", "LogError", keysAndValues...)
}

func appendExtraValues(keysAndValues []any) []any {
	// Add absolute UTC timestamp for easy filtering
	keysAndValues = append(keysAndValues, "time")
	keysAndValues = append(keysAndValues, time.Now().UTC().Format(PaletteTimeLayout))
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
	LogRaw("warn", msg, keysAndValues...)
}

func LogRaw(loglevel string, msg string, keysAndValues ...any) {
	if (len(keysAndValues) % 2) != 0 {
		LogWarn("LogRaw function given bad number of arguments", "msg", msg)
	} else {
		keysAndValues = appendExtraValues(keysAndValues)
		keysAndValues = append(keysAndValues, "loglevel")
		keysAndValues = append(keysAndValues, loglevel)
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
	"obs":            false,
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

/*
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
*/

func InitLogTypes() {
	logtypes := os.Getenv("PALETTE_LOGTYPES")
	if logtypes == "" {
		s, err := GetParam("global.log")
		if err == nil {
			logtypes = s
		}
	}
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

// LogEntry represents a parsed log entry with absolute timestamp
type LogEntry struct {
	Time    string         `json:"time"`
	Msg     string         `json:"msg"`
	Uptime  float64        `json:"uptime"`
	Data    map[string]any `json:"-"` // All other fields
	RawJSON string         `json:"-"` // Original JSON line
}

// MaxLogResponseBytes is the maximum size of log response (~900KB to stay under 1MB NATS limit)
const MaxLogResponseBytes = 900000

// ReadLogEntries reads engine.log and returns entries filtered by time range
// startTime and endTime are optional (nil means no bound)
// limit caps the number of entries (0 means default of 50)
// offset skips entries for pagination
func ReadLogEntries(startTime, endTime *time.Time, limit, offset int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 500
	}

	logPath := LogFilePath("engine.log")
	file, err := os.Open(logPath)
	if err != nil {
		return nil, fmt.Errorf("cannot open engine.log: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Increase buffer size for potentially long lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	// For backward compatibility with old logs without "time" field
	var sessionStart time.Time
	var sessionStartUptime float64

	var results []map[string]any
	skipped := 0
	totalBytes := 0

	for scanner.Scan() {
		line := scanner.Text()
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		// Check for InitLog to establish session start time
		if msgStr, ok := entry["msg"].(string); ok && strings.Contains(msgStr, "InitLog") {
			// Check for "time" field, fall back to "date" for old log files
			timeStr, ok := entry["time"].(string)
			if !ok {
				timeStr, ok = entry["date"].(string)
			}
			if ok {
				if uptimeStr, ok := entry["uptime"].(string); ok {
					uptime, _ := strconv.ParseFloat(uptimeStr, 64)
					t, err := time.Parse(PaletteTimeLayout, timeStr)
					if err == nil {
						sessionStart = t
						sessionStartUptime = uptime
					}
				}
			}
		}

		// Get absolute time - prefer "time" field, fall back to computed time
		var absTime time.Time
		if timeStr, ok := entry["time"].(string); ok {
			// New format: use the "time" field directly
			t, err := time.Parse(PaletteTimeLayout, timeStr)
			if err == nil {
				absTime = t.UTC()
				// Normalize time to UTC in the output
				entry["time"] = absTime.Format(PaletteTimeLayout)
			}
		}

		// Fall back to computing from uptime if no "time" field
		if absTime.IsZero() {
			if sessionStart.IsZero() {
				// Can't compute time yet, skip this entry
				continue
			}
			uptimeStr, ok := entry["uptime"].(string)
			if !ok {
				continue
			}
			uptime, err := strconv.ParseFloat(uptimeStr, 64)
			if err != nil {
				continue
			}
			relativeUptime := uptime - sessionStartUptime
			absTime = sessionStart.Add(time.Duration(relativeUptime * float64(time.Second)))
			// Add time to entry for consistency
			entry["time"] = absTime.Format(PaletteTimeLayout)
		}

		// Filter by time range
		if startTime != nil && absTime.Before(*startTime) {
			continue
		}
		if endTime != nil && absTime.After(*endTime) {
			continue
		}

		// Handle offset
		if skipped < offset {
			skipped++
			continue
		}

		// Check size limit before adding
		entryJSON, _ := json.Marshal(entry)
		if totalBytes+len(entryJSON) > MaxLogResponseBytes {
			// Add truncation notice
			truncEntry := map[string]any{
				"_truncated": true,
				"_message":   "Response truncated due to size limit",
			}
			results = append(results, truncEntry)
			break
		}

		results = append(results, entry)
		totalBytes += len(entryJSON)

		if len(results) >= limit {
			break
		}
	}
	LogInfo("RESULTS LogEntries", "len", len(results))

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading engine.log: %w", err)
	}

	return results, nil
}

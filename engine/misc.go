package engine

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/gomail.v2"
)

// Debug controls debugging
var Debug = debugFlags{}

type debugFlags struct {
	Advance   bool
	API       bool
	Config    bool
	Drawing   bool
	Cursor    bool
	Erae      bool
	GenSound  bool
	GenVisual bool
	Go        bool
	Loop      bool
	MIDI      bool
	MMTT      bool
	Morph     bool
	MotorAPI  bool
	Mouse     bool
	NATS      bool
	Notify    bool
	OSC       bool
	Resolume  bool
	Responder bool
	Realtime  bool
	Remote    bool
	Router    bool
	Scale     bool
	Transpose bool
	Values    bool
}

func setDebug(dtype string, b bool) error {
	d := strings.ToLower(dtype)
	switch d {
	case "advance":
		Debug.Advance = b
	case "api":
		Debug.API = b
	case "config":
		Debug.Config = b
	case "cursor":
		Debug.Cursor = b
	case "drawing":
		Debug.Drawing = b
	case "erae":
		Debug.Erae = b
	case "executeapi":
		Debug.MotorAPI = b
	case "gen":
		Debug.GenSound = b
		Debug.GenVisual = b
	case "gensound":
		Debug.GenSound = b
	case "genvisual":
		Debug.GenVisual = b
	case "go":
		Debug.Go = b
	case "loop":
		Debug.Loop = b
	case "midi":
		Debug.MIDI = b
	case "mmtt":
		Debug.MMTT = b
	case "morph":
		Debug.Morph = b
	case "mouse":
		Debug.Mouse = b
	case "nats":
		Debug.NATS = b
	case "notify":
		Debug.Notify = b
	case "osc":
		Debug.OSC = b
	case "resolume":
		Debug.Resolume = b
	case "realtime":
		Debug.Realtime = b
	case "remote":
		Debug.Remote = b
	case "router":
		Debug.Router = b
	case "responder":
		Debug.Responder = b
	case "scale":
		Debug.Scale = b
	case "transpose":
		Debug.Transpose = b
	case "values":
		Debug.Values = b
	default:
		return fmt.Errorf("setDebug: unrecognized debug type=%s", dtype)
	}
	return nil
}

func BoundAndScaleController(v, vmin, vmax float32, cmin, cmax int) int {
	newv := BoundAndScaleFloat(v, vmin, vmax, float32(cmin), float32(cmax))
	return int(newv)
}

func BoundAndScaleFloat(v, vmin, vmax, outmin, outmax float32) float32 {
	if v < vmin {
		v = vmin
	} else if v > vmax {
		v = vmax
	}
	out := outmin + (outmax-outmin)*((v-vmin)/(vmax-vmin))
	return out
}

// InitDebug xxx
func InitDebug() {
	debug := ConfigValueWithDefault("debug", "")
	darr := strings.Split(debug, ",")
	for _, d := range darr {
		if d != "" {
			log.Printf("Turning Debug ON for %s\n", d)
			setDebug(d, true)
		}
	}
}

type logWriter struct {
	file *os.File
}

func (writer logWriter) Write(bytes []byte) (int, error) {

	t := time.Now()
	year, month, day := t.Date()
	hour, min, sec := t.Clock()
	micro := t.Nanosecond() / 1e3

	var s string
	if Debug.Go {
		goid := GoroutineID()
		// Add GO# to log to indicate Goroutine
		s = fmt.Sprintf("%04d/%02d/%02d %02d:%02d:%02d.%06d GO#%d %s",
			year, month, day, hour, min, sec, micro, goid, bytes)
	} else {
		s = fmt.Sprintf("%04d/%02d/%02d %02d:%02d:%02d.%06d %s",
			year, month, day, hour, min, sec, micro, bytes)

	}
	bb := []byte(s)
	return writer.file.Write(bb)
}

// InitLog xxx
func InitLog(logname string) {

	defaultLogger := log.Default()
	defaultLogger.SetFlags(0)

	logfile := logname + ".log"
	logpath := LogFilePath(logfile)
	file, err := os.OpenFile(logpath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("InitLog: Unable to open logfile=%s logpath=%s err=%s", logfile, logpath, err)
		return
	}
	log.SetFlags(0)
	logger := logWriter{file: file}
	log.SetFlags(0)
	log.SetOutput(logger)
}

// fileExists checks if a file exists
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		// complain but still act as if it doesn't exist
		log.Printf("fileExists: err=%s\n", err)
		return false
	}
	return true
}

var paletteRoot string

// PaletteDir is the value of environment variable PALETTE
func PaletteDir() string {
	if paletteRoot == "" {
		paletteRoot = os.Getenv("PALETTE")
		if paletteRoot == "" {
			log.Panicf("PALETTE environment variable needs to be set.")
		}
	}
	return paletteRoot
}

func PaletteVersion() string {
	path := filepath.Join(PaletteDir(), "VERSION")
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "Unknown" // It's okay if file isn't present
	}
	return string(bytes)
}

// ReadablePresetFilePath xxx
func ReadablePresetFilePath(preset string) string {
	return presetFilePath(preset)
}

// WritablePresetFilePath xxx
func WriteablePresetFilePath(preset string) string {
	path := presetFilePath(preset)
	os.MkdirAll(filepath.Dir(path), 0777)
	return path
}

func PresetsDir() string {
	return "presets"
}

// presetFilePath returns the full path of a preset file.
func presetFilePath(preset string) string {
	category := ""
	i := strings.Index(preset, ".")
	if i >= 0 {
		category = preset[0:i]
		preset = preset[i+1:]
	}
	presetjson := preset + ".json"
	localpath := filepath.Join(PaletteDataPath(), PresetsDir(), category, presetjson)
	return localpath
}

func PresetNameSplit(preset string) (string, string) {
	words := strings.SplitN(preset, ".", 2)
	if len(words) == 1 {
		return "", words[0]
	} else {
		return words[0], words[1]
	}
}

// MIDIFilePath xxx
func MIDIFilePath(nm string) string {
	return filepath.Join(PaletteDataPath(), "midifiles", nm)
}

// LocalPaletteDir gets used for local (and changed) presets and config
func LocalPaletteDir() string {
	localapp := os.Getenv("CommonProgramFiles")
	if localapp == "" {
		log.Printf("Expecting CommonProgramFiles to be set.")
		return ""
	}
	return filepath.Join(localapp, "Palette")
}

var localMap map[string]string

func LocalMap() map[string]string {
	if localMap == nil {
		var err error
		f := filepath.Join(LocalPaletteDir(), "local.json")
		if !FileExists(f) {
			// log.Printf("No local.json file, assuming datapath is data_default\n")
			localMap, _ = StringMap("{ \"datapath\": \"data_default\" }")
		} else {
			localMap, err = ReadConfigFile(f)
			if err != nil {
				log.Printf("Bad format of local.json?  err=%s\n", err)
			}
		}
	}
	return localMap
}

var paletteDataPath = ""

// PaletteDataPath returns the datadir value in local.json
func PaletteDataPath() string {

	if paletteDataPath != "" {
		return paletteDataPath
	}

	local := LocalMap()
	datapath, ok := local["datapath"]
	if !ok {
		datapath = filepath.Join(LocalPaletteDir(), "data_default")
	}
	if filepath.Dir(datapath) == "." {
		datapath = filepath.Join(LocalPaletteDir(), datapath)
	}
	paletteDataPath = datapath
	return datapath
}

// PaletteDataPath returns the datadir value in local.json
func TwitchUser() (username string, authtoken string) {
	local := LocalMap()
	twitchuser, ok := local["twitchuser"]
	if !ok {
		twitchuser = "foo"
	}
	twitchtoken, ok := local["twitchtoken"]
	if !ok {
		twitchtoken = "foo"
	}
	log.Printf("TwitchUser = %s %s\n", twitchuser, twitchtoken)
	return twitchuser, twitchtoken
}

// LocalConfigFilePath xxx
func ConfigFilePath(nm string) string {
	return filepath.Join(PaletteDataPath(), "config", nm)
}

// LogFilePath has a default if LocalPaletteDir fails
func LogFilePath(nm string) string {
	localdir := LocalPaletteDir()
	if localdir == "" {
		log.Printf("Warning - using c:/windows/tmp for log directory.\n")
		return filepath.Join("C:/windows/tmp", nm)
	}
	return filepath.Join(localdir, "logs", nm)
}

func FileExists(filepath string) bool {
	fileinfo, err := os.Stat(filepath)
	if os.IsNotExist(err) {
		return false
	}
	// Return false if the fileinfo says the file path is a directory.
	return !fileinfo.IsDir()
}

// StringMap takes a JSON string and returns a map of elements
func StringMap(params string) (map[string]string, error) {
	// The enclosing curly braces are optional
	if params == "" || params[0] == '"' {
		params = "{ " + params + " }"
	}
	dec := json.NewDecoder(strings.NewReader(params))
	t, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if t != json.Delim('{') {
		log.Printf("StringMap: no curly - %s\n", params)
		return nil, errors.New("expected '{' delimiter")
	}
	values := make(map[string]string)
	for dec.More() {
		name, err := dec.Token()
		if err != nil {
			return nil, err
		}
		if !dec.More() {
			return nil, errors.New("incomplete JSON?")
		}
		value, err := dec.Token()
		if err != nil {
			return nil, err
		}
		// The name and value Tokens can be floats or strings or ...
		n := fmt.Sprintf("%v", name)
		v := fmt.Sprintf("%v", value)
		values[n] = v
	}
	return values, nil
}

// ResultResponse returns a JSON 2.0 result response
func ResultResponse(resultObj interface{}) string {
	bytes, err := json.Marshal(resultObj)
	if err != nil {
		log.Printf("ResultResponse: unable to marshal resultObj\n")
		return ""
	}
	result := string(bytes)
	if result == "" {
		result = "\"0\""
	}
	return `{ "result": ` + result + ` }`
}

func jsonEscape(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\") // has to be first
	s = strings.ReplaceAll(s, "\b", "\\b")
	s = strings.ReplaceAll(s, "\f", "\\f")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// ErrorResponse return an error response
func ErrorResponse(err error) string {
	escaped := jsonEscape(err.Error())
	return `{ "error": "` + escaped + `" }`
}

// LoadImage reads an image file
func LoadImage(path string) (*image.NRGBA, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()
	nrgba := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(nrgba, nrgba.Bounds(), img, bounds.Min, draw.Src)
	return nrgba, nil
}

// GetString complains if a parameter is not there, but still returns ""
func GetString(pmap map[string]string, name string) (string, error) {
	value, ok := pmap[name]
	if !ok {
		return "", fmt.Errorf("GetString: no param value named %s!?", name)
	}
	return value, nil
}

// StringParamOfAPI xxx
func StringParamOfAPI(api string, pmap map[string]string, name string) (string, error) {
	value, ok := pmap[name]
	if !ok {
		return "", fmt.Errorf("api '%s' is missing required parameter '%s'", api, name)
	}
	return value, nil
}

// IsTrueValue returns true if the value is some version of true
func IsTrueValue(value string) (bool, error) {
	switch value {
	case "True":
		return true, nil
	case "true":
		return true, nil
	case "1":
		return true, nil
	case "on":
		return true, nil
	case "False":
		return false, nil
	case "false":
		return false, nil
	case "0":
		return false, nil
	case "off":
		return false, nil
	default:
		return false, fmt.Errorf("IsTrueValue: invalid boolean value (%s), assuming false", value)
	}
}

type NoWriter struct {
	// Source string
}

type FileWriter struct {
	File *os.File
}

var NoWriterInstance io.Writer

// InitLog xxx
func MakeFileWriter(path string) io.Writer {

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("MakeFileWriter: Unable to open path=%s err=%s", path, err)
		return nil
	}
	_ = file
	return &FileWriter{File: file}
	// return &FileWriter{File: file}
}

/*
func (w *FileWriter) Close() {
	err := w.File.Close()
	if err != nil {
		// doing a log.Printf here might be a recursive error
		// log.Printf("FileWriter.Close: err=%s\n", err)
	}
}
*/

func (w *FileWriter) Write(p []byte) (n int, err error) {
	s := string(p)
	newline := ""
	if !strings.HasSuffix(s, "\n") {
		newline = "\n"
	}
	final := fmt.Sprintf("%s%s", s, newline)
	w.File.Write([]byte(final))
	return len(p), nil
}

func (w *NoWriter) Write(p []byte) (n int, err error) {
	// ignore all output
	return len(p), nil
}

// ReadConfigFile xxx
func ReadConfigFile(path string) (map[string]string, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	pmap, err := StringMap(string(bytes))
	if err != nil {
		return nil, err
	}
	return pmap, nil
}

// ConfigBool returns bool value of nm, or false if nm not set
func ConfigBool(nm string) bool {
	v := ConfigValue(nm)
	if v == "" {
		return false
	}
	b, err := IsTrueValue(v)
	if err != nil {
		log.Printf("Config value of %s (%s) is invalid, assuming false\n", nm, v)
		return false
	}
	return b
}

// ConfigBoolWithDefault xxx
func ConfigBoolWithDefault(nm string, dflt bool) bool {
	v := ConfigValue(nm)
	b, err := IsTrueValue(v)
	if err != nil {
		return dflt
	}
	return b
}

func ConfigIntWithDefault(nm string, dflt int) int {
	s := ConfigValue(nm)
	if s == "" {
		return dflt
	}
	var val int
	nfound, err := fmt.Sscanf(s, "%d", &val)
	if nfound == 0 || err != nil {
		log.Printf("Config value of %s isn't an integer (%s)\n", nm, s)
		return dflt
	}
	return val
}

func ConfigFloatWithDefault(nm string, dflt float32) float32 {
	s := ConfigValue(nm)
	if s == "" {
		return dflt
	}
	var f float64
	f, err := strconv.ParseFloat(s, 32)
	if err != nil {
		log.Printf("Unable to parse config value of %s\n", s)
		return dflt
	}
	return float32(f)
}

func ConfigStringWithDefault(nm string, dflt string) string {
	s := ConfigValue(nm)
	if s == "" {
		return dflt
	}
	return s
}

var configMap map[string]string
var configMutex sync.Mutex

// ConfigValue returns "" if there's no value.  I.e. "" and 'no value' are identical
func ConfigValue(nm string) string {
	return ConfigValueWithDefault(nm, "")
}

func ConfigValueWithDefault(nm string, dflt string) string {

	configMutex.Lock()
	defer configMutex.Unlock()

	if configMap == nil {
		// Only do this once, perhaps should re-read if file has changed?
		path := ConfigFilePath("settings.json")
		var err error
		configMap, err = ReadConfigFile(path) // make sure you're setting global configMap
		if err != nil {
			log.Printf("ReadConfigFile: path=%s err=%s", path, err)
			return ""
		}
	}
	val, ok := configMap[nm]
	if ok {
		return val
	}
	// log.Printf("There is no config value named '%s'", nm)
	return dflt
}

func needFloatArg(nm string, api string, args map[string]string) (float32, error) {
	val, ok := args[nm]
	if !ok {
		return 0.0, fmt.Errorf("api/event=%s missing value for %s", api, nm)
	}
	f, err := strconv.ParseFloat(val, 32)
	if err != nil {
		return 0.0, fmt.Errorf("api/event=%s bad value, expecting float for %s, got %s", api, nm, val)
	}
	return float32(f), nil
}

func optionalStringArg(nm string, args map[string]string, dflt string) string {
	val, ok := args[nm]
	if !ok {
		return dflt
	}
	return val
}

func needStringArg(nm string, api string, args map[string]string) (string, error) {
	val, ok := args[nm]
	if !ok {
		return "", fmt.Errorf("api/event=%s missing value for %s", api, nm)
	}
	return val, nil
}

func needIntArg(nm string, api string, args map[string]string) (int, error) {
	val, ok := args[nm]
	if !ok {
		return 0, fmt.Errorf("api/event=%s missing value for %s", api, nm)
	}
	v, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("api/event=%s bad value for %s", api, nm)
	}
	return int(v), nil
}

func needBoolArg(nm string, api string, args map[string]string) (bool, error) {
	val, ok := args[nm]
	if !ok {
		return false, fmt.Errorf("api/event=%s missing value for %s", api, nm)
	}
	b, err := IsTrueValue(val)
	if err != nil {
		return false, fmt.Errorf("api/event=%s bad value for %s", api, val)
	}
	return b, nil
}

func ziplogs(logsdir string, zipfile string) error {
	file, err := os.Create(zipfile)
	if err != nil {
		return err
	}
	defer file.Close()

	w := zip.NewWriter(file)
	defer w.Close()

	walker := func(path string, info os.FileInfo, err error) error {
		// log.Printf("Crawling: %#v\n", path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		// Transform path into a zip-root relative path.
		lastslash := strings.LastIndex(path, "logs\\")
		relativePath := path
		if lastslash >= 0 {
			relativePath = relativePath[lastslash+5:]
		}
		f, err := w.Create(relativePath)
		if err != nil {
			return err
		}

		_, err = io.Copy(f, file)
		if err != nil {
			return err
		}

		return nil
	}
	err = filepath.Walk(logsdir, walker)
	if err != nil {
		log.Printf("filepath.Walk: err=%s\n", err)
	}
	return err
}

func SendLogs() error {

	recipient := ConfigValue("emailto")
	if recipient == "" {
		msg := "SendLogs: not sending, no emailto in settings"
		log.Printf("%s\n", msg)
		return fmt.Errorf(msg)
	}

	zipfile := ""
	logsdir := LogFilePath("")

	currentTime := time.Now()
	timeStampString := currentTime.Format("2006-01-02 15:04:05")
	layOut := "2006-01-02 15:04:05"
	hr := 0
	min := 0
	sec := 0
	timeStamp, err := time.Parse(layOut, timeStampString)
	if err == nil {
		hr, min, sec = timeStamp.Clock()
	}
	year, month, day := time.Now().Date()
	zipname := fmt.Sprintf("%s_logs_%04d_%02d_%02d_%02d_%02d_%02d.zip", Hostname(), year, month, day, hr, min, sec)
	zipfile = ConfigFilePath(zipname)
	err = ziplogs(logsdir, zipfile)
	if err != nil {
		return fmt.Errorf("sendLogs: err=%s", err)
	} else {
		log.Printf("SendLogs: zipfile=%s\n", zipfile)
	}
	body := fmt.Sprintf("host=%s palette logfiles attached\n", Hostname())
	return SendMailWithAttachment(body, zipfile)
}

func Hostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("SendMail: hostname err=%s\n", err)
		hostname = "Unknown"
	}
	return hostname
}

func SendMail(body string) error {
	return SendMailWithAttachment(body, "")
}

// SendMail xxx
func SendMailWithAttachment(body, attachfile string) error {

	recipient := ConfigValue("emailto")
	login := ConfigValue("emaillogin")
	password := ConfigValue("emailpassword")

	if recipient == "" {
		return fmt.Errorf("sendMail: not sending, no emailto in settings")
	}
	log.Printf("SendMail: recipient=%s\n", recipient)

	smtpHost := "smtp.gmail.com"
	smtpPort := 587
	subject := "Palette Report from " + Hostname()

	m := gomail.NewMessage()
	m.SetHeader("From", "me@timthompson.com")
	m.SetHeader("To", recipient)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)
	if attachfile != "" {
		m.Attach(attachfile)
	}

	d := gomail.NewDialer(smtpHost, smtpPort, login, password)

	return d.DialAndSend(m)
}

func GoroutineID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

func JsonObject(args ...string) string {
	if len(args)%2 != 0 {
		log.Printf("ApiParams: odd number of arguments, args=%v\n", args)
		return "{}"
	}
	params := ""
	sep := ""
	for n := range args {
		if n%2 == 0 {
			params = params + sep + "\"" + args[n] + "\": \"" + args[n+1] + "\""
		}
		sep = ", "
	}
	return "{" + params + "}"
}

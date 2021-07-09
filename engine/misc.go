package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	mail "gopkg.in/mail.v2"
)

// DebugUtil controls debugging
var DebugUtil = DebugFlags{}

// DebugFlags xxx
type DebugFlags struct {
	Advance   bool
	API       bool
	Cursor    bool
	GenVisual bool
	GenSound  bool
	Loop      bool
	Config    bool
	MIDI      bool
	Morph     bool
	NATS      bool
	OSC       bool
	Resolume  bool
	Notify    bool
	Realtime  bool
	Remote    bool
}

func setDebug(dtype string, b bool) error {
	d := strings.ToLower(dtype)
	switch d {
	case "advance":
		DebugUtil.Advance = b
	case "api":
		DebugUtil.API = b
	case "cursor":
		DebugUtil.Cursor = b
	case "notify":
		DebugUtil.Notify = b
	case "gen":
		DebugUtil.GenSound = b
		DebugUtil.GenVisual = b
	case "gensound":
		DebugUtil.GenSound = b
	case "genvisual":
		DebugUtil.GenVisual = b
	case "loop":
		DebugUtil.Loop = b
	case "config":
		DebugUtil.Config = b
	case "midi":
		DebugUtil.MIDI = b
	case "morph":
		DebugUtil.Morph = b
	case "nats":
		DebugUtil.NATS = b
	case "osc":
		DebugUtil.OSC = b
	case "resolume":
		DebugUtil.Resolume = b
	case "realtime":
		DebugUtil.Realtime = b
	case "remote":
		DebugUtil.Remote = b
	default:
		return fmt.Errorf("setDebug: unrecognized debug type=%s", dtype)
	}
	return nil
}

// InitDebug xxx
func InitDebug() {
	debug := ConfigValue("debug")
	darr := strings.Split(debug, ",")
	for _, d := range darr {
		if d != "" {
			log.Printf("Turning Debug ON for %s\n", d)
			setDebug(d, true)
		}
	}
}

// InitLogs xxx
func InitLogs() {
	logfile := "engine.log"
	logpath := LogFilePath(logfile)
	file, err := os.OpenFile(logpath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("InitLogs: Unable to open logfile=%s logpath=%s err=%s", logfile, logpath, err)
		return
	}
	log.Printf("Logs are being saved in %s\n", logpath)
	log.SetOutput(file)
	log.Printf("====================== Palette Engine is starting\n")
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

var paletteSourceLogged = false

// ConfigFilePath xxx
func ConfigFilePath(nm string) string {
	return filepath.Join(LocalPaletteDir(), "config", nm)
}

// MIDIFilePath xxx
func MIDIFilePath(nm string) string {
	return filepath.Join(LocalPaletteDir(), "midifiles", nm)
}

// LocalPaletteDir xxx
func LocalPaletteDir() string {
	// If PALETTESOURCE is defined, we use it so that during development,
	// everything (both logs and config) come from there
	ps := os.Getenv("PALETTESOURCE")
	if ps != "" {
		dir := filepath.Join(ps, "default")
		if !paletteSourceLogged {
			paletteSourceLogged = true
			log.Printf("Using PALETTESOURCE to get default from %s\n", dir)
		}
		return dir
	}
	localapp := os.Getenv("LOCALAPPDATA")
	if localapp == "" {
		log.Printf("Expecting LOCALAPPDATA to be set.")
		return ""
	}
	return filepath.Join(localapp, "Palette")
}

// LocalConfigFilePath xxx
func LocalConfigFilePath(nm string) string {
	localdir := LocalPaletteDir()
	if localdir == "" {
		return ""
	}
	return filepath.Join(localdir, "config", nm)
}

// LogFilePath xxx
func LogFilePath(nm string) string {
	localdir := LocalPaletteDir()
	return filepath.Join(localdir, "logs", nm)
}

// StringMap takes a JSON string and returns a map of elements
func StringMap(params string) (map[string]string, error) {
	dec := json.NewDecoder(strings.NewReader(params))
	t, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if t != json.Delim('{') {
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

// ErrorResponse return a JSON 2.0 error response
func ErrorResponse(err error) string {
	escaped := jsonEscape(err.Error())
	return `{ "error": { "code": 999, "message": "` + escaped + `" } }`
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

// SendMail xxx
func SendMail(recipient, subject, body string) error {
	log.Printf("mysendmail recipient=%s subject=%s len(body)=%d\n", recipient, subject, len(body))
	m := mail.NewMessage()
	m.SetHeader("From", "me@timthompson.com")
	m.SetHeader("To", recipient)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)
	// m.Attach("/home/Alex/lolcat.jpg")

	d := mail.NewDialer("smtp.gmail.com", 587, "me@timthompson.com", "zsdntvhomjnnmmmp")

	if err := d.DialAndSend(m); err != nil {
		panic(err)
	}
	return nil
}

// VizLogWriter xxx
type VizLogWriter struct {
	Source string
}

// InitLog xxx
func InitLog(source string) {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)
	log.SetOutput(&VizLogWriter{Source: source})
}

func (w *VizLogWriter) Write(p []byte) (n int, err error) {
	s := string(p)
	newline := ""
	if !strings.HasSuffix(s, "\n") {
		newline = "\n"
	}
	// Only add a prefix if the thing being written doesn't start with "<"
	// I.e. if there's already a log prefix, don't add another one.
	myprefix := ""
	if strings.Index(s, "<") < 0 {
		myprefix = "<" + w.Source + "> "
	}
	final := fmt.Sprintf("%s%s%s", myprefix, s, newline)
	os.Stderr.Write([]byte(final))
	return len(p), nil
}

// NoWriter xxx
type NoWriter struct {
	Source string
}

func (w *NoWriter) Write(p []byte) (n int, err error) {
	// ignore all output
	return len(p), nil
}

var configMap map[string]string
var configMutex sync.Mutex

// ReadConfigFile xxx
func ReadConfigFile(path string) (map[string]string, error) {
	bytes, err := ioutil.ReadFile(path)
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
		log.Printf("Config value of %s (%s) is invalid, assuming false", nm, v)
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

// ConfigValue returns "" if there's no value
func ConfigValue(nm string) string {

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

		// If it exists, merge local settings.json
		localpath := LocalConfigFilePath("settings.json")
		if localpath != "" && fileExists(localpath) {
			localconfigMap, err := ReadConfigFile(localpath)
			if err != nil {
				log.Printf("ReadConfigFile: localpath=%s err=%s", localpath, err)
			} else {
				log.Printf("Merging settings from %s\n", localpath)
				for k, v := range localconfigMap {
					configMap[k] = v
				}
			}
		}
	}
	val, ok := configMap[nm]
	if ok {
		return val
	}
	// log.Printf("There is no config value named '%s'", nm)
	return ""
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

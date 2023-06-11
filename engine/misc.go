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
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"gopkg.in/gomail.v2"
)

var PaletteExe = "palette.exe"
var MonitorExe = "palette_monitor.exe"
var EngineExe = "palette_engine.exe"
var GuiExe = "palette_gui.exe"
var BiduleExe = "bidule.exe"
var ResolumeExe = "avenue.exe"
var KeykitExe = "key.exe"
var MmttExe = "mmtt_kinect.exe"

func KillAll() {
	LogInfo("KillAll")
	err := KillExecutable(KeykitExe)
	LogIfError(err)
	err = KillExecutable(MmttExe)
	LogIfError(err)
	err = KillExecutable(BiduleExe)
	LogIfError(err)
	err = KillExecutable(ResolumeExe)
	LogIfError(err)
	err = KillExecutable(GuiExe)
	LogIfError(err)
	err = KillExecutable(EngineExe)
	LogIfError(err)
}

func KillMonitor() {
	LogInfo("KillMonitor")
	err := KillExecutable(MonitorExe)
	LogIfError(err)
}

func IsRunning(process string) bool {
	if process == "engine" {
		return IsRunningExecutable(EngineExe)
	}
	return TheProcessManager.IsRunning(process)
}

// func IsEngineRunning() bool {
// 	return isRunningExecutable(EngineExe)
// 	// return isRunningExecutable(EngineExe) || isRunningExecutable(EngineExeDebug)
// }

// fileExists checks if a file exists
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		// complain but still act as if it doesn't exist
		LogIfError(err)
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
			LogWarn("PALETTE environment variable needs to be set.")
		}
	}
	return paletteRoot
}

func GetPaletteVersion() string {
	path := filepath.Join(PaletteDir(), "VERSION")
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "Unknown" // It's okay if file isn't present
	}
	return strings.TrimSuffix(string(bytes), "\n")
}

// MIDIFilePath xxx
func MIDIFilePath(nm string) string {
	return filepath.Join(PaletteDataPath(), "midifiles", nm)
}

// LocalPaletteDir gets used for local (and changed) things in saved and config
func LocalPaletteDir() string {
	localapp := os.Getenv("CommonProgramFiles")
	if localapp == "" {
		var err error
		tempdir, err := GetParam("engine.tempdir")
		LogWarn("Expecting CommonProgramFiles to be set, using engine.tempdir value", "tempdir", tempdir)
		if err != nil {
			LogIfError(err)
			return ""
		}
		localapp = tempdir
	}
	return filepath.Join(localapp, "Palette")
}

var DataDir = "data_omnisphere" // default, override with environment variable
var FullDataPath string

func PaletteDataPath() string {
	if FullDataPath != "" {
		return FullDataPath
	}
	// Let environment variable override the compiled-in default of DataDir
	dataPath := os.Getenv("PALETTE_DATA_PATH")
	if dataPath != "" {
		DataDir = dataPath
	}
	// The value can be either a full path or just the "data_..." part.
	if filepath.IsAbs(DataDir) {
		FullDataPath = DataDir
	} else {
		FullDataPath = filepath.Join(LocalPaletteDir(), DataDir)
	}
	return FullDataPath
}

func TwitchUser() (username string, authtoken string) {
	LogWarn("TwitchUser needs to be updated to use an environment variable")
	/*
		local := LocalMap()
		twitchuser, ok := local["twitchuser"]
		if !ok {
			twitchuser = "foo"
		}
		twitchtoken, ok := local["twitchtoken"]
		if !ok {
			twitchtoken = "foo"
		}
		LogInfo("TwitchUser", "user", twitchuser, "token", twitchtoken)
		return twitchuser, twitchtoken
	*/
	return
}

func ConfigDir() string {
	return filepath.Join(PaletteDataPath(), "config")
}

func ConfigFilePath(nm string) string {
	return filepath.Join(ConfigDir(), nm)
}

// LogFilePath has a default if LocalPaletteDir fails
func LogFilePath(nm string) string {
	localdir := LocalPaletteDir()
	if localdir == "" {
		tempdir, err := GetParam("engine.tempdir")
		if err != nil {
			LogIfError(err)
		}
		LogWarn("Last resort, using engine.tempdir value for log directory.", "tempdir", tempdir)
		return filepath.Join(tempdir, nm)
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

func MapString(amap map[string]string) string {
	final := ""
	sep := ""
	for _, val := range amap {
		final = final + sep + "\"" + val + "\""
		sep = ","
	}
	return final
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
		LogWarn("no curly", "params", params)
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

// ExtractAndRemoveValueOf removes a named value from a map and returns it.
// If the value doesn't exist, "" is returned.
func ExtractAndRemoveValueOf(valName string, argsmap map[string]string) string {
	val, ok := argsmap[valName]
	if !ok {
		val = ""
	}
	delete(argsmap, valName)
	return val
}

// ResultResponse returns a JSON 2.0 result response
func ResultResponse(resultObj any) string {
	bytes, err := json.Marshal(resultObj)
	if err != nil {
		LogWarn("ResultResponse: unable to marshal resultObj")
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

// IsTrueValue returns true if the value is some version of true, and false otherwise.
func IsTrueValue(value string) bool {
	switch value {
	case "True":
		return true
	case "true":
		return true
	case "1":
		return true
	case "on":
		return true
	case "False":
		return false
	case "false":
		return false
	case "0":
		return false
	case "off":
		return false
	default:
		LogIfError(fmt.Errorf("IsTrueValue: invalid boolean value (%s), assuming false", value))
		return false
	}
}

type NoWriter struct {
}

type FileWriter struct {
	Exe string
}

var NoWriterInstance io.Writer

func NewExecutableLogWriter(exe string) io.Writer {
	return &FileWriter{Exe: exe}
}

func (w *FileWriter) Write(p []byte) (n int, err error) {
	LogInfo("ExecutableOutput", "exe", w.Exe, "output", string(p))
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

var _ = needStringArg

/*
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
*/

func needBoolArg(nm string, api string, args map[string]string) (bool, error) {
	val, ok := args[nm]
	if !ok {
		return false, fmt.Errorf("api/event=%s missing value for %s", api, nm)
	}
	b := IsTrueValue(val)
	return b, nil
}

var _ = needBoolArg

func ziplogs(logsdir string, zipfile string) error {
	file, err := os.Create(zipfile)
	if err != nil {
		return err
	}
	defer file.Close()

	w := zip.NewWriter(file)
	defer w.Close()

	walker := func(path string, info os.FileInfo, err error) error {
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
		LogIfError(err)
	}
	return err
}

func RemoteAPI(api string, args ...string) (map[string]string, error) {

	if len(args)%2 != 0 {
		return nil, fmt.Errorf("RemoteAPI: odd nnumber of args, should be even")
	}
	apijson := "\"api\": \"" + api + "\""
	for n := range args {
		if n%2 == 0 {
			apijson = apijson + ",\"" + args[n] + "\": \"" + args[n+1] + "\""
		}
	}
	return RemoteAPIRaw(apijson)
}

func RemoteAPIRaw(args string) (map[string]string, error) {
	url := fmt.Sprintf("http://127.0.0.1:%d/api", HTTPPort)
	postBody := []byte(args)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(postBody))
	if err != nil {
		if strings.Contains(err.Error(), "target machine actively refused") {
			err = fmt.Errorf("Engine isn't running or responding")
		}
		return nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("RemoteAPIRaw: ReadAll err=%s", err)
	}
	output, err := StringMap(string(body))
	if err != nil {
		return nil, fmt.Errorf("RemoteAPIRaw: unable to interpret output, err=%s", err)
	}
	errstr, haserror := output["error"]
	if haserror && !strings.Contains(errstr, "exit status") {
		return map[string]string{}, fmt.Errorf("RemoteApiRaw: error=%s", errstr)
	}
	return output, nil
}

func SendLogs() error {

	recipient, err := GetParam("engine.emailto")
	LogIfError(err)
	if recipient == "" {
		msg := "SendLogs: not sending, no emailto in settings"
		LogWarn(msg)
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
		LogInfo("SendLogs", "zipfile", zipfile)
	}
	body := fmt.Sprintf("host=%s palette logfiles attached\n", Hostname())
	return SendMailWithAttachment(body, zipfile)
}

func Hostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		LogIfError(err)
		hostname = "Unknown"
	}
	return hostname
}

func SendMail(body string) error {
	return SendMailWithAttachment(body, "")
}

// SendMail xxx
func SendMailWithAttachment(body, attachfile string) error {

	recipient, _ := GetParam("engine.emailto")
	login, _ := GetParam("engine.emaillogin")
	password, _ := GetParam("engine.emailpassword")

	if recipient == "" {
		return fmt.Errorf("sendMail: not sending, no emailto in settings")
	}
	LogInfo("SendMail", "recipient", recipient)

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
	s := JsonString(args...)
	return "{ " + s + " }"
}

func JsonString(args ...string) string {
	if len(args)%2 != 0 {
		LogWarn("ApiParams: odd number of arguments", "args", args)
		return ""
	}
	params := ""
	sep := ""
	for n := range args {
		if n%2 == 0 {
			params = params + sep + "\"" + args[n] + "\": \"" + args[n+1] + "\""
		}
		sep = ", "
	}
	return params
}

func boundval32(v float64) float32 {
	if v < 0.0 {
		return 0.0
	}
	if v > 1.0 {
		return 1.0
	}
	return float32(v)
}

func GetNameValue(apiargs map[string]string) (name string, value string, err error) {
	name, ok := apiargs["name"]
	if !ok {
		err = fmt.Errorf("missing name argument")
		return
	}
	value, ok = apiargs["value"]
	if !ok {
		err = fmt.Errorf("missing value argument")
		return
	}
	return
}

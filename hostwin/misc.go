package hostwin

import (
	"archive/zip"
	"fmt"
	"image"
	"image/draw"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vizicist/palette/kit"
	"gopkg.in/gomail.v2"
)

var PaletteExe = "palette.exe"
var MonitorExe = "palette_monitor.exe"
var EngineExe = "palette_engine.exe"
var GuiExe = "palette_gui.exe"
var BiduleExe = "bidule.exe"
var ResolumeExe = "avenue.exe"

// var KeykitExe = "key.exe"
var MmttExe = "mmtt_kinect.exe"

var OscPort = 3333
var EventClientPort = 6666
var GuiPort = 3943

func IsRunning(process string) bool {
	if process == "engine" {
		return IsRunningExecutable(EngineExe)
	}
	return TheProcessManager.IsRunning(process)
}

func MonitorIsRunning() bool {
	return IsRunningExecutable(MonitorExe)
}

// func IsEngineRunning() bool {
// 	return isRunningExecutable(EngineExe)
// 	// return isRunningExecutable(EngineExe) || isRunningExecutable(EngineExeDebug)
// }

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

// LocalPaletteDir gets used for local (and changed) things in saved and config.
func LocalPaletteDir() string {
	localapp := os.Getenv("CommonProgramFiles")
	if localapp == "" {
		localapp = "C:/windows/temp"
	}
	return filepath.Join(localapp, "Palette")
}

var FullDataPath string

func PaletteDataPath() string {
	if FullDataPath != "" {
		return FullDataPath
	}
	FullDataPath = filepath.Join(LocalPaletteDir(), "data")
	return FullDataPath
}

func ConfigDir() string {
	return filepath.Join(PaletteDataPath(), "config")
}

func (h HostWin) ConfigFilePath(nm string) string {
	return filepath.Join(ConfigDir(), nm)
}

func (h HostWin) FileExists(filepath string) bool {
	fileinfo, err := os.Stat(filepath)
	if os.IsNotExist(err) {
		return false
	}
	// Return false if the fileinfo says the file path is a directory.
	return !fileinfo.IsDir()
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
	// Hack, don't log resolume output
	if w.Exe != "resolume" {
		LogInfo("ExecutableOutput", "exe", w.Exe, "output", string(p))
	}
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
	pmap, err := kit.StringMap(string(bytes))
	if err != nil {
		return nil, err
	}
	return pmap, nil
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

func (h HostWin) ArchiveLogs() error {

	LogInfo("CycleTheLogs is starting.")

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
	zipname := fmt.Sprintf("%s_logs_%04d_%02d_%02d_%02d_%02d_%02d", Hostname(), year, month, day, hr, min, sec)
	zippath, err := WritableSavedFilePath("logsarchive", zipname+".zip")
	LogIfError(err)
	LogInfo("CycleTheLogs should be zipping logs to", "zippath", zippath)

	err = ziplogs(logsdir, zippath)
	if err != nil {
		return fmt.Errorf("CycleTheLogs: err=%s", err)
	} else {
		LogInfo("CycleTheLogs is done.")
	}
	return nil
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

	recipient, _ := kit.GetParam("engine.emailto")
	login, _ := kit.GetParam("engine.emaillogin")
	password, _ := kit.GetParam("engine.emailpassword")

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

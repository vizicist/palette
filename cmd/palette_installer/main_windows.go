//go:build windows

package main

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"unsafe"

	"github.com/vizicist/palette/internal/installerbundle"
)

const (
	paletteURL = "https://github.com/vizicist/palette"
	publisher  = "Nosuch Media"

	mbOK           = 0x00000000
	mbYesNo        = 0x00000004
	mbIconError    = 0x00000010
	mbIconInfo     = 0x00000040
	mbIconQuestion = 0x00000020
	idYes          = 6
)

type installRecord struct {
	Manifest installerbundle.Manifest `json:"manifest"`
	Root     string                   `json:"root"`
	DataRoot string                   `json:"data_root"`
	Files    []string                 `json:"files"`
	Dirs     []string                 `json:"dirs,omitempty"`
}

type options struct {
	quiet       bool
	uninstall   bool
	installRoot string
	dataRoot    string
}

func main() {
	opts, err := parseOptions(os.Args[1:])
	if err != nil {
		finish(err, false)
	}
	exePath, err := os.Executable()
	if err != nil {
		finish(err, opts.quiet)
	}
	exePath, _ = filepath.Abs(exePath)
	if !opts.uninstall && strings.HasPrefix(strings.ToLower(filepath.Base(exePath)), "uninstall") {
		opts.uninstall = true
	}
	if opts.uninstall {
		err = uninstall(exePath, opts)
	} else {
		err = install(exePath, opts)
	}
	finish(err, opts.quiet)
}

func parseOptions(args []string) (options, error) {
	var opts options
	for i := 0; i < len(args); i++ {
		switch strings.ToLower(args[i]) {
		case "--quiet", "/silent", "/verysilent", "/suppressmsgboxes":
			opts.quiet = true
		case "--uninstall", "/uninstall":
			opts.uninstall = true
		case "/currentuser": // Accepted for compatibility; installs are always per-user.
		case "--install-root":
			i++
			if i == len(args) {
				return opts, errors.New("--install-root requires a path")
			}
			opts.installRoot = args[i]
		case "--data-root":
			i++
			if i == len(args) {
				return opts, errors.New("--data-root requires a path")
			}
			opts.dataRoot = args[i]
		default:
			return opts, fmt.Errorf("unknown installer option %q", args[i])
		}
	}
	return opts, nil
}

func install(exePath string, opts options) error {
	f, err := os.Open(exePath)
	if err != nil {
		return fmt.Errorf("open installer: %w", err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("inspect installer: %w", err)
	}
	bundle, err := installerbundle.Read(f, info.Size())
	if err != nil {
		return err
	}
	if err := rejectLegacySystemInstall(); err != nil {
		return err
	}
	root, err := installationRoot(bundle.Manifest, opts)
	if err != nil {
		return err
	}
	dataRoot, err := paletteDataRoot(opts)
	if err != nil {
		return err
	}
	if !opts.quiet {
		name := "Palette"
		if bundle.Manifest.Kind == "data" {
			name = "Palette data_" + bundle.Manifest.DataName
		}
		message := fmt.Sprintf("Install %s %s for the current user?\n\nDestination:\n%s", name, bundle.Manifest.Version, root)
		if messageBox("Palette Installer", message, mbYesNo|mbIconQuestion) != idYes {
			return nil
		}
	}
	if bundle.Manifest.Kind == "app" {
		stopPaletteProcesses()
	}
	files, dirs, err := extractPayload(bundle.Archive, root)
	if err != nil {
		return err
	}
	for _, name := range bundle.Manifest.Delete {
		if err := os.Remove(filepath.Join(root, filepath.FromSlash(name))); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove obsolete file %q: %w", name, err)
		}
	}
	uninstaller := filepath.Join(root, "uninstall.exe")
	if bundle.Manifest.Kind == "data" {
		uninstaller = filepath.Join(root, "uninstall_data_"+bundle.Manifest.DataName+".exe")
	}
	if err := copyStub(f, bundle.StubSize, uninstaller); err != nil {
		return err
	}
	record := installRecord{Manifest: bundle.Manifest, Root: root, DataRoot: dataRoot, Files: files, Dirs: dirs}
	if err := writeRecord(uninstaller+".json", record); err != nil {
		return err
	}
	if err := configureInstall(record, uninstaller); err != nil {
		return err
	}
	cleanupReplacedInstaller(root, bundle.Manifest.Kind)
	if !opts.quiet {
		messageBox("Palette Installer", "Installation completed successfully.", mbOK|mbIconInfo)
	}
	return nil
}

func installationRoot(manifest installerbundle.Manifest, opts options) (string, error) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return "", errors.New("LOCALAPPDATA is not set")
	}
	if manifest.Kind == "app" {
		if opts.installRoot != "" {
			return filepath.Abs(opts.installRoot)
		}
		return filepath.Join(localAppData, "Programs", "Palette"), nil
	}
	dataRoot, err := paletteDataRoot(opts)
	if err != nil {
		return "", err
	}
	return filepath.Join(dataRoot, "data_"+manifest.DataName), nil
}

func paletteDataRoot(opts options) (string, error) {
	dataRoot := opts.dataRoot
	if dataRoot == "" {
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			return "", errors.New("LOCALAPPDATA is not set")
		}
		dataRoot = filepath.Join(localAppData, "Palette")
	}
	dataRoot, err := filepath.Abs(dataRoot)
	if err != nil {
		return "", fmt.Errorf("resolve Palette data directory: %w", err)
	}
	return dataRoot, nil
}

func rejectLegacySystemInstall() error {
	programFiles := os.Getenv("ProgramFiles")
	var legacyDirs []string
	if programFiles != "" {
		legacyDirs = append(legacyDirs,
			filepath.Join(programFiles, "Palette"),
			filepath.Join(programFiles, "Common Files", "Palette"),
		)
	}
	for _, dir := range legacyDirs {
		if dir != "" {
			if info, err := os.Stat(dir); err == nil && info.IsDir() {
				return legacyInstallError()
			}
		}
	}
	for _, name := range []string{"PALETTE", "PALETTE_DATA", "PALETTE_DATAROOT"} {
		if commandSucceeds("reg.exe", "query", `HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Environment`, "/v", name) {
			return legacyInstallError()
		}
	}
	return nil
}

func legacyInstallError() error {
	return errors.New("a previous all-users Palette installation was detected. Run scripts\\cleanup_system_palette_install.bat as Administrator, then run this installer again")
}

func cleanupReplacedInstaller(root, kind string) {
	// Remove the previous per-user installer's registration and support files
	// only after the new payload has been installed successfully enough to take
	// over. Machine-wide predecessors are handled by the explicit cleanup script.
	appID := "{398CDB78-EAB7-4928-9F19-CC42107ACEFA}_is1"
	if kind == "data" {
		appID = "{443296B2-FB07-4F76-992A-A100D5E51ADF}_is1"
	}
	_ = runCommand("reg.exe", "delete", `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\`+appID, "/f")
	for _, pattern := range []string{"unins???.exe", "unins???.dat", "unins???.msg"} {
		matches, _ := filepath.Glob(filepath.Join(root, pattern))
		for _, name := range matches {
			_ = os.Remove(name)
		}
	}
}

func extractPayload(zr *zip.Reader, root string) ([]string, []string, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, nil, fmt.Errorf("create install directory: %w", err)
	}
	stage, err := os.MkdirTemp(root, ".palette-install-*")
	if err != nil {
		return nil, nil, fmt.Errorf("create staging directory: %w", err)
	}
	defer os.RemoveAll(stage)
	var files []string
	dirSet := make(map[string]bool)
	for _, entry := range zr.File {
		name := strings.TrimSuffix(entry.Name, "/")
		if !installerbundle.SafeRelativePath(name) {
			return nil, nil, fmt.Errorf("payload contains unsafe path %q", entry.Name)
		}
		if entry.Mode()&os.ModeSymlink != 0 {
			return nil, nil, fmt.Errorf("payload contains unsupported symlink %q", entry.Name)
		}
		destination := filepath.Join(stage, filepath.FromSlash(name))
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(destination, 0o755); err != nil {
				return nil, nil, fmt.Errorf("create payload directory %q: %w", name, err)
			}
			dirSet[filepath.ToSlash(name)] = true
			continue
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return nil, nil, fmt.Errorf("create payload directory: %w", err)
		}
		in, err := entry.Open()
		if err != nil {
			return nil, nil, fmt.Errorf("open payload file %q: %w", name, err)
		}
		out, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			in.Close()
			return nil, nil, fmt.Errorf("create payload file %q: %w", name, err)
		}
		_, copyErr := io.Copy(out, in)
		inErr := in.Close()
		outErr := out.Close()
		if copyErr != nil || inErr != nil || outErr != nil {
			return nil, nil, fmt.Errorf("extract payload file %q", name)
		}
		if installerbundle.IsPresetPath(name) {
			// Stamp the staged preset with the bundled modification time so
			// that, once installed, a later user edit is detectable and a
			// reinstall of the same version does not look "newer" than itself.
			_ = os.Chtimes(destination, entry.Modified, entry.Modified)
		}
		files = append(files, filepath.ToSlash(name))
		for dir := filepath.ToSlash(filepath.Dir(filepath.FromSlash(name))); dir != "."; dir = filepath.ToSlash(filepath.Dir(filepath.FromSlash(dir))) {
			dirSet[dir] = true
		}
	}
	var dirs []string
	for dir := range dirSet {
		dirs = append(dirs, dir)
	}
	sort.Slice(dirs, func(i, j int) bool { return len(dirs[i]) < len(dirs[j]) })
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o755); err != nil {
			return nil, nil, fmt.Errorf("create install directory %q: %w", dir, err)
		}
	}
	for _, name := range files {
		source := filepath.Join(stage, filepath.FromSlash(name))
		destination := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return nil, nil, fmt.Errorf("create install directory: %w", err)
		}
		if installerbundle.IsPresetPath(name) {
			keep, err := installedPresetIsNewer(destination, source)
			if err != nil {
				return nil, nil, fmt.Errorf("compare installed preset %q: %w", name, err)
			}
			if keep {
				continue // preserve the user's more recently modified preset
			}
		}
		if err := os.Remove(destination); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, nil, fmt.Errorf("replace installed file %q: %w", name, err)
		}
		if err := os.Rename(source, destination); err != nil {
			return nil, nil, fmt.Errorf("install file %q: %w", name, err)
		}
	}
	sort.Strings(files)
	return files, dirs, nil
}

// installedPresetIsNewer reports whether an already-installed preset at dest was
// modified more recently than the bundled version staged at source. When true,
// the installer keeps the user's copy instead of overwriting it. A missing
// destination (fresh install) or a directory is never "newer".
func installedPresetIsNewer(dest, source string) (bool, error) {
	destInfo, err := os.Stat(dest)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	if destInfo.IsDir() {
		return false, nil
	}
	sourceInfo, err := os.Stat(source)
	if err != nil {
		return false, err
	}
	return destInfo.ModTime().After(sourceInfo.ModTime()), nil
}

func copyStub(installer *os.File, size int64, destination string) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return fmt.Errorf("create uninstaller directory: %w", err)
	}
	tmp := destination + ".new"
	out, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("create uninstaller: %w", err)
	}
	_, copyErr := io.Copy(out, io.NewSectionReader(installer, 0, size))
	closeErr := out.Close()
	if copyErr != nil || closeErr != nil {
		os.Remove(tmp)
		return errors.New("write uninstaller")
	}
	if err := os.Remove(destination); err != nil && !errors.Is(err, os.ErrNotExist) {
		os.Remove(tmp)
		return fmt.Errorf("replace uninstaller: %w", err)
	}
	if err := os.Rename(tmp, destination); err != nil {
		return fmt.Errorf("publish uninstaller: %w", err)
	}
	return nil
}

func writeRecord(path string, record installRecord) error {
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("encode install record: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write install record: %w", err)
	}
	return nil
}

func configureInstall(record installRecord, uninstaller string) error {
	if record.Manifest.Kind == "app" {
		if err := os.MkdirAll(filepath.Join(record.DataRoot, "logs"), 0o755); err != nil {
			return fmt.Errorf("create Palette data directory: %w", err)
		}
		if err := setUserEnvironment("PALETTE", record.Root); err != nil {
			return err
		}
		if err := setUserEnvironment("PALETTE_DATAROOT", record.DataRoot); err != nil {
			return err
		}
		if err := addUserPath(filepath.Join(record.Root, "ffgl"), filepath.Join(record.Root, "bin")); err != nil {
			return err
		}
		if err := createShortcuts(record.Root); err != nil {
			return err
		}
	} else {
		if err := setUserEnvironment("PALETTE_DATA", record.Manifest.DataName); err != nil {
			return err
		}
		if err := setUserEnvironment("PALETTE_DATAROOT", record.DataRoot); err != nil {
			return err
		}
	}
	if err := registerUninstaller(record, uninstaller); err != nil {
		return err
	}
	broadcastEnvironmentChange()
	return nil
}

func uninstall(exePath string, opts options) error {
	data, err := os.ReadFile(exePath + ".json")
	if err != nil {
		return fmt.Errorf("read install record: %w", err)
	}
	var record installRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return fmt.Errorf("decode install record: %w", err)
	}
	if record.Root == "" || (record.Manifest.Kind != "app" && record.Manifest.Kind != "data") {
		return errors.New("install record is invalid")
	}
	for _, name := range record.Files {
		if !installerbundle.SafeRelativePath(name) {
			return fmt.Errorf("install record contains unsafe path %q", name)
		}
	}
	for _, name := range record.Dirs {
		if !installerbundle.SafeRelativePath(name) {
			return fmt.Errorf("install record contains unsafe directory %q", name)
		}
	}
	if !opts.quiet {
		name := "Palette"
		if record.Manifest.Kind == "data" {
			name = "Palette data_" + record.Manifest.DataName
		}
		if messageBox("Palette Uninstaller", "Remove "+name+" from this account?", mbYesNo|mbIconQuestion) != idYes {
			return nil
		}
	}
	if record.Manifest.Kind == "app" {
		stopPaletteProcesses()
		if err := removeUserPath(filepath.Join(record.Root, "ffgl"), filepath.Join(record.Root, "bin")); err != nil {
			return err
		}
		deleteUserEnvironmentIfEqual("PALETTE", record.Root)
		removeShortcuts()
	} else {
		current, _ := queryRegistryValue(`HKCU\Environment`, "PALETTE_DATA")
		if strings.EqualFold(current, record.Manifest.DataName) {
			deleteUserEnvironment("PALETTE_DATA")
		}
	}
	deleteUserEnvironmentIfEqual("PALETTE_DATAROOT", record.DataRoot)
	unregisterUninstaller(record.Manifest)
	removeInstalledFiles(record)
	broadcastEnvironmentChange()
	if !opts.quiet {
		messageBox("Palette Uninstaller", "Uninstallation completed successfully.", mbOK|mbIconInfo)
	}
	scheduleSelfRemoval(exePath, record.Root)
	return nil
}

func removeInstalledFiles(record installRecord) {
	files := append([]string(nil), record.Files...)
	sort.Slice(files, func(i, j int) bool { return len(files[i]) > len(files[j]) })
	for _, name := range files {
		os.Remove(filepath.Join(record.Root, filepath.FromSlash(name)))
	}
	dirs := append([]string(nil), record.Dirs...)
	for i, dir := range dirs {
		dirs[i] = filepath.Join(record.Root, filepath.FromSlash(dir))
	}
	dirs = append(dirs, record.Root)
	sort.Slice(dirs, func(i, j int) bool { return len(dirs[i]) > len(dirs[j]) })
	for _, dir := range dirs {
		os.Remove(dir)
	}
}

func stopPaletteProcesses() {
	for _, name := range []string{"palette_engine.exe", "palette_monitor.exe"} {
		cmd := exec.Command("taskkill.exe", "/F", "/T", "/IM", name)
		cmd.SysProcAttr = hiddenProcess()
		_ = cmd.Run()
	}
}

func setUserEnvironment(name, value string) error {
	if err := runCommand("reg.exe", "add", `HKCU\Environment`, "/v", name, "/t", "REG_EXPAND_SZ", "/d", value, "/f"); err != nil {
		return fmt.Errorf("set user environment variable %s: %w", name, err)
	}
	return nil
}

func deleteUserEnvironment(name string) {
	_ = runCommand("reg.exe", "delete", `HKCU\Environment`, "/v", name, "/f")
}

func deleteUserEnvironmentIfEqual(name, expected string) {
	current, err := queryRegistryValue(`HKCU\Environment`, name)
	if err == nil && strings.EqualFold(strings.TrimSpace(current), strings.TrimSpace(expected)) {
		deleteUserEnvironment(name)
	}
}

func addUserPath(names ...string) error {
	current, _ := queryRegistryValue(`HKCU\Environment`, "Path")
	parts := splitPath(current)
	for _, name := range names {
		if !containsFold(parts, name) {
			parts = append(parts, name)
		}
	}
	return setUserEnvironment("Path", strings.Join(parts, ";"))
}

func removeUserPath(names ...string) error {
	current, err := queryRegistryValue(`HKCU\Environment`, "Path")
	if err != nil {
		return nil
	}
	var kept []string
	for _, part := range splitPath(current) {
		if !containsFold(names, part) {
			kept = append(kept, part)
		}
	}
	return setUserEnvironment("Path", strings.Join(kept, ";"))
}

func splitPath(value string) []string {
	var result []string
	for _, part := range strings.Split(value, ";") {
		part = strings.TrimSpace(part)
		if part != "" && !containsFold(result, part) {
			result = append(result, part)
		}
	}
	return result
}

func containsFold(values []string, want string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(want)) {
			return true
		}
	}
	return false
}

func queryRegistryValue(key, name string) (string, error) {
	cmd := exec.Command("reg.exe", "query", key, "/v", name)
	cmd.SysProcAttr = hiddenProcess()
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		fields := strings.Fields(line)
		if len(fields) >= 3 && strings.EqualFold(fields[0], name) && strings.HasPrefix(fields[1], "REG_") {
			return strings.TrimSpace(strings.Join(fields[2:], " ")), nil
		}
	}
	return "", errors.New("registry value not found")
}

func createShortcuts(root string) error {
	programs := filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "Palette")
	if err := os.MkdirAll(programs, 0o755); err != nil {
		return fmt.Errorf("create Start Menu group: %w", err)
	}
	paletteExe := filepath.Join(root, "bin", "palette.exe")
	for _, shortcut := range []struct{ name, args string }{{"Start Palette.lnk", "start"}, {"Stop Palette.lnk", "stop"}} {
		path := filepath.Join(programs, shortcut.name)
		script := fmt.Sprintf("$w=New-Object -ComObject WScript.Shell;$s=$w.CreateShortcut('%s');$s.TargetPath='%s';$s.Arguments='%s';$s.WorkingDirectory='%s';$s.WindowStyle=7;$s.Save()", psQuote(path), psQuote(paletteExe), shortcut.args, psQuote(filepath.Dir(paletteExe)))
		if err := runCommand("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script); err != nil {
			return fmt.Errorf("create Start Menu shortcut: %w", err)
		}
	}
	url := "[InternetShortcut]\r\nURL=" + paletteURL + "\r\n"
	if err := os.WriteFile(filepath.Join(programs, "Palette on the Web.url"), []byte(url), 0o644); err != nil {
		return fmt.Errorf("create web shortcut: %w", err)
	}
	return nil
}

func removeShortcuts() {
	programs := filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "Palette")
	_ = os.RemoveAll(programs)
}

func psQuote(value string) string { return strings.ReplaceAll(value, "'", "''") }

func uninstallKey(manifest installerbundle.Manifest) string {
	name := "Palette"
	if manifest.Kind == "data" {
		name = "PaletteData_" + manifest.DataName
	}
	return `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\` + name
}

func registerUninstaller(record installRecord, uninstaller string) error {
	key := uninstallKey(record.Manifest)
	displayName := "Palette"
	if record.Manifest.Kind == "data" {
		displayName = "Palette data_" + record.Manifest.DataName
	}
	values := [][3]string{
		{"DisplayName", "REG_SZ", displayName},
		{"DisplayVersion", "REG_SZ", record.Manifest.Version},
		{"Publisher", "REG_SZ", publisher},
		{"URLInfoAbout", "REG_SZ", paletteURL},
		{"InstallLocation", "REG_SZ", record.Root},
		{"UninstallString", "REG_SZ", `"` + uninstaller + `" --uninstall`},
		{"QuietUninstallString", "REG_SZ", `"` + uninstaller + `" --uninstall --quiet`},
	}
	for _, value := range values {
		if err := runCommand("reg.exe", "add", key, "/v", value[0], "/t", value[1], "/d", value[2], "/f"); err != nil {
			return fmt.Errorf("register uninstaller: %w", err)
		}
	}
	return nil
}

func unregisterUninstaller(manifest installerbundle.Manifest) {
	_ = runCommand("reg.exe", "delete", uninstallKey(manifest), "/f")
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = hiddenProcess()
	return cmd.Run()
}

func commandSucceeds(name string, args ...string) bool { return runCommand(name, args...) == nil }

func hiddenProcess() *syscall.SysProcAttr { return &syscall.SysProcAttr{HideWindow: true} }

func scheduleSelfRemoval(exePath, root string) {
	script := fmt.Sprintf(`timeout /t 2 /nobreak >nul & del /f /q "%s" "%s" & rmdir "%s" 2>nul`, exePath, exePath+".json", root)
	cmd := exec.Command("cmd.exe", "/d", "/c", script)
	cmd.SysProcAttr = hiddenProcess()
	_ = cmd.Start()
}

func broadcastEnvironmentChange() {
	user32 := syscall.NewLazyDLL("user32.dll")
	proc := user32.NewProc("SendMessageTimeoutW")
	environment, _ := syscall.UTF16PtrFromString("Environment")
	_, _, _ = proc.Call(0xffff, 0x001A, 0, uintptr(unsafe.Pointer(environment)), 0x0002, 5000, 0)
}

func messageBox(title, message string, flags uintptr) int {
	user32 := syscall.NewLazyDLL("user32.dll")
	proc := user32.NewProc("MessageBoxW")
	titlePtr, _ := syscall.UTF16PtrFromString(title)
	messagePtr, _ := syscall.UTF16PtrFromString(message)
	result, _, _ := proc.Call(0, uintptr(unsafe.Pointer(messagePtr)), uintptr(unsafe.Pointer(titlePtr)), flags)
	return int(result)
}

func finish(err error, quiet bool) {
	if err != nil {
		if !quiet {
			messageBox("Palette Installer", err.Error(), mbOK|mbIconError)
		}
		os.Exit(1)
	}
	os.Exit(0)
}

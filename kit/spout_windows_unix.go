//go:build !windows

package kit

func syncSpoutWindows(level int, args map[string]string) error {
	return nil
}

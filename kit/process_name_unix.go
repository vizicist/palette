//go:build !windows

package kit

func executableName(base string) string {
	return base
}

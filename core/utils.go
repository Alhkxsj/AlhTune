package core

import (
	"fmt"
	"os/exec"
	"runtime"
)

// FormatSize formats file size in MB
func FormatSize(s int64) string {
	if s <= 0 {
		return "-"
	}
	return fmt.Sprintf("%.1f MB", float64(s)/1024/1024)
}

// OpenBrowser opens a URL in the default browser
func OpenBrowser(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd, args = "cmd", []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default:
		cmd = "xdg-open"
	}

	args = append(args, url)
	_ = exec.Command(cmd, args...).Start()
}

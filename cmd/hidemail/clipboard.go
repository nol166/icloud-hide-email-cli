package main

import (
	"os/exec"
	"runtime"
	"strings"
)

// tryCopy best-effort copies text to the system clipboard; returns true on success.
func tryCopy(text string) bool {
	var cmds [][]string
	switch runtime.GOOS {
	case "darwin":
		cmds = [][]string{{"pbcopy"}}
	case "windows":
		cmds = [][]string{{"clip"}}
	default: // linux, incl. WSL
		cmds = [][]string{{"wl-copy"}, {"xclip", "-selection", "clipboard"}, {"clip.exe"}}
	}
	for _, c := range cmds {
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Stdin = strings.NewReader(text)
		if cmd.Run() == nil {
			return true
		}
	}
	return false
}

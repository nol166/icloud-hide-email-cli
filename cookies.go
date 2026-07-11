package main

import (
	"os"
	"path/filepath"
	"strings"
)

// cookiePath returns ~/.config/hidemail/cookies.txt (respects XDG_CONFIG_HOME).
func cookiePath() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "hidemail", "cookies.txt"), nil
}

// saveCookies stores the raw Cookie header string with owner-only perms.
func saveCookies(header string) error {
	p, err := cookiePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(strings.TrimSpace(header)+"\n"), 0o600)
}

// loadCookies returns the saved Cookie header, or a non-nil error if unset.
func loadCookies() (string, error) {
	p, err := cookiePath()
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

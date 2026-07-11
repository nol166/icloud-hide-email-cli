package main

import (
	_ "embed"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
)

//go:embed web.html
var webHTML string

// authWeb serves a localhost page that walks the user through copying their
// iCloud Cookie header from DevTools and posts it back to the CLI.
//
// ponytail: manual copy is the ceiling — iCloud auth cookies are HttpOnly and
// unreadable by page JS, so zero-copy capture would need a CDP-driven debug
// browser or an extension. Add that only if the copy step proves too fiddly.
func authWeb() error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	addr := "http://" + ln.Addr().String()

	done := make(chan string, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, webHTML)
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		cookie := strings.TrimSpace(r.FormValue("cookie"))
		if !looksLikeCookies(cookie) {
			http.Error(w, "That doesn't look like a Cookie header — it should contain several name=value pairs including X-APPLE-WEBAUTH.", http.StatusBadRequest)
			return
		}
		w.Write([]byte("ok"))
		done <- cookie
	})

	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)
	defer srv.Close()

	fmt.Printf("Opening %s in your browser — follow the steps there.\n", addr)
	fmt.Println("(If it didn't open, paste that URL into your browser manually.)")
	openBrowser(addr)

	cookie := <-done
	if err := saveCookies(cookie); err != nil {
		return err
	}
	fmt.Println("Cookies saved. Verifying with iCloud…")
	if _, err := NewClient(); err != nil {
		return fmt.Errorf("saved, but verification failed: %w", err)
	}
	fmt.Println("Connected. You can close the browser tab. Try `hidemail gen`.")
	return nil
}

func looksLikeCookies(s string) bool {
	return len(s) > 40 && strings.Contains(s, "=") && strings.Contains(strings.ToUpper(s), "X-APPLE-WEBAUTH")
}

func openBrowser(u string) {
	var cmds [][]string
	switch runtime.GOOS {
	case "darwin":
		cmds = [][]string{{"open", u}}
	case "windows":
		cmds = [][]string{{"cmd", "/c", "start", u}}
	default: // linux, incl. WSL
		cmds = [][]string{{"wslview", u}, {"explorer.exe", u}, {"xdg-open", u}}
	}
	for _, c := range cmds {
		if exec.Command(c[0], c[1:]...).Start() == nil {
			return
		}
	}
}

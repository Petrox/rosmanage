package main

import (
	"log"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

// TerminalEvent describes one ssh command with its output and timings
type TerminalEvent struct {
	Sent     time.Time
	Begin    time.Time
	End      time.Time
	Command  string
	Stdout   string
	Stderr   string
	ExitCode int
}

func webterminal(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	log.Println("http", "terminal", r.RemoteAddr)
	addr := p.ByName("addr")
	h, ok := KnownHosts[addr]
	if !ok {
		panic("Not found " + addr)
	}
	cmd := r.FormValue("command")
	h.runTerminalCommand(cmd)
	http.Redirect(w, r, "/host/"+addr, 302)
}

package main

import (
	"log"
	"net/http"
	"strconv"
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

func (t *TerminalEvent) BeginTime() string {
	return t.Begin.Format(time.RFC3339)
}
func (t *TerminalEvent) SentTime() string {
	return t.Sent.Format(time.RFC3339)
}
func (t *TerminalEvent) RunTime() string {
	if t.End.Unix() < t.Begin.Unix() {
		return strconv.FormatFloat(time.Now().Sub(t.Begin).Seconds(), 0, 1, 64)
	}
	return strconv.FormatFloat(t.End.Sub(t.Begin).Seconds(), 0, 1, 64)
}
func (t *TerminalEvent) EndTime() string {
	return t.End.Format(time.RFC3339)
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

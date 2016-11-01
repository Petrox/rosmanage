package main

import (
	"log"
	"strings"
)

func (h *Host) rosmsgmd5() {
	packages, err := h.Props["rospack list"]
	if err != true {
		return
	}

	messages, err2 := h.Props["rosmsg list"]
	if err2 != true {
		return
	}
	pkgs := strings.Split(packages, "\n")
	msgs := strings.Split(messages, "\n")
	localpkgs := make(map[string]bool)

	for _, pkg := range pkgs {
		if !strings.Contains(pkg, "/opt/") {
			pkgfields := strings.Fields(pkg)
			if len(pkgfields) > 0 {
				localpkgs[pkgfields[0]] = true
			}
		}
	}
	retval := make([]string, 20)
	for _, msg := range msgs {
		msgfields := strings.Split(msg, "/")
		if len(msgfields) > 0 {
			if localpkgs[msgfields[0]] {
				msgmd5, err := h.sshCommand("bash -c rosmsg md5 " + msg)
				if err != nil {
					retval = append(retval, msg+" "+msgmd5)
					log.Println("rosmsg md5:", msg, msgmd5)
				}
			}
		}
	}

}

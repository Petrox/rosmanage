package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"os/user"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"
)

/*
    "github.com/google/uuid"
    "golang.org/x/crypto/ssh"
  	"log"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"
*/

// HostControlClient contains the details how can we control the given host
type HostControlClient struct {
	Active          bool
	CurrentJob      string
	CurrentJobSince time.Time
	client          *ssh.Client
	FirstTry        time.Time
	LastTry         time.Time
	Props           properties
	chQuit          chan bool
	chCommand       chan string
	chTerminal      chan TerminalEvent
}

func (h *Host) sshClientWorker() {
	if time.Since(h.ControlClient.FirstTry) > time.Hour*365*24 {
		h.ControlClient.FirstTry = time.Now()
	} else {
		if time.Since(h.ControlClient.LastTry) < cfgSSHRetry {
			return
		}
	}
	log.Println(h.Addr, "active", h.ControlClient.Active)
	h.ControlClient.Active = true
	log.Println(h.Addr, "active", h.ControlClient.Active)
	defer h.stoppedSSHClient()
	h.ControlClient.LastTry = time.Now()
	//	proc := exec.Command("ssh", "-o TCPKeepAlive", h.addr)
	key, err := getKeyFile()
	if err != nil {
		log.Println("SSH key error", h.Addr, err.Error())
		return
	}
	log.Println(h.Addr, "active", h.ControlClient.Active)
	usr, _ := user.Current()
	config := &ssh.ClientConfig{
		User: usr.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
	}
	client, err := ssh.Dial("tcp", h.Addr+":22", config)
	if err != nil {
		log.Println("SSH connect error", h.Addr, err.Error())
		return
	}
	//var retval string
	h.ControlClient.client = client
	rosmanageuuidstr, err := h.sshCommand("cat .rosmanage.uuid")
	log.Println(h.Addr, "active", h.ControlClient.Active)
	if err == nil {
		h.Props["rosmanage.uuid"] = rosmanageuuidstr
	} else {
		rosmanageuuid := uuid.New()
		log.Println("UUID", rosmanageuuid.String())
		h.sshCommand("echo '" + rosmanageuuid.String() + "' > .rosmanage.uuid")
		rosmanageuuidstr, err = h.sshCommand("cat .rosmanage.uuid")
		if err == nil {
			h.Props["rosmanage.uuid"] = rosmanageuuidstr
		}
	}
	log.Println(h.Addr, "active", h.ControlClient.Active)

	h.setStaticProps()
	h.setDynamicRareProps()
	h.setDynamicOftenProps()
	log.Println(h.Addr, "active", h.ControlClient.Active)
	var quit bool
	for !quit {
		select {
		case quit = <-h.ControlClient.chQuit:
			log.Println(h.Addr, "active", h.ControlClient.Active)
			if quit {
				log.Println("sshclient", h.Addr, "exiting")
			}
		case command := <-h.ControlClient.chCommand:
			log.Println("sshcommand", command, h.runCommand(command))
			log.Println(h.Addr, "active", h.ControlClient.Active)
		case term := <-h.ControlClient.chTerminal:
			log.Println("sshtermcommand", term.Command)
			log.Println(h.Addr, "active", h.ControlClient.Active)
			term.Begin = time.Now()
			term.Stdout = h.runCommand(term.Command)
			term.End = time.Now()
			h.TerminalHistory = append(h.TerminalHistory, term)
			log.Println("terminalhistory", h.Addr, h.TerminalHistory)

		case <-time.After(time.Second * 1):
			log.Println("timeout")
			log.Println(h.Addr, "active", h.ControlClient.Active)
			if time.Since(h.LastStaticUpdate) > cfgUpdateIntervalStatic {
				h.setStaticProps()
			}
			if time.Since(h.LastDynamicRareUpdate) > cfgUpdateIntervalDynamicRare {
				h.setDynamicRareProps()
			}
			if time.Since(h.LastDynamicOftenUpdate) > cfgUpdateIntervalDynamicOften {
				h.setDynamicOftenProps()
			}
		}

	}
}

func (h *Host) runTerminalCommand(command string) {
	log.Println("runTerminalCommandx", command, h.ControlClient.Active)
	log.Println(h.Addr, "active", h.ControlClient.Active)
	if !h.ControlClient.Active {
		return
	}
	log.Println("runTerminalCommandy", command)

	t := TerminalEvent{Sent: time.Now(), Command: command}
	log.Println("runTerminalCommand", command)
	h.ControlClient.chTerminal <- t
}

func (h *Host) runCommand(command string) string {
	retval, err := h.sshCommand(command)
	if err == nil {
		return retval
	}
	return ""
}

func (h *Host) setStaticProps() {
	h.LastScanned = time.Now()
	h.LastStaticUpdate = time.Now()
	h.setPropsViaSSH("pwd", "pwd")
	h.setPropsViaSSH("/usr/bin/whoami", "whoami")
	h.setPropsViaSSH("lsb_release -a", "lsb_release -a")
	h.setPropsViaSSH("uname -a", "uname -a")
	h.setPropsViaSSH("hostname", "hostname")
	h.setPropsViaSSH("dpkg --list ros*", "dpkg --list ros*")
	h.setPropsViaSSH("cat /proc/meminfo", "meminfo")
	h.setPropsViaSSH("cat /proc/cpuinfo", "cpuinfo")
}

func (h *Host) setDynamicRareProps() {
	h.LastScanned = time.Now()
	h.LastDynamicRareUpdate = time.Now()
	h.setPropsViaSSH("ifconfig", "ifconfig")
	h.setPropsViaSSH("lsusb", "lsusb")
	h.setPropsViaSSH("lshw", "lshw")
	h.setPropsViaSSH("cat .rosmanage.role", "rosmanage.role")
	h.setPropsViaSSH("which iperf", "which iperf")
	h.setPropsViaSSH("which nmap", "which nmap")
	h.setPropsViaSSH("rosmsg list", "rosmsg list")
	h.setPropsViaSSH("rospack list", "rospack list")
}

func (h *Host) setDynamicOftenProps() {
	h.LastScanned = time.Now()
	h.LastDynamicOftenUpdate = time.Now()
	h.setPropsViaSSH("ps aux", "ps aux")
	h.setPropsViaSSH("uptime", "uptime")
	h.setPropsViaSSH("rosnode list", "rosnode list")
	h.setPropsViaSSH("rostopic list", "rostopic list")
}

func (h *Host) setPropsViaSSH(command string, key string) {
	retval, err := h.sshCommand(command)
	if err == nil {
		h.Props[key] = strings.TrimSuffix(retval, "\n")
	}
}

func (h *HostControlClient) Working(command string) {
	h.CurrentJob = command
	h.CurrentJobSince = time.Now()
}

func (h *Host) sshCommand(command string) (string, error) {
	h.ControlClient.Working(command)
	defer h.ControlClient.Working("")
	controlsession, err := h.ControlClient.client.NewSession()
	if err != nil {
		log.Println("SSH session error", h.Addr, err.Error())
	}
	defer controlsession.Close()
	var b bytes.Buffer
	controlsession.Stdout = &b
	if err := controlsession.Run(command); err != nil {
		log.Println("SSH command error", h.Addr, command, err.Error())
		return "", err
	}
	// log.Println("SSH command result", h.Addr, command, b.String())
	return b.String(), nil
}

func getKeyFile() (key ssh.Signer, err error) {
	usr, _ := user.Current()
	file := usr.HomeDir + "/.ssh/id_rsa"
	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return
	}
	key, err = ssh.ParsePrivateKey(buf)
	if err != nil {
		return
	}
	return
}

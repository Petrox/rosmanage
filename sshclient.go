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
	h.ControlClient.Active = true
	defer h.stoppedSSHClient()
	h.ControlClient.LastTry = time.Now()
	//	proc := exec.Command("ssh", "-o TCPKeepAlive", h.addr)
	key, err := getKeyFile()
	if err != nil {
		log.Println("SSH key error", h.Addr, err.Error())
		return
	}
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
	var quit bool
	quit = checkUUIDforDuplicates(h)
	h.setStaticProps()
	h.setDynamicRareProps()
	h.setDynamicOftenProps()
	h.rosmsgmd5()
	for !quit {
		select {
		case quit = <-h.ControlClient.chQuit:
			if quit {
				log.Println("sshclient", h.Addr, "exiting")
			}
		case command := <-h.ControlClient.chCommand:
			log.Println("sshcommand", command, h.runCommand(command))
		case term := <-h.ControlClient.chTerminal:
			log.Println("sshtermcommand", term.Command)
			term.Begin = time.Now()
			term.Stdout = h.runCommand(term.Command)
			term.End = time.Now()
			h.TerminalHistory = append(h.TerminalHistory, term)
			log.Println("terminalhistory", h.Addr, h.TerminalHistory)

		case <-time.After(time.Second * 1):
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

// checkUUIDforDuplicates returns true if we should quit, false if we can move on
func checkUUIDforDuplicates(h *Host) bool {
	for _, host := range KnownHosts {
		if (host.Props["uuid"] == h.Props["uuid"]) && (host.Addr != h.Addr || host.NetAddr != h.NetAddr || host.Iface != h.Iface) {
			if host.ControlClient.Active {
				log.Println("compare: ", h.Iface, h.NetAddr, h.Addr, " --- ", host.Iface, h.NetAddr, h.Addr)
				if h.betterThan(host) > 0 {
					host.disconnect()
				} else {
					log.Println("ssh disconnect ", h.Iface, h.NetAddr, h.Addr, " because we're worse than ", host.Iface, host.NetAddr, host.Addr)
					return true
				}
			}
		}
	}
	return false
}

func (h *Host) runTerminalCommand(command string) {
	if !h.ControlClient.Active {
		return
	}
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
	h.setPropsViaSSH("if [ -r .rosmanage.role ]; then cat .rosmanage.role ; fi", "rosmanage.role")
	h.setPropsViaSSH("which iperf", "which iperf")
	h.setPropsViaSSH("which nmap", "which nmap")
	h.setPropsViaSSH("which lshw", "which lshw")
	h.setPropsViaSSH("which lsusb", "which lsusb")
	h.setPropsViaSSH("env", "env")
	h.setPropsViaSSH("bash -lc env", "bash -c env")
	if len(h.Props["which lsusb"]) > 0 {
		h.setPropsViaSSH("lsusb", "lsusb")
	}
	if len(h.Props["which lshw"]) > 0 {
		h.setPropsViaSSH("lshw", "lshw")
	}
	//	if strings.Contains(h.Props["env"], "ROS_DISTRO") && strings.Contains(h.Props["env"], "ROS_MASTER_URI") && strings.Contains(h.Props["env"], "ROS_PACKAGE_PATH") {
	h.setPropsViaSSH("bash -lc \"rosmsg list\"", "rosmsg list")
	//	h.setPropsViaSSH("bash -lc \"rosmsg list| while read MSG; do echo $MSG `rosmsg md5 $MSG`; done\"", "rosmsg list md5")
	h.setPropsViaSSH("bash -lc \"rospack list\"", "rospack list")
	//	}
}

func (h *Host) setDynamicOftenProps() {
	h.LastScanned = time.Now()
	h.LastDynamicOftenUpdate = time.Now()
	h.setPropsViaSSH("ps aux", "ps aux")
	h.setPropsViaSSH("uptime", "uptime")
	if strings.Contains(h.Props["env"], "ROS_DISTRO") && strings.Contains(h.Props["env"], "ROS_MASTER_URI") && strings.Contains(h.Props["env"], "ROS_PACKAGE_PATH") {
		h.setPropsViaSSH("rosnode list", "rosnode list")
		h.setPropsViaSSH("rostopic list", "rostopic list")
	}
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
	controlsession.Setenv("PS1", "Itssomething")
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	if err := controlsession.RequestPty("xterm", 400, 40, modes); err != nil {
		log.Println("request for pseudo terminal failed: ", err)
	}

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

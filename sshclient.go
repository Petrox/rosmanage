package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"os/user"
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

func sshClientWorker(h *Host) {
	if time.Since(h.client.firsttry) > time.Hour*365*24 {
		h.client.firsttry = time.Now()
	} else {
		if time.Since(h.client.lasttry) < cfgSSHRetry {
			return
		}
	}
	h.client.active = true
	defer h.stoppedSSHClient()
	h.client.lasttry = time.Now()
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
	h.client.client = client
	rosmanageuuidstr, err := sshCommand(h, client, "cat .rosmanage.uuid")
	if err == nil {
		h.Props["rosmanage.uuid"] = rosmanageuuidstr
	} else {
		rosmanageuuid := uuid.New()
		log.Println("UUID", rosmanageuuid.String())
		sshCommand(h, client, "echo '"+rosmanageuuid.String()+"' > .rosmanage.uuid")
		rosmanageuuidstr, err = sshCommand(h, client, "cat .rosmanage.uuid")
		if err == nil {
			h.Props["rosmanage.uuid"] = rosmanageuuidstr
		}
	}

	h.setStaticProps()
	h.setDynamicProps()

}

func (h *Host) setStaticProps() {
	h.setPropsViaSSH("/usr/bin/whoami", "whoami")
	h.setPropsViaSSH("lsusb", "lsusb")
	h.setPropsViaSSH("lsb_release -a", "lsb_release -a")
	h.setPropsViaSSH("uname -a", "uname -a")
	h.setPropsViaSSH("hostname", "hostname")
	h.setPropsViaSSH("dpkg --list ros*", "dpkg --list ros*")
}

func (h *Host) setDynamicProps() {
	h.setPropsViaSSH("ifconfig", "ifconfig")
	h.setPropsViaSSH("ps aux", "ps aux")
	h.setPropsViaSSH("cat .rosmanage.role", "rosmanage.role")
	h.setPropsViaSSH("cat /proc/meminfo", "meminfo")
	h.setPropsViaSSH("cat /proc/cpuinfo", "cpuinfo")
	h.setPropsViaSSH("which iperf", "which iperf")
	h.setPropsViaSSH("which nmap", "which nmap")
}

func (h *Host) setPropsViaSSH(command string, key string) {
	retval, err := sshCommand(h, h.client.client, command)
	if err == nil {
		h.Props[key] = retval
	}
}

func sshCommand(h *Host, client *ssh.Client, command string) (string, error) {
	controlsession, err := client.NewSession()
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
	log.Println("SSH command result", h.Addr, command, b.String())
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
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
		log.Println("SSH key error", h.addr, err.Error())
		return
	}
	usr, _ := user.Current()
	config := &ssh.ClientConfig{
		User: usr.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
	}
	client, err := ssh.Dial("tcp", h.addr+":22", config)
	if err != nil {
		log.Println("SSH connect error", h.addr, err.Error())
		return
	}
	//var retval string
	sshCommand(h, client, "/usr/bin/whoami")
	lsbRelease, err := sshCommand(h, client, "lsb_release -a")
	if err == nil {
		h.props["lsb_release"] = lsbRelease
	}
	rosmanagerolestr, err := sshCommand(h, client, "cat .rosmanage.role")
	if err == nil {
		h.props["rosmanage.role"] = rosmanagerolestr
	}

	rosmanageuuidstr, err := sshCommand(h, client, "cat .rosmanage.uuid")
	if err == nil {
		h.props["rosmanage.uuid"] = rosmanageuuidstr
	} else {
		rosmanageuuid := uuid.New()
		log.Println("UUID", rosmanageuuid.String())
		sshCommand(h, client, "echo '"+rosmanageuuid.String()+"' > .rosmanage.uuid")
		rosmanageuuidstr, err = sshCommand(h, client, "cat .rosmanage.uuid")
		if err == nil {
			h.props["rosmanage.uuid"] = rosmanageuuidstr
		}
	}

	//return, _ = sshCommand(controlsession, h.addr, "/usr/bin/whoami")
	//return, _ = sshCommand(controlsession, h.addr, "/usr/bin/whoami")
	if err != nil {

	}
	time.Sleep(time.Second * 10)
}

func sshCommand(h *Host, client *ssh.Client, command string) (string, error) {
	controlsession, err := client.NewSession()
	if err != nil {
		log.Println("SSH session error", h.addr, err.Error())
	}
	defer controlsession.Close()
	var b bytes.Buffer
	controlsession.Stdout = &b
	if err := controlsession.Run(command); err != nil {
		log.Println("SSH command error", h.addr, command, err.Error())
		return "", err
	}
	log.Println("SSH command result", h.addr, command, b.String())
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

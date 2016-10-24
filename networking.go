package main

import (
	"log"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// Network contains details about a network accessible to the app
type Network struct {
	network     string
	firstseen   time.Time
	lastscanned time.Time
	hostok      map[string]*Host
	props       properties
}

// Host contains details of a known remote host
type Host struct {
	addr        string
	rosroles    string
	rosid       string
	islocalhost bool
	ispreferred bool
	openports   map[int16]struct{}
	firstseen   time.Time
	lastscanned time.Time
	client      sshClient
	props       properties
}

type properties map[string]string

type sshClient struct {
	active    bool
	client    *ssh.Client
	firsttry  time.Time
	lasttry   time.Time
	props     properties
	chQuit    chan bool
	chCommand chan string
}

// KnownNetworks contains all the Networks we can access
var KnownNetworks = make(map[string]Network)

// KnownHosts containt all the Hosts we can access
var KnownHosts = make(map[string]Host)

func networkmain() {
	updateNetworks()
	interfaceticker := time.NewTicker(cfgInterfacepolling)
	go func() {
		for range interfaceticker.C {
			updateNetworks()
			scanNetworks()
		}
	}()
}

func getNetworks() map[string]Network {

	nets := make(map[string]Network)
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		// fmt.Printf("%v \n", iface.Flags)
		addrs, _ := iface.Addrs()
		if len(addrs) > 0 && (iface.Flags&net.FlagUp) > 0 {
			// fmt.Printf("%s : %s\n", iface.Name, addrs[0])
			thisnet := Network{network: addrs[0].String(), hostok: make(map[string]*Host), props: make(properties)}
			nets[iface.Name] = thisnet
		}
	}
	return nets
}

func updateNetworks() {
	nets := getNetworks()
	for name, net := range nets {
		if val, ok := KnownNetworks[name]; ok == false {
			log.Println("Network found:", name, net.network)
			net.firstseen = time.Now()
			net.lastscanned = time.Unix(0, 0)
			KnownNetworks[name] = net
		} else {
			if val.network != net.network {
				log.Println("Network changed:", name, val.network, net.network)
				net.firstseen = time.Now()
				net.lastscanned = time.Unix(0, 0)
				KnownNetworks[name] = net
			}
		}
	}
	for name, net := range KnownNetworks {
		if _, ok := nets[name]; ok == false {
			log.Println("Network gone:", name, net.network)
			delete(KnownNetworks, name)
		}
	}
}

func scanNetworks() {
	for index, halo := range KnownNetworks {
		if time.Since(halo.lastscanned) > cfgNetworkscanning {
			halo.lastscanned = time.Now()
			KnownNetworks[index] = halo
			go updateHosts(halo)
		}
	}
}

func updateHosts(halo Network) {
	hostok := scanHosts(halo)
	for addr, host := range hostok {
		if val, ok := halo.hostok[addr]; ok == false {
			log.Println("Host found:", addr)
			host.firstseen = time.Now()
			halo.hostok[addr] = host
			host.startSSHClient()
		} else {
			val.openports = host.openports
			val.islocalhost = host.islocalhost
			val.ispreferred = host.ispreferred
			halo.hostok[addr] = val
		}
	}
	for addr := range halo.hostok {
		if _, ok := hostok[addr]; ok == false {
			log.Println("Host down:", addr)
			delete(halo.hostok, addr)
		}
	}
}
func scanHosts(halo Network) map[string]*Host {
	beginning := time.Now()
	var hostok = make(map[string]*Host)
	halo.lastscanned = time.Now()
	networksimplified := halo.network
	networkparts := strings.Split(networksimplified, "/")
	if networkparts[0] == "127.0.0.1" {
		networksimplified = "127.0.0.1/32"
	} else {
		var netmaskslashnew int64 = 24
		netmaskslashoriginal, _ := strconv.ParseInt(networkparts[1], 10, 32)
		if netmaskslashoriginal > 24 {
			netmaskslashnew = netmaskslashoriginal
		}
		networksimplified = networkparts[0] + "/" + strconv.Itoa(int(netmaskslashnew))
	}
	/*	if networksimplified != halo.network {
			log.Println("Network scanning start:", networksimplified)
		} else {
			log.Println("Network scanning (simplified) start:", networksimplified)
		}
	*/
	process := exec.Command("nmap", "-n", "-oG", "-", "-sT", networksimplified, "-p 22,11311")
	output, err := process.CombinedOutput()
	if err != nil {
		log.Printf("Nmap error: %v\n", err)
	}
	//	log.Printf("Output: %v\n", string(output))
	outputlines := strings.Split(string(output), "\n")
	var goodlines []string
	for _, line := range outputlines {
		if !strings.HasPrefix(line, "#") && !strings.Contains(line, "Status: Up") {
			goodlines = append(goodlines, line)
		}
	}
	for _, line := range goodlines {
		lineparts := strings.Split(line, " ")
		if len(lineparts) > 3 {
			var host Host
			//			log.Println("lineparts", line, lineparts)
			host.islocalhost = strings.HasPrefix(line, "Host: "+networkparts[0]+" ")
			host.ispreferred = host.islocalhost && networkparts[0] != "127.0.0.1"
			host.addr = lineparts[1]
			host.props = make(properties)
			host.client.props = make(properties)
			host.openports = make(map[int16]struct{})
			var portcolumn = false
			for _, column := range lineparts {
				//				log.Println("portcolumn", portcolumn, column)
				if strings.HasSuffix(column, ":") {
					portcolumn = strings.Contains(column, "Ports:")
				} else {
					if portcolumn {
						portpart := strings.Split(column, "/")
						port, _ := strconv.Atoi(portpart[0])
						host.openports[int16(port)] = struct{}{}
					}
				}
			}
			hostok[host.addr] = &host
			// log.Printf("host %s\n", host.addr)
		}
	}
	//	log.Println("Hostok: ", hostok)
	log.Println("Network scanning ", networksimplified, "took", time.Since(beginning))
	return hostok
	// log.Printf("%v goodlines\n", goodlines)
}

func (h *Host) startSSHClient() bool {
	_, ok := h.openports[22]
	if !ok || h.client.active {
		return false
	}

	log.Println("SSH client started", h.addr)
	go sshClientWorker(h)
	return true
}

func (h *Host) stoppedSSHClient() bool {
	log.Println("SSH client stopped", h.addr)
	if !h.client.active {
		return false
	}
	h.client.active = false
	return true
}

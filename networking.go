package main

import (
	"log"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Network contains details about a network accessible to the app
type Network struct {
	Iface       string
	NetAddr     string
	FirstSeen   time.Time
	LastScanned time.Time
	Hostok      map[string]*Host
	Props       properties
}

// Host contains details of a known remote host
type Host struct {
	Addr                   string
	IsLocalhost            bool
	IsPreferred            bool
	OpenPorts              map[int16]struct{}
	FirstSeen              time.Time
	LastScanned            time.Time
	LastStaticUpdate       time.Time
	LastDynamicRareUpdate  time.Time
	LastDynamicOftenUpdate time.Time
	ControlClient          HostControlClient
	Props                  properties
	TerminalHistory        []TerminalEvent
}

type properties map[string]string

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
			thisnet := Network{Iface: iface.Name, NetAddr: addrs[0].String(), Hostok: make(map[string]*Host), Props: make(properties)}
			nets[iface.Name] = thisnet
		}
	}
	return nets
}

func updateNetworks() {
	nets := getNetworks()
	for name, net := range nets {
		if val, ok := KnownNetworks[name]; ok == false {
			log.Println("Network found:", name, net.NetAddr)
			net.FirstSeen = time.Now()
			net.LastScanned = time.Unix(0, 0)
			KnownNetworks[name] = net
		} else {
			if val.NetAddr != net.NetAddr {
				log.Println("Network changed:", name, val.NetAddr, net.NetAddr)
				net.FirstSeen = time.Now()
				net.LastScanned = time.Unix(0, 0)
				KnownNetworks[name] = net
			}
		}
	}
	for name, net := range KnownNetworks {
		if _, ok := nets[name]; ok == false {
			log.Println("Network gone:", name, net.NetAddr)
			delete(KnownNetworks, name)
		}
	}
}

func scanNetworks() {
	for index, halo := range KnownNetworks {
		if time.Since(halo.LastScanned) > cfgNetworkscanning {
			halo.LastScanned = time.Now()
			KnownNetworks[index] = halo
			go updateHosts(halo)
		}
	}
}

func updateHosts(halo Network) {
	hostok := scanHosts(halo)
	for addr, host := range hostok {
		if val, ok := halo.Hostok[addr]; ok == false {
			log.Println("Host found:", addr)
			host.FirstSeen = time.Now()
			halo.Hostok[addr] = host
			KnownHosts[addr] = *host
			host.startSSHClient()
		} else {
			val.OpenPorts = host.OpenPorts
			val.IsLocalhost = host.IsLocalhost
			val.IsPreferred = host.IsPreferred
			halo.Hostok[addr] = val
			KnownHosts[addr] = *val
		}
	}
	for addr := range halo.Hostok {
		if _, ok := hostok[addr]; ok == false {
			log.Println("Host down:", addr)
			delete(halo.Hostok, addr)
			delete(KnownHosts, addr)
		}
	}
}
func scanHosts(halo Network) map[string]*Host {
	beginning := time.Now()
	var hostok = make(map[string]*Host)
	halo.LastScanned = time.Now()
	networksimplified := halo.NetAddr
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
			host.IsLocalhost = strings.HasPrefix(line, "Host: "+networkparts[0]+" ")
			host.IsPreferred = host.IsLocalhost && networkparts[0] != "127.0.0.1"
			host.Addr = lineparts[1]
			host.Props = make(properties)
			host.ControlClient.chCommand = make(chan string, 10)
			host.ControlClient.chQuit = make(chan bool, 10)
			host.ControlClient.chTerminal = make(chan TerminalEvent, 10)
			host.ControlClient.Props = make(properties)
			host.TerminalHistory = make([]TerminalEvent, 0, 10)
			host.OpenPorts = make(map[int16]struct{})
			var portcolumn = false
			for _, column := range lineparts {
				//				log.Println("portcolumn", portcolumn, column)
				if strings.HasSuffix(column, ":") {
					portcolumn = strings.Contains(column, "Ports:")
				} else {
					if portcolumn {
						portpart := strings.Split(column, "/")
						port, _ := strconv.Atoi(portpart[0])
						host.OpenPorts[int16(port)] = struct{}{}
					}
				}
			}
			hostok[host.Addr] = &host
			// log.Printf("host %s\n", host.addr)
		}
	}
	//	log.Println("Hostok: ", hostok)
	log.Println("Network scanning ", networksimplified, "took", time.Since(beginning))
	return hostok
	// log.Printf("%v goodlines\n", goodlines)
}

func (h *Host) startSSHClient() bool {
	_, ok := h.OpenPorts[22]
	if !ok || h.ControlClient.Active {
		return false
	}

	log.Println("SSH client started", h.Addr)
	go h.sshClientWorker()
	return true
}

func (h *Host) stoppedSSHClient() bool {
	log.Println("SSH client stopped", h.Addr)
	if !h.ControlClient.Active {
		return false
	}
	h.ControlClient.Active = false
	return true
}

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
	network     string
	firstseen   time.Time
	lastscanned time.Time
	hostok      map[string]Host
}

// Host contains details of a known remote host
type Host struct {
	addr        string
	rosroles    string
	islocalhost bool
	ispreferred bool
	openports   []int16
	firstseen   time.Time
	lastscanned time.Time
}

// KnownNetworks contains all the Networks we can access
var KnownNetworks = make(map[string]Network)

// KnownHosts containt all the Hosts we can access
var KnownHosts = make(map[string]Host)

func getNetworks() map[string]Network {

	nets := make(map[string]Network)
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		// fmt.Printf("%v \n", iface.Flags)
		addrs, _ := iface.Addrs()
		if len(addrs) > 0 && (iface.Flags&net.FlagUp) > 0 {
			// fmt.Printf("%s : %s\n", iface.Name, addrs[0])
			thisnet := Network{network: addrs[0].String(), hostok: make(map[string]Host)}
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
func scanHosts(halo Network) map[string]Host {
	beginning := time.Now()
	var hostok = make(map[string]Host)
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
			var portcolumn = false
			for _, column := range lineparts {
				//				log.Println("portcolumn", portcolumn, column)
				if strings.HasSuffix(column, ":") {
					portcolumn = strings.Contains(column, "Ports:")
				} else {
					if portcolumn {
						portpart := strings.Split(column, "/")
						port, _ := strconv.Atoi(portpart[0])
						host.openports = append(host.openports, int16(port))
					}
				}
			}
			hostok[host.addr] = host
			// log.Printf("host %s\n", host.addr)
		}
	}
	//	log.Println("Hostok: ", hostok)
	log.Println("Network scanning ", networksimplified, "took", time.Since(beginning))
	return hostok
	// log.Printf("%v goodlines\n", goodlines)
}

// CommandRunner runs a command and returns with it's output later
func CommandRunner(stdin string, timeout time.Duration, cmd string, arg ...string) (stdout []string, stderr []string, err error) {
	// process := exec.Command(cmd, arg...)

	/*	procstdin, err := process.StdinPipe()
			if err != nil {
				return nil, nil, err
			}
			procstdout, err := process.StdoutPipe()
			if err != nil {
				return nil, nil, err
			}
			procstderr, err := process.StderrPipe()
			if err != nil {
				return nil, nil, err
			}
			process.Start()
			procstdin.Write([]byte(stdin))
			procstdin.Close()
			stdoutbytes, err := ioutil.ReadAll(procstdout)
			if err != nil {
				return nil, nil, err
			}
			stdout = strings.Split(string(stdoutbytes), "\n")
			stderrbytes, err := ioutil.ReadAll(procstderr)
			if err != nil {
				return stdout, nil, err
			}
			stderr = strings.Split(string(stderrbytes), "\n")

			process.Wait()
		process.Stdin = stdin
		process.Run()
		stdout = process.StdoutPipe()
		stderr = process.Stderr
		return stdout, stderr, nil
	*/
	return nil, nil, nil
}

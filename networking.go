package main

import (
	"log"
	"net"
	"os/exec"
	"time"
)

// Network contains details about a network accessible to the app
type Network struct {
	network     string
	firstseen   time.Time
	lastscanned time.Time
}

// KnownNetworks contains all the Networks we can access
var KnownNetworks = make(map[string]Network)

func getNetworks() map[string]Network {

	nets := make(map[string]Network)
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		// fmt.Printf("%v \n", iface.Flags)
		addrs, _ := iface.Addrs()
		if len(addrs) > 0 && (iface.Flags&net.FlagUp) > 0 {
			// fmt.Printf("%s : %s\n", iface.Name, addrs[0])
			thisnet := Network{network: addrs[0].String()}
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

func scanHosts(halo Network) {
	process := exec.Command("nmap", "-n", "-oG -", "-sT", halo.network, "-p 22,11311")
	process.Run()
	process.CombinedOutput()
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

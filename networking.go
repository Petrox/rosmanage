package main

import "os/exec"
import "time"
import "net"

func GetInterfaces() {
	ifaces, err := net.Interfaces()
	println(ifaces)
}

// CommandRunner runs a command and returns with it's output later
func CommandRunner(stdin string, timeout time.Duration, cmd string, arg ...string) (stdout []string, stderr []string, err error) {
	process := exec.Command(cmd, arg...)

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
	*/
	process.Stdin = stdin
	process.Run()
	stdout = process.StdoutPipe()
	stderr = process.Stderr
	return stdout, stderr, nil
}

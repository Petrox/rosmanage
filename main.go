package main

import "time"

const cfgInterfacepolling = time.Second * 2
const cfgNetworkscanning = time.Second * 10
const cfgSSHRetry = time.Second * 300

/*type Host struct {
	IP        string
	Name      string
	Ports     []int16
	Ping      int32
	Firstseen int32
	Lastseen  int32
	Role      string
}*/

// var Render = render.New(render.Options{IsDevelopment: true})

func main() {
	go networkmain()
	httpmain()
}

/*
func host(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	h := Host{IP: "ip"}
	Render.HTML(rw, 200, "host", h)
	//	h := Host{IP: "ip", Name: "name", Ports: []int16{22, 11311}, Ping: 123, Firstseen: 111, Lastseen: 222, Role: "master"}

}

func hello(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	fmt.Fprintln(rw, "Hello "+p.ByName("name"))
}
*/

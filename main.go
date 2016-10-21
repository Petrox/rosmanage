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
	updateNetworks()
	interfaceticker := time.NewTicker(cfgInterfacepolling)
	go func() {
		for range interfaceticker.C {
			updateNetworks()
			scanNetworks()
		}
	}()

	for {
		time.Sleep(time.Second)
	}
	// stdout, stderr, err := CommandRunner("hello\nhellopetros\nhi", time.Second, "grep", "hello")

	// fmt.Printf("stdout: %v\n stderr: %v\nerr: %v\n", stdout, stderr, err)
	/*
		r := httprouter.New()
		r.GET("/hello/:name", hello)
		r.GET("/host", host)
		n := negroni.Classic()
		n.UseHandler(r)
		n.Run(":3001")*/
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

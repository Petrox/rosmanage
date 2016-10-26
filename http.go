package main

import (
	"log"
	"net/http"
	"text/template"

	"github.com/julienschmidt/httprouter"
)

type dashDataStruct struct {
	Handler     string
	Selected    string
	SelectedObj interface{}
	Params      httprouter.Params
	Networks    map[string]*Network
	Hosts       map[string]*Host
	Checks      map[string]*Check
}

var htmlTemplates *template.Template
var dashTemplate *template.Template

func getDashboard(handlername string, p httprouter.Params) dashDataStruct {
	var dashData dashDataStruct

	dashTemplate, _ = template.New("dashboardlabel").ParseFiles("templates/dashboard.gotmpl")

	dashData.Networks = KnownNetworks
	dashData.Checks = KnownChecks
	dashData.Hosts = make(map[string]*Host)
	for _, halo := range KnownNetworks {
		for name, host := range halo.Hostok {
			dashData.Hosts[name] = host
		}
	}
	dashData.Handler = handlername
	dashData.Params = p
	return dashData
}

func webindex(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	dashData := getDashboard("webindex", p)
	log.Println("http", "webindex", r.RemoteAddr)
	dashTemplate.ExecuteTemplate(w, "dashboard", dashData)
}

func webnet(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	dashData := getDashboard("webnet", p)
	dashData.Selected = p.ByName("iface")
	var ok bool
	log.Println("http", "webnet", r.RemoteAddr, p)

	dashData.SelectedObj, ok = KnownNetworks[dashData.Selected]
	if !ok {
		panic("Not found " + dashData.Selected)
	}
	err := dashTemplate.ExecuteTemplate(w, "webnet", dashData)
	if err != nil {
		panic(err)
	}
}

func webhost(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	dashData := getDashboard("webhost", p)
	log.Println("http", "webhost", r.RemoteAddr)
	dashData.Selected = p.ByName("addr")
	var ok bool
	dashData.SelectedObj, ok = KnownHosts[dashData.Selected]
	log.Println(KnownHosts[dashData.Selected].TerminalHistory)
	if !ok {
		panic("Not found " + dashData.Selected)
	}
	err := dashTemplate.ExecuteTemplate(w, "webhost", dashData)
	if err != nil {
		panic(err)
	}
}

/*func webhello(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Fprintf(w, "hello, %s!\n", ps.ByName("name"))
}*/

/*var FuncMap = template.FuncMap{
	"eq": func(a, b interface{}) bool {
		return a == b
	},
}*/

func httpmain() {
	var err error
	dashTemplate, err = template.New("dashboardlabel").ParseFiles("templates/dashboard.gotmpl")
	if err != nil {
		log.Fatalln("template parsing error", err)
	}
	router := httprouter.New()
	router.GET("/", webindex)
	router.GET("/net/:iface", webnet)
	router.GET("/host/:addr", webhost)
	router.POST("/terminal/:addr", webterminal)
	router.ServeFiles("/static/*filepath", http.Dir("static/"))

	log.Fatal(http.ListenAndServe(":8080", router))
}

package main

import (
	"fmt"
	"log"
	"net/http"
	"text/template"

	"github.com/julienschmidt/httprouter"
)

type dashDataStruct struct {
	Revision int
	Networks map[string]Network
	Hosts    map[string]Host
}

var htmlTemplates *template.Template
var dashTemplate *template.Template

func webindex(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var dashData dashDataStruct
	log.Println("http", "webindex", r.RemoteAddr)
	dashData.Revision = 1
	dashData.Hosts = KnownHosts
	dashData.Networks = KnownNetworks
	// dashTemplate.Execute(w, dashData)
	dashTemplate.ExecuteTemplate(w, "dashboard", dashData)
}

func webhello(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Fprintf(w, "hello, %s!\n", ps.ByName("name"))
}

func httpmain() {
	var err error
	dashTemplate, err = template.New("dashboardlabel").ParseFiles("templates/dashboard.gotmpl")
	if err != nil {
		log.Fatalln("template parsing error", err)
	}
	router := httprouter.New()
	router.GET("/", webindex)
	router.GET("/hello/:name", webhello)

	log.Fatal(http.ListenAndServe(":8080", router))
}

package main

import (
	"fmt"
	"net/http"

	"github.com/codegangsta/negroni"
	"github.com/julienschmidt/httprouter"
)

func main() {
	r := httprouter.New()
	r.GET("/hello/:name", hello)
	n := negroni.Classic()
	n.UseHandler(r)
	n.Run(":3001")
}

func hello(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	fmt.Fprintln(rw, "Hello "+p.ByName("name"))
}

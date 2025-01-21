package main

import (
	"log"
	"net/http"

	"linkernetworks.com/opendcos_deploy/services"

	"github.com/emicklei/go-restful"
)

func main() {

	err := services.DownloadInstaller()
	if err != nil {
		log.Fatal("DownloadInstaller failed, err is " + err.Error())
	}
	wsContainer := restful.NewContainer()
	u := services.Resource{}
	u.Register(wsContainer)

	log.Printf("start listening on localhost:28080")
	server := &http.Server{Addr: ":28080", Handler: wsContainer}
	log.Fatal(server.ListenAndServe())
}

package main

import (
	"flag"
	"log"

	"linkernetworks.com/dcos-backend/autoscaling/metrics-collector/api"
	"linkernetworks.com/dcos-backend/autoscaling/metrics-collector/daemon"

	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
	"gopkg.in/kataras/iris.v6/middleware/pprof"
)

func main() {
	// start metrics daemon in background
	// this daemon will retrieve cAdvisor metrics continuously in background
	var errCh = make(chan *error)
	go startMetricsDaemon(errCh)

	go func() {
		e := <-errCh
		log.Fatalf("start daemon instance error: %v\n", *e)
	}()

	// start HTTP server
	var enablePProf bool
	flag.BoolVar(&enablePProf, "pprof", false, "enable web pprof, default false.")
	flag.Parse()
	if enablePProf {
		log.Printf("web pprof enabled")
	}

	startHttpServer(enablePProf)
}

func startMetricsDaemon(errCh chan *error) {
	log.Printf("== starting metrics daemon\n")
	err := daemon.StartDaemonInstance()
	if err != nil {
		errCh <- &err
		return
	}
}

func startHttpServer(enablePProf bool) {
	app := iris.New()

	app.Adapt(
		iris.DevLogger(),
		httprouter.New())

	app.Get("/", api.GetRoot)
	app.Get("/metrics", api.GetMetrics)

	if enablePProf {
		app.Get("/debug/pprof/*action", pprof.New())
	}

	app.Listen(":10005")
}

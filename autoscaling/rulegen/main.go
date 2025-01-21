package main

import (
	"fmt"
	"log"

	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
	"linkernetworks.com/dcos-backend/autoscaling/rulegen/api"
	"linkernetworks.com/dcos-backend/autoscaling/rulegen/consts"
	"linkernetworks.com/dcos-backend/autoscaling/rulegen/env"
	"linkernetworks.com/dcos-backend/autoscaling/rulegen/midware"
	"linkernetworks.com/dcos-backend/autoscaling/rulegen/runt"
)

func main() {
	// set path
	if err := runt.InitRuleFilePath(); err != nil {
		log.Printf("access rule file error: %v\n", err)
		return
	}

	// Docs of Iris web framework:
	// https://godoc.org/gopkg.in/kataras/iris.v6
	app := iris.New()

	app.Adapt(
		iris.DevLogger(),
		httprouter.New())

	app.UseGlobalFunc(midware.RouterLog)

	app.Put("/rules", api.UpdateRules)

	addr, port := consts.DefaultListenAddr, consts.DefaultListenPort
	envAddr, _ := env.Get(consts.EnvListenAddr).ToString()
	if len(envAddr) > 0 {
		addr = envAddr
	}
	envPort, _ := env.Get(consts.EnvListenPort).ToString()
	if len(envPort) > 0 {
		port = envPort
	}
	app.Listen(fmt.Sprintf("%s:%s", addr, port))
}

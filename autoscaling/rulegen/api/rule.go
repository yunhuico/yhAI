package api

import (
	"log"
	"path/filepath"

	"linkernetworks.com/dcos-backend/autoscaling/rulegen/runt"
	"linkernetworks.com/dcos-backend/autoscaling/rulegen/svc"
	"linkernetworks.com/dcos-backend/autoscaling/rulegen/types"

	iris "gopkg.in/kataras/iris.v6"
)

func UpdateRules(ctx *iris.Context) {
	req, resp := types.ReqPutRules{}, types.RespPutRules{}

	if err := ctx.ReadJSON(&req); err != nil {
		resp.Errmsg = "parse body: " + err.Error()
		ctx.JSON(iris.StatusBadRequest, resp)
		return
	}

	cpuEnabled, memEnabled := req.CPUEnabled, req.MemEnabled
	duration := req.Duration
	cpuHigh, cpuLow := req.Thresholds["cpu_high"], req.Thresholds["cpu_low"]
	memHigh, memLow := req.Thresholds["mem_high"], req.Thresholds["mem_low"]

	if cpuEnabled || memEnabled {
		// duration is required when CPU or Memory monitor enabled
		if req.Duration == "" {
			resp.Errmsg = "'duration' not set"
			ctx.JSON(iris.StatusBadRequest, resp)
			return
		}
	}

	var ruleFile = runt.RuleFilePath
	err := svc.UpdateRules(cpuEnabled, memEnabled, cpuHigh, cpuLow, memHigh, memLow, duration, ruleFile)
	if err != nil {
		resp.Errmsg = "update rule: " + err.Error()
		ctx.JSON(iris.StatusInternalServerError, resp)
		return
	}

	fullPath, _ := filepath.Abs(ruleFile)
	log.Printf("rule file(%s) updated\n", fullPath)
	resp.Success = true
	ctx.JSON(iris.StatusOK, resp)
	return
}

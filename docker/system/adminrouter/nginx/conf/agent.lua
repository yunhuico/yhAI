local util = require "common"

local state = util.mesos_get_state()
if state == nil then
    ngx.status = ngx.HTTP_SERVICE_UNAVAILABLE
    ngx.say("503 Service Unavailable: invalid Mesos state.")
    return ngx.exit(ngx.HTTP_SERVICE_UNAVAILABLE)
end
for _, agent in ipairs(state["slaves"]) do
    if agent["id"] == ngx.var.agentid then
        local split_pid = agent["pid"]:split("@")
        local host_port = split_pid[2]:split(":")
        ngx.var.agentaddr = host_port[1]
        ngx.var.agentport = host_port[2]

        ngx.log(
            ngx.DEBUG, "agentid / agentaddr:" ..
            ngx.var.agentid .. " / " .. ngx.var.agentaddr
            )
        return
    end
end
ngx.status = ngx.HTTP_NOT_FOUND
ngx.say("404 Not Found: agent " .. ngx.var.agentid .. " unknown.")
return ngx.exit(ngx.HTTP_NOT_FOUND)
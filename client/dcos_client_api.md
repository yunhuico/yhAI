[TOC]

## 一. 服务管理
### 创建空服务
REQUEST
```
Url :
POST  /v1/appsets
Body{}
```

Example：
```
POST http://54.250.151.32:10004/v1/appsets
Body
{
"name":"test",
"description":"the description for you group",
"created_by_json":false,
"group":{}
}
```

### 创建服务
REQUEST
```
Url :
POST  /v1/appsets
Body{}
```

Example:
```
POST http://54.250.151.32:10004/v1/appsets
Body
{
    "name": "nginxjson",
    "created_by_json": true,
    "description": "",
    "group": {
        "id": "/nginxjson",
        "apps": [
            {
                "id": "/nginxjson/nginx",
                "container": {
                    "type": "DOCKER",
                    "docker": {
                        "forcePullImage": true,
                        "image": "docker.io/nginx",
                        "network": "BRIDGE",
                        "portMappings": [
                            {
                                "containerPort": 80,
                                "hostPort": 0,
                                "servicePort": 10099,
                                "protocol": "tcp"
                            }
                        ],
                        "privileged": true
                    }
                },
                "cpus": 0.1,
                "env": {},
                "instances": 2,
                "mem": 128,
                "ports": [],
                "dependencies": [],
                "labels": {
                    "HAPROXY_GROUP": "linkermgmt"
                }
            }
        ],
        "dependencies": [],
        "groups": []
    }
}
```
### 查询服务概览
REQUEST
```
Url :
GET /v1/appsets
参数：
/v1/appsets?count=true
&skip=10
&limit=5
&sort=time_update
```

RESPONSE
Example：
```
GET http://54.250.151.32:10004/v1/appsets?limit=2&count=true&sort=time_update
RESPONSE
{
  "success": true,
  "count": 2,
  "data": [
    {
      "_id": "58b78a8dad37720031391d94",
      "name": "test",
      "status": "IDLE",
      "description": "the description for you group",
      "template_id": "test",
      "created_by_json": false,
      "time_deployed": "0001-01-01T00:00:00Z",
      "time_create": "2017-03-02T02:59:25Z",
      "time_update": "2017-03-02T02:59:25Z"
    },
    {
      "_id": "58b792f9ad37720031391d96",
      "name": "nginxjson",
      "status": "IDLE",
      "description": "",
      "template_id": "nginxjson",
      "created_by_json": true,
      "time_deployed": "0001-01-01T00:00:00Z",
      "time_create": "2017-03-02T03:35:21Z",
      "time_update": "2017-03-02T03:35:21Z"
    }
  ]
}
```
### 查询服务详情
REQUEST
```
URL :
GET /v1/appsets/{name}
参数：
/v1/appsets/nginxjson?skip_group=true
/v1/appsets/nginxjson?monitor=true
```

如果参数monitor设置为true，则在component下将增加monitor_url_map字段，里面含有组件下每个容器（在Marathon/Mesos称task）的cAdvisor监控地址。结构是个Map，键为Task的ID，值为容器监控数据的URL，里面的IP为主机的内网地址。
即：
```
"components": [
            {
                "appset_name": "nginxjson",
                "total_cpu": 0.2,
                "can_modify": true,
                "can_scale": false,
                "can_shell": false,
        "monitor_url_map": {
          "nginx_nginx2.87510b9b-3136-11e7-8e5c-eeab8a06c4af": "http://192.168.7.73:10000/api/v1.2/docker/mesos-bf2b618e-15f0-4c88-8193-120f29b5d7a0-S2.2870a575-9c21-49c6-a4b6-9c549f355f33",
          "nginx_nginx2.8d5b10dd-3136-11e7-8e5c-eeab8a06c4af": "http://192.168.7.72:10000/api/v1.2/docker/mesos-bf2b618e-15f0-4c88-8193-120f29b5d7a0-S1.c7e2c784-2cbb-4cdd-9444-8bdd5d29e0c2",
          "nginx_nginx2.971525cf-3136-11e7-8e5c-eeab8a06c4af": "http://192.168.7.73:10000/api/v1.2/docker/mesos-bf2b618e-15f0-4c88-8193-120f29b5d7a0-S2.9ac34c0a-564d-404d-954a-2d7019c4ed97"
        },
"app": {
					// …
				}
}
]
```

RESPONSE
Example：
```
GET http://54.250.151.32:10004/v1/appsets/nginxjson?skip_group=true
RESPONSE
{
    "success": true,
    "data": {
        "_id": "58b79398ad37720031391d97",
        "name": "nginxjson",
        "status": "IDLE",
        "description": "",
        "template_id": "nginxjson",
        "total_cpu": 0.2,
        "total_mem": 256,
        "total_container": 2,
        "total_host": 0,
        "running_cpu": 0,
        "running_mem": 0,
        "running_container": 0,
        "created_by_json": true,
        "group": {
            "id": "",
            "apps": null,
            "dependencies": null,
            "groups": null
        },
        "components": [
            {
                "appset_name": "nginxjson",
                "total_cpu": 0.2,
                "total_mem": 256,
                "status": "IDLE",
                "can_delete": true,
                "can_stop": false,
                "can_start": true,
                "can_modify": true,
                "can_scale": false,
                "can_shell": false,
                "app": {
                    "id": "/nginxjson/nginx",
                    "container": {
                        "type": "DOCKER",
                        "docker": {
                            "forcePullImage": true,
                            "image": "docker.io/nginx",
                            "network": "BRIDGE",
                            "portMappings": [
                                {
                                    "containerPort": 80,
                                    "hostPort": 0,
                                    "servicePort": 10099,
                                    "protocol": "tcp"
                                }
                            ],
                            "privileged": true
                        }
                    },
                    "cpus": 0.1,
                    "env": {},
                    "instances": 2,
                    "mem": 128,
                    "ports": [],
                    "dependencies": [],
                    "labels": {
                        "HAPROXY_GROUP": "linkermgmt"
                    }
                }
            }
        ],
        "time_deployed": "0001-01-01T00:00:00Z",
        "time_create": "2017-03-02T03:38:00Z",
        "time_update": "2017-03-02T03:38:00Z"
    }
}
```

### 启动服务
REQUEST
```
Url :
PUT /v1/appsets/{name}/start
```

Example:
```
PUT http://54.250.151.32:10004/v1/appsets/nginxjson/start
RESPONSE
{
"success":true
}
```


### 停止服务
REQUEST
```
Url :
PUT /v1/appsets/{name}/stop
```

Example:
```
PUT http://54.250.151.32:10004/v1/appsets/nginxjson/stop
RESPONSE
{
"success":true
}
```


### 删除服务
REQUEST
```
Url :
DELETE /v1/appsets/{name}
```
Example:
```
DELETE http://54.250.151.32:10004/v1/appsets/nginxjson
RESPONSE
{
"success":true
}
```

### 更新服务
REQUEST
```
Url :
PUT /v1/appsets/{name}
Body{}
```

Example：
```
PUT http://54.250.151.32:10004/v1/appsets/nginxjson
Body
{
    "name": "nginxjson",
    "created_by_json": true,
    "description": "",
    "group": {
        "id": "/nginxjson",
        "apps": [
            {
                "id": "/nginxjson/nginx",
                "container": {
                    "type": "DOCKER",
                    "docker": {
                        "forcePullImage": true,
                        "image": "docker.io/nginx",
                        "network": "BRIDGE",
                        "portMappings": [
                            {
                                "containerPort": 80,
                                "hostPort": 0,
                                "servicePort": 10099,
                                "protocol": "tcp"
                            }
                        ],
                        "privileged": true
                    }
                },
                "cpus": 0.2,
                "env": {},
                "instances": 1,
                "mem": 128,
                "ports": [],
                "dependencies": [],
                "labels": {
                    "HAPROXY_GROUP": "linkermgmt"
                }
            }
        ],
        "dependencies": [],
        "groups": []
    }
}
```

## 二. 组件管理
### 新增组件
REQUEST
```
Url :
POST /v1/components
Body{}
```

Example:
```
POST http://54.250.151.32:10004/v1/components
Body
{
    "appset_name": "test",
    "app": {
        "id": "busybox",
        "container": {
            "type": "DOCKER",
            "docker": {
                "forcePullImage": true,
                "image": "docker.io/busybox",
                "network": "BRIDGE",
                "portMappings": [],
                "privileged": false
            }
        },
        "cpus": 0.1,
        "env": {},
        "instances": 2,
        "mem": 128,
        "labels": {}
    }
}
```

### 查询某个组件
REQUEST
```
Url :
GET /v1/components?name=
```

Example:
```
GET http://54.250.151.32:10004/v1/components?name=/nginxjson/nginx
RESPONSE
{
    "success": true,
    "data": {
        "appset_name": "nginxjson",
        "total_cpu": 0.2,
        "total_mem": 256,
        "status": "RUNNING",
        "can_delete": true,
        "can_stop": true,
        "can_start": false,
        "can_modify": true,
        "can_scale": true,
        "can_shell": true,
        "app": {
            "id": "/nginxjson/nginx",
            "constraints": [],
            "container": {
                "type": "DOCKER",
                "docker": {
                    "forcePullImage": true,
                    "image": "docker.io/nginx",
                    "network": "BRIDGE",
                    "parameters": [],
                    "portMappings": [
                        {
                            "containerPort": 80,
                            "hostPort": 0,
                            "labels": {},
                            "servicePort": 10099,
                            "protocol": "tcp"
                        }
                    ],
                    "privileged": true
                },
                "volumes": []
            },
            "cpus": 0.1,
            "disk": 0,
            "env": {
                "APPSET_OBJ_ID": "58b7b3b3ad37720031391d9a",
                "LINKER_APP_ID": "/nginxjson/nginx",
                "LINKER_GROUP_ID": "/nginxjson",
                "LINKER_REPAIR_TEMPLATE_ID": "nginxjson"
            },
            "executor": "",
            "healthChecks": [],
            "instances": 2,
            "mem": 128,
            "tasks": [
                {
                    "id": "nginxjson_nginx.c8ca9993-ff0c-11e6-a925-5e0585ecca91",
                    "appId": "/nginxjson/nginx",
                    "host": "172.31.26.243",
                    "healthCheckResults": null,
                    "ports": [
                        12790
                    ],
                    "servicePorts": null,
                    "slaveId": "0e5afd2b-2b45-4482-9968-c0e43c3ca4ca-S0",
                    "stagedAt": "2017-03-02T05:55:05.479Z",
                    "startedAt": "2017-03-02T05:55:08.336Z",
                    "ipAddresses": [
                        {
                            "ipAddress": "172.17.0.3",
                            "protocol": "IPv4"
                        }
                    ],
                    "version": "2017-03-02T05:55:05.453Z"
                },
                {
                    "id": "nginxjson_nginx.c8ca7282-ff0c-11e6-a925-5e0585ecca91",
                    "appId": "/nginxjson/nginx",
                    "host": "172.31.26.243",
                    "healthCheckResults": null,
                    "ports": [
                        11777
                    ],
                    "servicePorts": null,
                    "slaveId": "0e5afd2b-2b45-4482-9968-c0e43c3ca4ca-S0",
                    "stagedAt": "2017-03-02T05:55:05.478Z",
                    "startedAt": "2017-03-02T05:55:08.435Z",
                    "ipAddresses": [
                        {
                            "ipAddress": "172.17.0.4",
                            "protocol": "IPv4"
                        }
                    ],
                    "version": "2017-03-02T05:55:05.453Z"
                }
            ],
            "ports": [
                10099
            ],
            "portDefinitions": [],
            "requirePorts": false,
            "backoffSeconds": 1,
            "backoffFactor": 1.15,
            "maxLaunchDelaySeconds": 3600,
            "dependencies": [],
            "tasksRunning": 2,
            "upgradeStrategy": {
                "minimumHealthCapacity": 1,
                "maximumOverCapacity": 1
            },
            "uris": [],
            "version": "2017-03-02T05:55:05.453Z",
            "versionInfo": {
                "lastScalingAt": "2017-03-02T05:55:05.453Z",
                "lastConfigChangeAt": "2017-03-02T05:55:05.453Z"
            },
            "labels": {
                "HAPROXY_GROUP": "linkermgmt"
            },
            "fetch": []
        }
    }
}
```

### 调节组件数目
REQUEST
```
Url :
PUT /v1/components/scale
参数
/v1/components/scale?name=
&scaleto=3
```

Example:
```
PUT http://54.250.151.32:10004/v1/components/scale?name=/nginxjson/nginx&scaleto=1
（将nginxjson服务中的nginx组件数目调节至1）
RESPONSE
{
"success":true
}
```

### 停止组件
REQUEST
```
Url :
PUT /v1/components/stop
参数
/v1/components/stop?name=
```

Example：
```
PUT http://54.250.151.32:10004/v1/components/stop?name=/nginxjson/nginx
RESPONSE
{
"success":true
}
```

### 启动组件
REQUEST
```
Url :
PUT /v1/components/start
参数
/v1/components/start?name=
```

Example:
```
PUT http://54.250.151.32:10004/v1/components/start?name=/nginxjson/nginx
RESPONSE
{
"success":true
}
```

### 更新组件
REQUEST
```
Url :
PUT /v1/components
Body{}
```

Example:
```
PUT http://54.250.151.32:10004/v1/components
Body
{
    "appset_name": "nginxjson",
    "app": {
        "id": "/nginxjson/nginx",
        "constraints": [],
        "container": {
            "type": "DOCKER",
            "docker": {
                "forcePullImage": true,
                "image": "docker.io/nginx",
                "network": "BRIDGE",
                "parameters": [],
                "portMappings": [
                    {
                        "containerPort": 80,
                        "hostPort": 0,
                        "labels": {},
                        "servicePort": 10099,
                        "protocol": "tcp"
                    }
                ],
                "privileged": true
            },
            "volumes": []
        },
        "cpus": 0.1,
        "disk": 0,
        "env": {
            "APPSET_OBJ_ID": "58b7b3b3ad37720031391d9a",
            "LINKER_APP_ID": "/nginxjson/nginx",
            "LINKER_GROUP_ID": "/nginxjson",
            "LINKER_REPAIR_TEMPLATE_ID": "nginxjson"
        },
        "executor": "",
        "healthChecks": [],
        "instances": 2,
        "mem": 128
    }
}
```

### 删除组件
REQUEST
```
Url :
DELETE /v1/components
参数
/v1/components?name=
```

Example:
```
DELETE http://54.250.151.32:10004/v1/components?name=/nginxjson/nginx
RESPONSE
{
"success":true
}
```


## 三. 容器管理
### 查询容器列表
REQUEST
```
Url :
GET /v1/containers
参数:
/v1/containers?count=true
&host_ip=
```

Example:
```
GET http://54.250.151.32:10004/v1/containers?count=true&host_ip=172.31.26.243
RESPONSE
{
  "success": true,
  "count": 2,
  "data": [
    {
      "id": "nginxjson_nginx.11e1b3d7-ff11-11e6-a925-5e0585ecca91",
      "appId": "/nginxjson/nginx",
      "host": "172.31.26.243",
      "healthCheckResults": null,
      "ports": [
        759
      ],
      "servicePorts": null,
      "slaveId": "0e5afd2b-2b45-4482-9968-c0e43c3ca4ca-S0",
      "stagedAt": "2017-03-02T06:25:46.091Z",
      "startedAt": "2017-03-02T06:25:48.928Z",
      "ipAddresses": [
        {
          "ipAddress": "172.17.0.3",
          "protocol": "IPv4"
        }
      ],
      "version": "2017-03-02T06:25:46.063Z",
      "name": "mesos-0e5afd2b-2b45-4482-9968-c0e43c3ca4ca-S0.2b7023d6-0981-4e03-bce6-c41da7fc54df"
    },
    {
      "id": "nginxjson_nginx.11e18cc6-ff11-11e6-a925-5e0585ecca91",
      "appId": "/nginxjson/nginx",
      "host": "172.31.26.243",
      "healthCheckResults": null,
      "ports": [
        16817
      ],
      "servicePorts": null,
      "slaveId": "0e5afd2b-2b45-4482-9968-c0e43c3ca4ca-S0",
      "stagedAt": "2017-03-02T06:25:46.090Z",
      "startedAt": "2017-03-02T06:25:49.025Z",
      "ipAddresses": [
        {
          "ipAddress": "172.17.0.4",
          "protocol": "IPv4"
        }
      ],
      "version": "2017-03-02T06:25:46.063Z",
      "name": "mesos-0e5afd2b-2b45-4482-9968-c0e43c3ca4ca-S0.d4a0e068-ed79-4c30-9cf1-042c3b0c032f"
    }
  ]
}
```

### 重新部署容器
REQUEST
```
Url :
PUT  /v1/containers/{taskId}/redeploy     其中taskId为marathon上的容器id
```

Example：
```
PUT http://54.250.151.32:10004/v1/containers/nginxjson_nginx.7c2b7399-ff14-11e6-a925-5e0585ecca91/redeploy
RESPONSE
{
"success":true
}
```

### 删除容器
REQUEST
```
Url :
PUT  /v1/containers/{taskId}/kill
```

Example：
```
PUT http://54.250.151.32:10004/v1/containers/nginxjson_nginx.7c2b7399-ff14-11e6-a925-5e0585ecca91/kill
RESPONSE
{
"success":true
}
```

## 四. 告警与 Auto-scaling (容器)
### 查询 Alerts 信息

这个 API 用于查询组件 scale in / scale out 的历史记录。一个启用了 “监控告警” 的组件
会进行 scale in、 scale out 操作，或者即使有告警，但因为实例数达到上限或者下限，也什么也没有做。

所以，组件 scale 的记录有下面几种。括号里内容分别是 API 查询结果中 alert_name 和 action 字段的值。

* 因 CPU 高进行了一次 Scale out (HighCpuAlert， REPAIR_ACTION_SUCCESS_OUT)
* CPU 高，没进行 Scale out，因为实例数达到上限 (HighCpuAlert， REPAIR_ACTION_DONOTHING_MAX)
* 因 CPU 低进行了一次 Scale in (LowCpuAlert， REPAIR_ACTION_SUCCESS_IN)
* CPU 低，没进行 Scale in，因为实例数达到下限 (HighCpuAlert， REPAIR_ACTION_DONOTHING_MIN)
* 因 Mem 高进行了一次 Scale out (HighMemoryAlert， REPAIR_ACTION_SUCCESS_OUT)
* Mem 高，没进行 Scale out，因为实例数达到上限 (HighMemoryAlert， REPAIR_ACTION_DONOTHING_MAX)
* 因 Mem 低进行了一次 Scale in (LowMemoryAlert， REPAIR_ACTION_SUCCESS_IN)
* Mem 低，没进行 Scale in，因为实例数达到下限 (HighMemoryAlert， REPAIR_ACTION_DONOTHING_MIN)
* 出现错误（""，REPAIR_ACTION_FAILURE）

**REQUEST**
```
Url :
GET  /v1/alerts?count=true&skip=20&limit=10&...
```

全部 URL 参数

|参数名|默认值|示例值|说明|
|:--|:--|:--|:--|
|count|false|true|是否在 Response 中显示查询的结果数量|
|skip|0|20|跳过前面 20 个结果|
|limit|0|10|限制查询结果最大数量|
|sort|""|time_update/action/app_id|按照 time_update/action/app_id 排序|
|action|""|REPAIR_ACTION_SUCCESS_IN/REPAIR_ACTION_SUCCESS_OUT/REPAIR_ACTION_DONOTHING_MIN/REPAIR_ACTION_DONOTHING_MAX|根据 action 字段筛选结果|
|alert_name|""|HighCpuAlert/LowCpuAlert/HighMemoryAlert/LowMemoryAlert|根据 alert_name 字段筛选结果|
|app_id|""|/servicename/appname|根据 app_id 字段筛选|
|group_id|""|/servicename|根据 group_id 字段筛选|
|donothing|false|true|默认不显示 action 为 REPAIR_ACTION_DONOTHING_MAX、REPAIR_ACTION_DONOTHING_MIN 的结果，如果想要显示设置参数为 true|

**Example**
```
GET http://54.250.151.32:10004/alerts?group_id=/stress2&action=REPAIR_ACTION_SUCCESS_IN&alert_name=LowMemoryAlert&count=true
```

**RESPONSE**
```
{
    "success": true,
    "count": 65,
    "data": [
        {
            "_id": "598d0eb7ead083000108c863",
            "app_id": "/stress2/stress3",
            "alert_name": "LowMemoryAlert",
            "action": "REPAIR_ACTION_SUCCESS_IN",
            "time_update": "2017-08-11T01:56:07Z"
        },
        {
            "_id": "598d0ef3ead083000108c86a",
            "app_id": "/stress2/stress3",
            "alert_name": "LowMemoryAlert",
            "action": "REPAIR_ACTION_SUCCESS_IN",
            "time_update": "2017-08-11T01:57:07Z"
        },
        // ...
    ]
}
```

# 五. 主机监控与告警
这个接口用于监控用户集群 slave 节点（包括 shared-slave）的 CPU 和内存占用率并发送告警邮件。
用户需要在 Linker DC/OS 页面根据填入 CPU 或者内存占用率的上限、下限（即阈值）、持续时间（duration），
这些规则会先发往 rulegen，生成 Prometheus 的规则文件（hostcpumem.rules），接着 dcosclient 调用
 Prometheus 的重载配置接口更新设置，最后存入 MongoDB 以供查询。CPU、内存占用率数据来源于 cAdvisor
 的 /containers 的接口，由 metrics-collector 采集并供以查询。实际采样与检测过程由 Prometheus 完成，
 告警由 AlertManager 完成。当 CPU、内存占用率高于阈值上限，或者低于阈值下限，持续一段时候后（duration），
 AlertManager 会产生告警并发往 dcosclient。dcosclient 收到告警后，根据告警信息和设置的规则，生成
 一封邮件，最后调用 clustermgmt 的接口，发送邮件到当前登录的用户。

|参数|数据类型|说明|
|--|--|--|
|cpu_enabled|bool|启用/禁用 CPU 监控|
|mem_enabled|bool|启用/禁用内存监控|
|duraion|string|持续时间（Unix 时长风格，如 '30s', '5m', '2h30m' 等）|
|threshold|map|包含阈值的结构|
|cpu_high|float|CPU 占用率上限（0-100 之间）|
|cpu_low|float|CPU 占用率下限（0-100 之间）|
|mem_high|float|内存占用率上限（0-100 之间）|
|mem_low|float|内存占用率下限（0-100 之间）|

## Update Host Rules
**Example 1: Enable CPU and Memory monitor**

**Request**
```
PUT /v1/hostrules
Content-Type: application/json
Cache-Control: no-cache

{
	"cpu_enabled": true,
	"mem_enabled": true,
	"duration": "10m",
	"thresholds": {
		"cpu_high": 80,
		"cpu_low": 20,
		"mem_high": 80,
		"mem_low": 20
	}
}
```

**Response**

200 OK

```
{
    "success": true,
    "data": {
        "cpu_enabled": true,
        "mem_enabled": true,
        "duration": "10m",
        "thresholds": {
            "cpu_high": 80,
            "cpu_low": 20,
            "mem_high": 80,
            "mem_low": 20
        },
        "_id": "59fbe9b27117f6003437843e",
        "time_create": "2017-11-03T03:59:46Z",
        "time_update": "2017-11-03T06:08:34Z"
    }
}
```

**Example 2: Enable CPU monitor only**

**Request**

```
{
	"cpu_enabled": true,
	"duration": "10m",
	"thresholds": {
		"cpu_high": 80,
		"cpu_low": 20
	}
}
```

**Response**

```
{
    "success": true,
    "data": {
        "cpu_enabled": true,
        "mem_enabled": false,
        "duration": "10m",
        "thresholds": {
            "cpu_high": 80,
            "cpu_low": 20
        },
        "_id": "59fbe9b27117f6003437843e",
        "time_create": "2017-11-03T03:59:46Z",
        "time_update": "2017-11-03T06:09:32Z"
    }
}
```

**Example 3: Enable Memory monitor only**

**Request**

```
PUT /v1/hostrules
Content-Type: application/json
Cache-Control: no-cache

{
    "mem_enabled": true,
	"duration": "10m",
	"thresholds": {
		"mem_high": 80,
		"mem_low": 20
	}
}
```

**Response**

```
{
    "success": true,
    "data": {
        "cpu_enabled": false,
        "mem_enabled": true,
        "duration": "10m",
        "thresholds": {
            "mem_high": 80,
            "mem_low": 20
        },
        "_id": "59fbe9b27117f6003437843e",
        "time_create": "2017-11-03T03:59:46Z",
        "time_update": "2017-11-03T06:10:17Z"
    }
}
```

**Example 4: Disable CPU and Memory monitor**

**Request**

```
PUT /v1/hostrules
Content-Type: application/json
Cache-Control: no-cache

{
    "cpu_enabled": false,
    "mem_enabled": false
}
```

**Response**

```
{
    "success": true,
    "data": {
        "cpu_enabled": false,
        "mem_enabled": false,
        "duration": "",
        "thresholds": {},
        "_id": "59fbe9b27117f6003437843e",
        "time_create": "2017-11-03T03:59:46Z",
        "time_update": "2017-11-03T06:10:17Z"
    }
}
```

## Get Host Rules

**Request**

```
GET /v1/hostrules
Content-Type: application/json
Cache-Control: no-cache
```

**Response**

200 OK

```
{
    "success": true,
    "data": {
        "cpu_enabled": true,
        "mem_enabled": true,
        "duration": "10m",
        "thresholds": {
            "cpu_high": 80,
            "cpu_low": 20,
            "mem_high": 80,
            "mem_low": 20
        },
        "_id": "59fbe9b27117f6003437843e",
        "time_create": "2017-11-03T03:59:46Z",
        "time_update": "2017-11-03T03:59:46Z"
    }
}
```

**Response ( on error )**

*500 Internal Server Error*

```
{
  "success": false,
  "error": {
   "code": "E55002",
   "errormsg": "not found"
  }
 }
```

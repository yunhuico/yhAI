[TOC]


# 一 创建集群
REQUEST

```
Url:
Header: X_Auth_Token
POST v1/cluster
Body {}
```

EXAMPLE

```
curl -X POST -H "Content-Type: application/json" -H "X-Auth-Token: 59c1d404403cba000a11635e" http://192.168.10.211:10002/v1/cluster -d @createUserCluster.json
```

其中createUserCluster.json示例如下：

```
{
    "name": "cluster",
    "owner": "sysadmin",
    "masterCount": 1,
    "sharedCount": 1,
    "pureslaveCount": 0,
    "user_id": "59bb4331403cba000a116359",
    "masterNodes": [
        {
            "ip": "192.168.10.212",
            "sshUser": "root",
            "privateKey": "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",
            "privateNicName": "eth0"
        }
    ],
    "sharedNodes": [
        {
            "ip": "192.168.10.213",
            "sshUser": "root",
            "privateKey": "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",
            "privateNicName": "eth0"
        }
    ],
    "type": "customized",
    "createCategory": "compact",
    "dockerRegistries": [
        {
            "_id": "59bb43f8284ae3000bf663e7",
            "name": "registry_name",
            "registry": "192.168.10.95:5000",
            "isSystemRegistry": true
        }
    ]
}
```

RESPONSE：

```
{
    "success": true,
    "data": {
        "_id": "59bb43f8284ae3000bf663ea",
        "name": "cluster",
        "owner": "sysadmin",
        "endPoint": "",
        "instances": 2,
        "pubkeyId": null,
        "providerId": "",
        "details": "",
        "status": "INSTALLING",
        "type": "customized",
        "createCategory": "compact",
        "user_id": "59bb4331403cba000a116359",
        "tenant_id": "59bb4330403cba000a116355",
        "time_create": "2017-09-15T03:07:36Z",
        "time_update": "2017-09-15T03:07:36Z",
        "dockerRegistries": [
            {
                "_id": "59bb43f8284ae3000bf663e7",
                "name": "linkerRegistry",
                "registry": "192.168.10.95:5000",
                "secure": false,
                "ca_text": "",
                "username": "",
                "password": "",
                "user_id": "",
                "tenant_id": "",
                "isUse": false,
                "isSystemRegistry": true,
                "TimeCreate": "0001-01-01T00:00:00Z"
            }
        ],
        "setProjectvalue": {
            "cmi": false
        }
    }
}
```


# 二 查询所有集群

REQUEST

```
Url :
Header: X_Auth_Token
GET /v1/cluster
参数
/v1/cluster?&count=true
&skip=
&name= (cluster'name)
&limit=
&sort=
&user_id=
&username=
&status=
```

EXAMPLE:

```
curl -X GET -H "Content-Type: application/json" -H "X-Auth-Token: 59c1d404403cba000a11635e" http://localhost:10002/v1/cluster?count=true&user_id=59bb4331403cba000a116359
```

RESPONSE

```
{
  "success": true,
  "count": 1,
  "data": [
   {
    "_id": "59bb43f8284ae3000bf663ea",
    "name": "cluster",
    "owner": "sysadmin",
    "endPoint": "192.168.10.212",
    "instances": 3,
    "pubkeyId": [],
    "providerId": "",
    "details": "",
    "status": "RUNNING",
    "type": "customized",
    "createCategory": "compact",
    "user_id": "59bb4331403cba000a116359",
    "tenant_id": "59bb4330403cba000a116355",
    "time_create": "2017-09-15T03:07:36Z",
    "time_update": "2017-09-15T04:24:31Z",
    "dockerRegistries": [
     {
      "_id": "59bb43f8284ae3000bf663e7",
      "name": "linkerRegistry",
      "registry": "192.168.10.95:5000",
      "secure": false,
      "ca_text": "",
      "username": "",
      "password": "",
      "user_id": "",
      "tenant_id": "",
      "isUse": false,
      "isSystemRegistry": true,
      "TimeCreate": "0001-01-01T00:00:00Z"
     }
    ],
    "setProjectvalue": {
     "cmi": false
    }
   }
  ]
 }
```

# 三 查询单个集群信息

REQUEST

```
Url :
Header: X_Auth_Token
GET /v1/cluster/{cluster_id}
```

EXAMPLE:

```
curl -X GET -H "Content-Type: application/json" -H "X-Auth-Token: 59c1d404403cba000a11635e" http://192.168.10.211:10002/v1/cluster/59bb43f8284ae3000bf663ea
```

RESPONSE

```
{
  "success": true,
  "data": {
   "_id": "59bb43f8284ae3000bf663ea",
   "name": "cluster",
   "owner": "sysadmin",
   "endPoint": "192.168.10.212",
   "instances": 3,
   "pubkeyId": [],
   "providerId": "",
   "details": "",
   "status": "RUNNING",
   "type": "customized",
   "createCategory": "compact",
   "user_id": "59bb4331403cba000a116359",
   "tenant_id": "59bb4330403cba000a116355",
   "time_create": "2017-09-15T03:07:36Z",
   "time_update": "2017-09-15T04:24:31Z",
   "dockerRegistries": [
    {
     "_id": "59bb43f8284ae3000bf663e7",
     "name": "linkerRegistry",
     "registry": "192.168.10.95:5000",
     "secure": false,
     "ca_text": "",
     "username": "",
     "password": "",
     "user_id": "",
     "tenant_id": "",
     "isUse": false,
     "isSystemRegistry": true,
     "TimeCreate": "0001-01-01T00:00:00Z"
    }
   ],
   "setProjectvalue": {
    "cmi": false
   }
  }
 }
```

# 四 删除集群

REQUEST

```
Url :
Header: X_Auth_Token
DELETE /v1/cluster/{cluster_id}
```

EXAMPLE:

```
curl -X DELETE -H "Content-Type: application/json" -H "X-Auth-Token: 59c1d404403cba000a11635e" http://192.168.10.211:10002/v1/cluster/59bb43f8284ae3000bf663ea
```

RESPONSE

```
{ "success": true, "data": [] }
```


# 五 添加节点

REQUEST

```
Url :
Header: X_Auth_Token
POST /v1/cluster/{cluster_id}/hosts 
body{}
```

EXAMPLE:

```
curl -X POST -H "Content-Type: application/json" -H "X-Auth-Token: 59c1d404403cba000a11635e" http://192.168.10.211:10002/v1/cluster/59bb43f8284ae3000bf663ea/hosts -d @add.json
```

其中add.json如下所示：

```
{
"clusterId":"59bb43f8284ae3000bf663ea",
"sharedCount":0,
"pureslaveCount":1,
"addMode":"reuse",
"pureslaveNodes":[
		{
            "ip": "192.168.10.214",
            "sshUser": "root",
            "privateKey": "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",
            "privateNicName": "eth0"
        }
    ],
"dockerRegistries":[
           {
              "_id": "59bb43f8284ae3000bf663e7",
              "registry":"192.168.10.95:5000",
              "isSystemRegistry":true
           }
        ]

}
```

RESPONSE

```
{
    "success": true,
    "data": {
        "_id": "59bb43f8284ae3000bf663ea",
        "name": "cluster",
        "owner": "sysadmin",
        "endPoint": "192.168.10.212",
        "instances": 2,
        "pubkeyId": [],
        "providerId": "",
        "details": "",
        "status": "RUNNING",
        "type": "customized",
        "createCategory": "compact",
        "user_id": "59bb4331403cba000a116359",
        "tenant_id": "59bb4330403cba000a116355",
        "time_create": "2017-09-15T03:07:36Z",
        "time_update": "2017-09-15T04:12:49Z",
        "dockerRegistries": [
            {
                "_id": "59bb43f8284ae3000bf663e7",
                "name": "linkerRegistry",
                "registry": "192.168.10.95:5000",
                "secure": false,
                "ca_text": "",
                "username": "",
                "password": "",
                "user_id": "",
                "tenant_id": "",
                "isUse": false,
                "isSystemRegistry": true,
                "TimeCreate": "0001-01-01T00:00:00Z"
            }
        ],
        "setProjectvalue": {
            "cmi": false
        }
    }
}
```

# 六 删除节点

REQUEST

```
Url :
Header: X_Auth_Token
DELETE /v1/cluster/{cluster_id}/hosts
body{}
```

EXAMPLE:

```
curl -X DELETE -H "Content-Type: application/json" -H "X-Auth-Token: 59c1d404403cba000a11635e" http://192.168.10.211:10002/v1/cluster/59bb43f8284ae3000bf663ea/hosts -d @delete.json
```

其中delete.json如下所示

```
{
"host_ids":["59bb5373284ae3000bf663ef"]
}
```

RESPONSE

```
{ "success": true, "data": [] }
```

# 七 查询一个集群中的所有节点

REQUEST

```
Url :
Header: X_Auth_Token
GET /v1/cluster/{cluster_id}/hosts
参数
/v1/cluster/{cluster_id}/hosts？count=true
&skip=
&limit=
&sort=
&status=
&tag=
```

EXAMPLE:

```
curl -X GET -H "Content-Type: application/json" -H "X-Auth-Token: 59c1d404403cba000a11635e" http://192.168.10.211:10002/v1/cluster/59bb43f8284ae3000bf663ea/hosts
```

RESPONSE


```
{
  "success": true,
  "data": [
   {
    "hostId": "59bb43f8284ae3000bf663eb",
    "hostName": "a-420286728-cluster-sysadmin",
    "clusterId": "59bb43f8284ae3000bf663ea",
    "clusterName": "cluster",
    "status": "RUNNING",
    "ip": "192.168.10.212",
    "privateIp": "192.168.10.212",
    "task": 0,
    "cpu": "",
    "memory": "",
    "gpu": "",
    "tag": null,
    "isMasterNode": true,
    "isSlaveNode": false,
    "isSharedNode": false,
    "isFullfilled": true,
    "isClientNode": true,
    "user_id": "59bb4331403cba000a116359",
    "tenant_id": "59bb4330403cba000a116355",
    "pubkeyName": "",
    "type": "customized",
    "time_create": "2017-09-15T03:07:36Z",
    "time_update": "2017-09-15T04:12:49Z"
   },
   {
    "hostId": "59bb43f8284ae3000bf663ec",
    "hostName": "f-952748725-cluster-sysadmin",
    "clusterId": "59bb43f8284ae3000bf663ea",
    "clusterName": "cluster",
    "status": "RUNNING",
    "ip": "192.168.10.213",
    "privateIp": "192.168.10.213",
    "task": 3,
    "cpu": "0.3/4",
    "memory": "712/2927",
    "gpu": "0/0",
    "tag": [
     "lb=enable",
     "public_ip=true"
    ],
    "isMasterNode": false,
    "isSlaveNode": true,
    "isSharedNode": true,
    "isFullfilled": true,
    "isClientNode": false,
    "user_id": "59bb4331403cba000a116359",
    "tenant_id": "59bb4330403cba000a116355",
    "pubkeyName": "",
    "type": "customized",
    "time_create": "2017-09-15T03:07:36Z",
    "time_update": "2017-09-15T04:12:49Z"
   },
   {
    "hostId": "59bb5373284ae3000bf663ef",
    "hostName": "u-135224906-cluster-sysadmin",
    "clusterId": "59bb43f8284ae3000bf663ea",
    "clusterName": "cluster",
    "status": "RUNNING",
    "ip": "192.168.10.214",
    "privateIp": "192.168.10.214",
    "task": 2,
    "cpu": "0.2/4",
    "memory": "200/2927",
    "gpu": "0/0",
    "tag": [],
    "isMasterNode": false,
    "isSlaveNode": true,
    "isSharedNode": false,
    "isFullfilled": true,
    "isClientNode": false,
    "user_id": "59bb4331403cba000a116359",
    "tenant_id": "59bb4330403cba000a116355",
    "pubkeyName": "",
    "type": "customized",
    "time_create": "2017-09-15T04:13:39Z",
    "time_update": "2017-09-15T04:24:31Z"
   }
  ]
 }
```

# 八 查询集群中某个节点信息

REQUEST

```
Url :
Header: X_Auth_Token
GET /v1/cluster/{cluster_id}/hosts/{host_id}
```

EXAMPLE:

```
curl -X GET -H "Content-Type: application/json" -H "X-Auth-Token: 59c1d404403cba000a11635e" http://192.168.10.211:10002/v1/cluster/59bb43f8284ae3000bf663ea/hosts/59bb5373284ae3000bf663ef
```

RESPONSE

```
{
  "success": true,
  "data": {
   "_id": "59bb5373284ae3000bf663ef",
   "hostName": "u-135224906-cluster-sysadmin",
   "clusterId": "59bb43f8284ae3000bf663ea",
   "clusterName": "cluster",
   "status": "RUNNING",
   "ip": "192.168.10.214",
   "privateIp": "192.168.10.214",
   "isMasterNode": false,
   "isSlaveNode": true,
   "isMonitorServer": false,
   "isSharedNode": false,
   "isFullfilled": true,
   "isClientNode": false,
   "user_id": "59bb4331403cba000a116359",
   "tenant_id": "59bb4330403cba000a116355",
   "type": "customized",
   "sshUser": "root",
   "time_create": "2017-09-15T04:13:39Z",
   "time_update": "2017-09-15T04:24:31Z"
  }
 }
```

# 九 给集群添加pubkey

REQUEST

```
Url :
Header: X_Auth_Token
POST /v1/cluster/{cluster_id}/pubkey
body{}
```

EXAMPLE:

```
curl -X POST -H "Content-Type: application/json" -H "X-Auth-Token: 59c1d404403cba000a11635e" http://192.168.10.211:10002/v1/cluster/59bb43f8284ae3000bf663ea/pubkey -d @pubkey.json
```

其中pubkey.json如下所示：

```
{
"ids":["59c21faf284ae3000bf663f4"]
}
```

RESPONSE

```
{
  "success": true
}
```

# 十 删除集群中pubkey

REQUEST

```
Url :
Header: X_Auth_Token
DELETE /v1/cluster/{cluster_id}/pubkey
body{}
```

EXAMPLE:

```
curl -X DELETE -H "Content-Type: application/json" -H "X-Auth-Token: 59c1d404403cba000a11635e" http://192.168.10.211:10002/v1/cluster/59bb43f8284ae3000bf663ea/pubkey -d @pubkey.json
```

其中pubkey.json如下所示：

```
{
"ids":["59c21faf284ae3000bf663f4"]
}
```

RESPONSE

```
{
  "success": true
}
```

# 十一 给集群添加dockerregistry


REQUEST

```
Url :
Header: X_Auth_Token
POST /v1/cluster/{cluster_id}/registry
body{}
```

EXAMPLE:

```
curl -X POST -H "Content-Type: application/json" -H "X-Auth-Token: 59c22883403cba000a116362" http://192.168.10.211:10002/v1/cluster/59bb43f8284ae3000bf663ea/registry -d @registry.josn
```

其中registry.json如下所示：

```
{
"ids":["59bb43f8284ae3000bf663e7"]
}
```

RESPONSE

```
{
  "success": true
}
```

# 十二 删除集群中dockerregistry

REQUEST

```
Url :
Header: X_Auth_Token
DELETE /v1/cluster/{cluster_id}/registry
body{}
```

EXAMPLE:

```
curl -X DELETE -H "Content-Type: application/json" -H "X-Auth-Token: 59c22883403cba000a116362" http://192.168.10.211:10002/v1/cluster/59bb43f8284ae3000bf663ea/registry -d @registry.josn
```

其中registry.json如下所示：

```
{
"ids":["59bb43f8284ae3000bf663e7"]
}
```

RESPONSE

```
{
  "success": true
}
```

# 十三 检查集群名称是否有效

REQUEST

```
Url :
Header: X_Auth_Token
GET /v1/cluster/validate
参数
/v1/cluster/validate？userid=
&clustername=
```

EXAMPLE:

```
curl -X GET -H "Content-Type: application/json" -H "X-Auth-Token: 59c22883403cba000a116362" http://192.168.10.211:10002/v1/cluster/validate?userid=59bb4331403cba000a116359&clustername=test
```

RESPONSE

```
{
  "success": true
}
```

# 十四 创建镜像仓库

REQUEST

```
Url :
Header: X_Auth_Token
POST /v1/dockerregistries
body{}
```

EXAMPLE:

```
curl -X POST -H "Content-Type: application/json" -H "X-Auth-Token: 59c22883403cba000a116362" http://192.168.10.211:10002/v1/dockerregistries -d @registry.json
```

其中registry.json如下所示：

```
{
    "name": "registry_name",
    "registry": "192.168.10.95:5000",
    "user_id": "59bb4331403cba000a116359",
    "isSystemRegistry": true
}
```

RESPONSE

```
{
    "success": true,
    "data": {
        "_id": "59bb43f8284ae3000bf663e7",
        "name": "linkerRegistry",
        "registry": "192.168.10.95:5000",
        "secure": false,
        "ca_text": "",
        "username": "",
        "password": "",
        "user_id": "59bb4331403cba000a116359",
        "tenant_id": "59bb4330403cba000a116355",
        "isUse": false,
        "isSystemRegistry": true,
        "TimeCreate": "2017-09-15T03:07:36Z"
    }
}
```

# 十五 查询dockerregistry


REQUEST

```
Url :
Header: X_Auth_Token
GET /v1/dockerregistries
参数
/v1/dockerregistries？count=true
&skip=
&limit=
&sort=
&user_id=
```

EXAMPLE:

```
curl -X GET -H "Content-Type: application/json" -H "X-Auth-Token: 59c324f9403cba000a116363" http://192.168.10.211:10002/v1/dockerregistries?count=true&user_id=59bb4331403cba000a116359
```

RESPONSE

```
{
  "success": true,
  "count": 1,
  "data": [
   {
    "_id": "59bb43f8284ae3000bf663e7",
    "name": "linkerRegistry",
    "registry": "192.168.10.95:5000",
    "secure": false,
    "ca_text": "",
    "username": "",
    "password": "",
    "user_id": "59bb4331403cba000a116359",
    "tenant_id": "59bb4330403cba000a116355",
    "isUse": true,
    "isSystemRegistry": true,
    "TimeCreate": "2017-09-15T03:07:36Z"
   }
  ]
 }
```

# 十六 查询单个dockerregistry

REQUEST

```
Url :
Header: X_Auth_Token
GET /v1/dockerregistries/{registry_id}
```

EXAMPLE:

```
curl -X GET -H "Content-Type: application/json" -H "X-Auth-Token: 59c324f9403cba000a116363" http://192.168.10.211:10002/v1/dockerregistries/59bb43f8284ae3000bf663e7
```

RESPONSE

```
{
  "success": true,
  "data": {
   "_id": "59bb43f8284ae3000bf663e7",
   "name": "linkerRegistry",
   "registry": "192.168.10.95:5000",
   "secure": false,
   "ca_text": "",
   "username": "",
   "password": "",
   "user_id": "59bb4331403cba000a116359",
   "tenant_id": "59bb4330403cba000a116355",
   "isUse": false,
   "isSystemRegistry": true,
   "TimeCreate": "2017-09-15T03:07:36Z"
  }
 }
```

# 十七 删除dockerregistry

REQUEST

```
Url :
Header: X_Auth_Token
DELETE /v1/dockerregistries/{registry_id}
```

EXAMPLE:

```
curl -X DELETE -H "Content-Type: application/json" -H "X-Auth-Token: 59c324f9403cba000a116363" http://192.168.10.211:10002/v1/dockerregistries/59bb43f8284ae3000bf663e7
```

RESPONSE

```
{
  "success": true
}
```

# 十八 查询registry名称是否合法以及registry是否有集群正在使用

REQUEST

```
Url :
Header: X_Auth_Token
GET /v1/dockerregistries/registryValidate
参数
/v1/dockerregistries/registryValidate？
type=(name or used)
&name=
&user_id=
```

EXAMPLE:

```
curl -X GET -H "Content-Type: application/json" -H "X-Auth-Token: 59c324f9403cba000a116363" http://192.168.10.211:10002/v1/dockerregistries/registryValidate?type=name&name=jgjg&user_id=59bb4331403cba000a116359
```

RESPONSE

```
{
  "success": true,
  "data": null
 }
```

# 十九 添加平台

REQUEST

```
Url :
Header: X_Auth_Token
POST /v1/provider
body{}
```

EXAMPLE:

```
curl -X POST -H "Content-Type: application/json" -H "X-Auth-Token: 59c324f9403cba000a116363" http://192.168.10.211:10002/v1/provider -d @provider.json
```

其中provider.json如下所示：

```
{
"name":"provider",
"user_id":"59bb4331403cba000a116359",
"type":"google",
"sshUser":"root",
"googleInfo":{
"google-project":"project",
"google-zone":"zone",
"google-machine-type":"type",
"google-machine-image":"image",
"google-network":"network",
"google-username":"username",
"google-disk-size":"size",
"google-disk-type":"type",
"google-use-internal-ip":"ip",
"google-tags":"tags",
"google-applocation-credentials":"credentials"
}
}
```

RESPONSE

```
{
  "success": true,
  "data": {
   "_id": "59c3556b284ae3000bf663f9",
   "name": "provider",
   "type": "google",
   "sshUser": "root",
   "openstackInfo": {
    "openstack-auth-url": "",
    "openstack-username": "",
    "openstack-password": "",
    "openstack-tenant-name": "",
    "openstack-flavor-name": "",
    "openstack-image-name": "",
    "openstack-ssh-user": "",
    "openstack-sec-groups": "",
    "openstack-floatingip-pool": "",
    "openstack-nova-network": ""
   },
   "awsEc2Info": {
    "amazonec2-access-key": "",
    "amazonec2-secret-key": "",
    "amazonec2-ami": "",
    "amazonec2-instance-type": "",
    "amazonec2-root-size": "",
    "amazonec2-region": "",
    "amazonec2-vpc-id": "",
    "amazonec2-ssh-user": ""
   },
   "googleInfo": {
    "google-project": "project",
    "google-zone": "zone",
    "google-machine-type": "type",
    "google-machine-image": "image",
    "google-network": "network",
    "google-username": "username",
    "google-disk-size": "size",
    "google-disk-type": "type",
    "google-use-internal-ip": "ip",
    "google-tags": "tags",
    "google-application-credentials": ""
   },
   "user_id": "59bb4331403cba000a116359",
   "tenant_id": "59bb4330403cba000a116355",
   "time_create": "2017-09-21T06:00:11Z",
   "time_update": "2017-09-21T06:00:11Z"
  }
 }
```

# 二十 查询平台

REQUEST

```
Url :
Header: X_Auth_Token
GET /v1/provider
参数
/v1/provider？count=true
&type=
&skip=
&limit=
&sort=
&user_id=
```

EXAMPLE:

```
curl -X GET -H "Content-Type: application/json" -H "X-Auth-Token: 59c324f9403cba000a116363" http://192.168.10.211:10002/v1/provider?count=true&user_id=59bb4331403cba000a116359&type=google
```

RESPONSE

```
{
  "success": true,
  "count": 2,
  "data": [
   {
    "_id": "59c353a2284ae3000bf663f6",
    "name": "t",
    "type": "google",
    "sshUser": "t",
    "openstackInfo": {
     "openstack-auth-url": "",
     "openstack-username": "",
     "openstack-password": "",
     "openstack-tenant-name": "",
     "openstack-flavor-name": "",
     "openstack-image-name": "",
     "openstack-ssh-user": "",
     "openstack-sec-groups": "",
     "openstack-floatingip-pool": "",
     "openstack-nova-network": ""
    },
    "awsEc2Info": {
     "amazonec2-access-key": "",
     "amazonec2-secret-key": "",
     "amazonec2-ami": "",
     "amazonec2-instance-type": "",
     "amazonec2-root-size": "",
     "amazonec2-region": "",
     "amazonec2-vpc-id": "",
     "amazonec2-ssh-user": ""
    },
    "googleInfo": {
     "google-project": "t",
     "google-zone": "t",
     "google-machine-type": "t",
     "google-machine-image": "t",
     "google-network": "t",
     "google-username": "t",
     "google-disk-size": "t",
     "google-disk-type": "t",
     "google-use-internal-ip": "true",
     "google-application-credentials": "yA=="
    },
    "user_id": "59bb4331403cba000a116359",
    "tenant_id": "59bb4330403cba000a116355",
    "isuse": false,
    "time_create": "2017-09-21T05:52:34Z",
    "time_update": "2017-09-21T05:52:34Z"
   },
   {
    "_id": "59c3556b284ae3000bf663f9",
    "name": "provider",
    "type": "google",
    "sshUser": "root",
    "openstackInfo": {
     "openstack-auth-url": "",
     "openstack-username": "",
     "openstack-password": "",
     "openstack-tenant-name": "",
     "openstack-flavor-name": "",
     "openstack-image-name": "",
     "openstack-ssh-user": "",
     "openstack-sec-groups": "",
     "openstack-floatingip-pool": "",
     "openstack-nova-network": ""
    },
    "awsEc2Info": {
     "amazonec2-access-key": "",
     "amazonec2-secret-key": "",
     "amazonec2-ami": "",
     "amazonec2-instance-type": "",
     "amazonec2-root-size": "",
     "amazonec2-region": "",
     "amazonec2-vpc-id": "",
     "amazonec2-ssh-user": ""
    },
    "googleInfo": {
     "google-project": "project",
     "google-zone": "zone",
     "google-machine-type": "type",
     "google-machine-image": "image",
     "google-network": "network",
     "google-username": "username",
     "google-disk-size": "size",
     "google-disk-type": "type",
     "google-use-internal-ip": "ip",
     "google-tags": "tags",
     "google-application-credentials": ""
    },
    "user_id": "59bb4331403cba000a116359",
    "tenant_id": "59bb4330403cba000a116355",
    "isuse": false,
    "time_create": "2017-09-21T06:00:11Z",
    "time_update": "2017-09-21T06:00:11Z"
   }
  ]
 }
```

# 二十一 查询单个平台信息

REQUEST

```
Url :
Header: X_Auth_Token
GET /v1/provider/{provider_id}
```

EXAMPLE：

```
curl -X GET -H "Content-Type: application/json" -H "X-Auth-Token: 59c324f9403cba000a116363" http://192.168.10.211:10002/v1/provider/59c3556b284ae3000bf663f9
```

RESPONSE:

```
{
  "success": true,
  "data": {
   "_id": "59c3556b284ae3000bf663f9",
   "name": "provider",
   "type": "google",
   "sshUser": "root",
   "openstackInfo": {
    "openstack-auth-url": "",
    "openstack-username": "",
    "openstack-password": "",
    "openstack-tenant-name": "",
    "openstack-flavor-name": "",
    "openstack-image-name": "",
    "openstack-ssh-user": "",
    "openstack-sec-groups": "",
    "openstack-floatingip-pool": "",
    "openstack-nova-network": ""
   },
   "awsEc2Info": {
    "amazonec2-access-key": "",
    "amazonec2-secret-key": "",
    "amazonec2-ami": "",
    "amazonec2-instance-type": "",
    "amazonec2-root-size": "",
    "amazonec2-region": "",
    "amazonec2-vpc-id": "",
    "amazonec2-ssh-user": ""
   },
   "googleInfo": {
    "google-project": "project",
    "google-zone": "zone",
    "google-machine-type": "type",
    "google-machine-image": "image",
    "google-network": "network",
    "google-username": "username",
    "google-disk-size": "size",
    "google-disk-type": "type",
    "google-use-internal-ip": "ip",
    "google-tags": "tags",
    "google-application-credentials": ""
   },
   "user_id": "59bb4331403cba000a116359",
   "tenant_id": "59bb4330403cba000a116355",
   "time_create": "2017-09-21T06:00:11Z",
   "time_update": "2017-09-21T06:00:11Z"
  }
 }
```

# 二十二 更新平台信息

REQUEST

```
Url :
Header: X_Auth_Token
PUT /v1/provider/{provider_id}
body{}
```

EXAMPLE：

```
curl -X PUT -H "Content-Type: application/json" -H "X-Auth-Token: 59c324f9403cba000a116363" http://192.168.10.211:10002/v1/provider/59c3556b284ae3000bf663f9 -d @provider.json
```

其中provider.json如下所示：

```
{
"name":"provider",
"_id":"59c3556b284ae3000bf663f9",
"user_id":"59bb4331403cba000a116359",
"type":"google",
"sshUser":"root",
"googleInfo":{
"google-project":"project-change",
"google-zone":"zone-change",
"google-machine-type":"type",
"google-machine-image":"image",
"google-network":"network",
"google-username":"username",
"google-disk-size":"size",
"google-disk-type":"type",
"google-use-internal-ip":"ip",
"google-tags":"tags",
"google-applocation-credentials":"credentials"
}
}
```

RESPONSE:

```
{
  "success": true,
  "data": {
   "created": false,
   "url": "/v1/provider/59c3556b284ae3000bf663f9"
  }
 }
```

# 二十三 删除平台信息

REQUEST

```
Url :
Header: X_Auth_Token
DELETE /v1/provider/{provider_id}
```

EXAMPLE：

```
curl -X DELETE -H "Content-Type: application/json" -H "X-Auth-Token: 59c324f9403cba000a116363" http://192.168.10.211:10002/v1/provider/59c3556b284ae3000bf663f9
```

RESPONSE:

```
{
  "success": true
 }
```

# 二十四 验证平台名称是否合法

REQUEST

```
Url :
Header: X_Auth_Token
GET /v1/provider/validate
参数
/v1/provider/validate？provider_name=
```

EXAMPLE：

```
curl -X GET -H "Content-Type: application/json" -H "X-Auth-Token: 59c324f9403cba000a116363" http://192.168.10.211:10002/v1/provider/validate?provider_name=test
```

RESPONSE:

```
{
  "success": true,
  "data": null
 }
```

# 二十五 上传秘钥

REQUEST

```
Url :
Header: X_Auth_Token
POST /v1/pubkey
body{}
```

EXAMPLE：

```
curl -X POST -H "Content-Type: application/json" -H "X-Auth-Token: 59c324f9403cba000a116363" http://192.168.10.211:10002/v1/pubkey -d @pubkey.json
```

其中pubkey.json如下所示：

```
{
"pubkeyValue":"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDTqvb0YAXZpQA4ci5RgbjKUQxwwJFIOKRzOQvZdfRhQx3aZ762I5vOahAk4ILMflxDPybNyB3pJI7uwuQ2bT8A2vTKbW57luoG2iFJngTN7XJZdbaD8cprNJiGqJiczPpQl5kty3E55U6gC4/X7CsI40QNnh/Kb0VFNYj57eryCxN/q3Bl42PSkmakryxd+fsRXbSqK35COnTLJrndKVwIYbYySD1OKeYCWgiBH87WSZLkVrlEfRauX/KLSZdY9aX/earfxym8XPbdp+8ZUJvlEBnLS/rpOkDHCq8bUWLRRoaqaxtxPMrWAUpXvLTZ9euMG3xQaVvpGUII7wkl9gb5 root@centosdesktop",
"name":"pubkey",
"user_id":"59bb4331403cba000a116359"
}
```

RESPONSE:

```
{
  "success": true,
  "data": {
   "_id": "59c35fc5284ae3000bf663fc",
   "pubkeyValue": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDTqvb0YAXZpQA4ci5RgbjKUQxwwJFIOKRzOQvZdfRhQx3aZ762I5vOahAk4ILMflxDPybNyB3pJI7uwuQ2bT8A2vTKbW57luoG2iFJngTN7XJZdbaD8cprNJiGqJiczPpQl5kty3E55U6gC4/X7CsI40QNnh/Kb0VFNYj57eryCxN/q3Bl42PSkmakryxd+fsRXbSqK35COnTLJrndKVwIYbYySD1OKeYCWgiBH87WSZLkVrlEfRauX/KLSZdY9aX/earfxym8XPbdp+8ZUJvlEBnLS/rpOkDHCq8bUWLRRoaqaxtxPMrWAUpXvLTZ9euMG3xQaVvpGUII7wkl9gb5 root@centosdesktop",
   "name": "pubkey",
   "owner": "sysadmin",
   "user_id": "59bb4331403cba000a116359",
   "isuse": false,
   "tenant_id": "59bb4330403cba000a116355",
   "time_create": "2017-09-21T06:44:21Z",
   "time_update": "2017-09-21T06:44:21Z"
  }
 }
```

# 二十六 查询秘钥

REQUEST

```
Url :
Header: X_Auth_Token
GET /v1/pubkey
参数
/v1/pubkey？count=true
&name=
&skip=
&limit=
&sort=
&user_id=
```

EXAMPLE：

```
curl -X GET -H "Content-Type: application/json" -H "X-Auth-Token: 59c324f9403cba000a116363" http://192.168.10.211:10002/v1/pubkey?user_id=59bb4331403cba000a116359&count=true
```

RESPONSE:

```
{
  "success": true,
  "data": [
   {
    "_id": "59c21faf284ae3000bf663f4",
    "pubkeyValue": "hhghghghghghg",
    "name": "key",
    "owner": "sysadmin",
    "user_id": "59bb4331403cba000a116359",
    "isuse": false,
    "tenant_id": "59bb4330403cba000a116355",
    "time_create": "2017-09-20T07:58:39Z",
    "time_update": "2017-09-20T07:58:39Z"
   },
   {
    "_id": "59c35fc5284ae3000bf663fc",
    "pubkeyValue": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDTqvb0YAXZpQA4ci5RgbjKUQxwwJFIOKRzOQvZdfRhQx3aZ762I5vOahAk4ILMflxDPybNyB3pJI7uwuQ2bT8A2vTKbW57luoG2iFJngTN7XJZdbaD8cprNJiGqJiczPpQl5kty3E55U6gC4/X7CsI40QNnh/Kb0VFNYj57eryCxN/q3Bl42PSkmakryxd+fsRXbSqK35COnTLJrndKVwIYbYySD1OKeYCWgiBH87WSZLkVrlEfRauX/KLSZdY9aX/earfxym8XPbdp+8ZUJvlEBnLS/rpOkDHCq8bUWLRRoaqaxtxPMrWAUpXvLTZ9euMG3xQaVvpGUII7wkl9gb5 root@centosdesktop",
    "name": "pubkey",
    "owner": "sysadmin",
    "user_id": "59bb4331403cba000a116359",
    "isuse": false,
    "tenant_id": "59bb4330403cba000a116355",
    "time_create": "2017-09-21T06:44:21Z",
    "time_update": "2017-09-21T06:44:21Z"
   }
  ]
 }
```

# 二十七 查询单个秘钥信息

REQUEST

```
Url :
Header: X_Auth_Token
GET /v1/pubkey/{pubkey_id}
```

EXAMPLE：

```
curl -X GET -H "Content-Type: application/json" -H "X-Auth-Token: 59c324f9403cba000a116363" http://192.168.10.211:10002/v1/pubkey/59c21faf284ae3000bf663f4
```

RESPONSE:

```
{
  "success": true,
  "data": {
   "_id": "59c21faf284ae3000bf663f4",
   "pubkeyValue": "hhghghghghghg",
   "name": "key",
   "owner": "sysadmin",
   "user_id": "59bb4331403cba000a116359",
   "isuse": false,
   "tenant_id": "59bb4330403cba000a116355",
   "time_create": "2017-09-20T07:58:39Z",
   "time_update": "2017-09-20T07:58:39Z"
  }
 }
```

# 二十八 创建秘钥

REQUEST

```
Url :
Header: X_Auth_Token
POST /v1/pubkey/userSelfCreate
body{}
```

EXAMPLE：

```
curl -X POST -H "Content-Type: application/json" -H "X-Auth-Token: 59c324f9403cba000a116363" http://192.168.10.211:10002/v1/pubkey/userSelfCreate -d @pubkey2.json
```

其中pubkey2.json如下所示：

```
{
"name":"pubkey2",
"user_id":"59bb4331403cba000a116359"
}
```

RESPONSE:

```
{
  "success": true,
  "data": "59bb4331403cba000a116359"
 }
```

# 二十九 下载创建的秘钥对应的私钥

REQUEST

```
Url :
Header: X_Auth_Token
GET /v1/pubkey/downLoadKey/{user_id}
```

EXAMPLE：

```
curl -X GET -H "Content-Type: application/json" -H "X-Auth-Token: 59c324f9403cba000a116363" http://192.168.10.211:10002/v1/pubkey/downLoadKey/59bb4331403cba000a116359
```

RESPONSE:

```
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAo+lO2hhDo6wkRZSk32qmYTN8Hdao1BFreJE2bZNcbwkb0daY
GvMTZ0zQy1Bw+0Hunpz2FfzgnguNwG0trl+BS4p9k2JYW2UiWAVTcYdoYIg5es4F
ljcK7hl5mtlfWF1YvWNmRxpGnfORTSMtPJzjx2TzIhNtMjfDDPO8BKaZ6t5+73nT
L6dZJWazv/dq9k+m5RFRrS1EC2Z2itjCuZKCP4Kp3EIZBOAax4hU/f6afk3MxoBB
DO172pqzwC3gyMVTlr+zRXT2/P1InE1q2p/FBZ1XHHbCXbrdv/2Ilh+9994ocxn1
Fxv3Jefxs8A7Hv0Uq+N8GXfuVmcI3hj0vHZh3wIDAQABAoIBAG0mnmXJprFFntnf
BHPq12T/HfXYzpB6ETE2siIB8ZnXXPk9iAjaOd+eXaQmqzYGT9q793vo68MTOpAb
pEHsQ3OEg98zrFcgX+Bxm4GMhEtUK8LFkx7XBKZNvJcLjdyQPNnRaXiL3N6uJeJS
PHuSlnRfmzDj8uFwFKl5XYlTUEgAVTO4EeZZwdDsXn1KGkEURU5Kth4xpm2qBLjJ
sTMxndE5F6VchOugcYmcjTuLsOQ0TLiZD6SooNaCivhjJuOXOFR2TisTlndImq/x
Ao5w2HrJ1p3SWttsod2vZ1sdeTyfXlPw8JVzNHKaosumzJBFnNCLWS3WDMF7Cs0/
9Gh2WdECgYEA1rIz+xm33Vrh6RC6o/BrGR3CMxYwib09GOipTKnWbwLn29KwUnsl
eyGIGGOtSME8yKv6QcSrCy8nUc1FKoBk/mBjBLRzsMyvK4VkYFGDZbfsjy1Jb9s+
pkpDsAJYfS79gRb/Z2q7myM26lbEZkIXcDM8EtmkkPAIpY70mgXaybMCgYEAw3Hz
9U6jKsdln618Y9jW61AjHG0DGzDRU/M28jbO/h1ShR8YqpSJqPCRXbidktPItRil
yfx7bKyao+xTptzdgwbnMh5x+QL9liueZoEpAIY2a+9z8aEwvQBUdxs0oHIC+big
CPLxzKMYJKroqRLNwAdG1iAYeTgBeGO3FhFoWSUCgYB5g9Eh7POBCKBWjo5knX2w
cIRq78M3InGDOKQh7PqeSFG8vGnptSOIpnjl/Pyl8iEaHyR8tvhsUxr5FKpyHMuM
ojdJAW19gsweYNhoH5q0Jr5wZxxqf/fcnKnk4977s23t83tJKELY0ryRM9zjV8L2
UTlOHfsjwYfTVK8iwe+MOwKBgQDAhcanlN/b6vErGqzWeiozQAxmGugdZ7g7tvAg
Jmc+IEpCQcB9f7YeyWKYbJwjnyUtZushDenSwi/OW6SHUTeOs0UYtK7WeOCthagS
Fxb5ojuHlSekFIE7HFEXxp/PkJ9nuDtEtXQEfX/x1r06lwBAMarQkGsrNUUVfzxB
q8IbVQKBgQCs9Nt5oH+TdJix7By7u41U/K8nk8zVkiLcqKKzTeESP/kmOTUCuNaQ
oht+BAsrCSO5Eq3RUddDnBF/IEB1UCH83pINDrNKfpg6q40DhJYwIoiS7PBNtmAS
Dd2DfJMyrNLfxty3y68uYgxiYRAlXwOk4luvHtk0Btne2DE6ERiX5Q==
-----END RSA PRIVATE KEY-----
```


# 三十 添加服务器

REQUEST

```
Url :
Header: X_Auth_Token
POST /v1/smtp
body{}
```

EXAMPLE：

```
curl -X POST -H "Content-Type: application/json" -H "X-Auth-Token: 59c324f9403cba000a116363" http://192.168.10.211:10002/v1/smtp -d @smtp.json
```

其中smtp.json如下所示

```
{
"name":"username",
"address":"smtp.qiye.163.com",
"passwd":"password"
}
```

RESPONSE:


```
{
  "success": true,
  "data": {
   "_id": "59c377d3284ae3000bf66400",
   "name": "username",
   "address": "smtp.qiye.163.com",
   "passwd": "xGFcxgyYx7Q="
  }
 }
```

# 三十一 查询smtp

REQUEST

```
Url :
Header: X_Auth_Token
GET /v1/smtp
参数
/v1/pubkey？count=true
&skip=
&limit=
&sort=
&address=
```

EXAMPLE：

```
curl -X GET -H "Content-Type: application/json" -H "X-Auth-Token: 59c324f9403cba000a116363" http://192.168.10.211:10002/v1/smtp?count=true
```

RESPONSE:

```
{
  "success": true,
  "count": 1,
  "data": [
   {
    "_id": "59c377d3284ae3000bf66400",
    "name": "username",
    "address": "smtp.qiye.163.com",
    "passwd": "xGFcxgyYx7Q="
   }
  ]
 }
```

# 三十二 更新smtp

REQUEST

```
Url :
Header: X_Auth_Token
PUT /v1/smtp/{smtp_id}
body{}
```

EXAMPLE：

```
curl -X PUT -H "Content-Type: application/json" -H "X-Auth-Token: 59c37a46403cba000a116364" http://192.168.10.211:10002/v1/smtp/59c377d3284ae3000bf66400 -d @smtp.json
```

其中smtp.json如下所示

```
{
"name":"username",
"address":"smtp.qiye.163.com",
"passwd":"password-change"
}
```

RESPONSE:

```
{
  "success": true,
  "data": {
   "created": false,
   "url": "/v1/smtp/59c377d3284ae3000bf66400"
  }
 }
```


# 三十三 删除smtp

REQUEST

```
Url :
Header: X_Auth_Token
DELETE /v1/smtp/{smtp_id}
```

EXAMPLE：

```
curl -X DELETE -H "Content-Type: application/json" -H "X-Auth-Token: 59c37a46403cba000a116364" http://192.168.10.211:10002/v1/smtp/59c377d3284ae3000bf66400
```

RESPONSE:

```
{
  "success": true,
  "data": null
 }
```





# 3.0新增功能：获取系统组件api

REQUEST

```
Url :
Header: X_Auth_Token
GET /v1/cluster/{cluster_id}/components
```

返回数据结构：

```
type ComponentsInfo struct {  
	UserName  string  json:"userName" bson:"userName"  
	ClusterName string        json:"clusterName" bson:"clusterName"  
	ClusterId   string        json:"clusterId" bson:"clusterId"  
	ComponentsStatus []ComponentsStatus json:"componentStatus" bson:"componentStatus"
}

type ComponentsStatus struct {  
	ComponentName string json:"componentName" bson:"componentName"
	Ip string json:"ip" bson:"ip"
	Status string json:"status" bson:"status"
}
```

EXAMPLE：

```
curl -X GET -H "X-Auth-Token: 596c246f6e9e18000b8f1168"  http://192.168.3.51:10002/v1/cluster/596c49767df837000baf9c49/components
```

RESPONSE:
```
{
  "success": true,  
  "data": {  
   "userName": "sysadmin",  
   "clusterName": "b",  
   "clusterId": "596c49767df837000baf9c49",  
   "componentStatus": [  
    {  
     "imageName": "mongodb",  
     "ip": "192.168.3.52",  
     "status": "Healthy"  
    },  
    {
     "imageName": "adminrouter",  
     "ip": "192.168.3.52",  
     "status": "Healthy"  
    },  
    {  
     "imageName": "universeregistry",  
     "ip": "192.168.3.52",  
     "status": "Healthy"  
    },  
    {  
     "imageName": "universenginx",  
     "ip": "192.168.3.52",  
     "status": "Healthy"  
    },  
    {  
     "imageName": "dnsserver",  
     "ip": "192.168.3.52",  
     "status": "Healthy"  
    },  
    {  
     "imageName": "cosmos",  
     "ip": "192.168.3.52",  
     "status": "Healthy"  
    },  
    {  
     "imageName": "marathon",  
     "ip": "192.168.3.52",  
     "status": "Healthy"  
    },  
    {  
     "imageName": "mesosmaster",  
     "ip": "192.168.3.52",  
     "status": "Healthy"  
    },  
    {  
     "imageName": "exhibitor",  
     "ip": "192.168.3.52",  
     "status": "Healthy"  
    },  
    {  
     "imageName": "cadvisormonitor",  
     "ip": "192.168.3.56",  
     "status": "Healthy"  
    },  
    {  
     "imageName": "cadvisormonitor",  
     "ip": "192.168.3.53",  
     "status": "Healthy"  
    },  
    {  
     "imageName": "mesosslave",  
     "ip": "192.168.3.53",  
     "status": "UnHealthy"  
    },  
    {  
     "imageName": "mesosslave",  
     "ip": "192.168.3.56",  
     "status": "UnHealthy"  
    },  
    {  
     "imageName": "webconsole",  
     "ip": "192.168.3.52",  
     "status": "Healthy"  
    },  
    {  
     "imageName": "dcosclient",  
     "ip": "192.168.3.52",  
     "status": "Healthy"  
    },  
    {  
     "imageName": "alertmanager",  
     "ip": "192.168.3.52",  
     "status": "Healthy"  
    },  
    {  
     "imageName": "prometheus",  
     "ip": "192.168.3.52",  
     "status": "Healthy"  
    },  
    {  
     "imageName": "genresolvconf",  
     "ip": "192.168.3.56",  
     "status": "Healthy"  
    },  
    {  
     "imageName": "genresolvconf",  
     "ip": "192.168.3.52",  
     "status": "Healthy"  
    },  
    {  
     "imageName": "genresolvconf",  
     "ip": "192.168.3.53",  
     "status": "Healthy"  
    }  
   ]  
  }  
 }  
```

# 3.0修改：查询log api

REQUEST

```
url:
Header: X_Auth_Token
GET /v1/logs
参数：
/v1/appsets?count=true
&skip=10
&limit=5
&sort=time_update
&user_name=
&queryType=
&user_id=
&cluster_id=
```

EXAMPLE:

```
curl -X GET  -H "X-Auth-Token:598d0bbf88a87300081c8a26" "http://192.168.10.117:10002/v1/logs?user_id=5982dba001bf6000084d02ce&queryType=service"
```

RESPONSE

```
{
  "success": true,
  "data": [
   {
    "_id": "598bd536f5528800083d58b4",
    "clusterName": "2",
    "clusterId": "598bd034f5528800083d58a2",
    "operateType": "create_service",
    "queryType": "service",
    "userName": "sysadmin",
    "user_id": "5982dba001bf6000084d02ce",
    "tenant_id": "5982dba001bf6000084d02ca",
    "status": "success",
    "comments": "1",
    "time_create": "2017-08-10T03:38:30Z",
    "time_update": "2017-08-10T03:38:30Z"
   },
   {
    "_id": "598bd953f5528800083d58b5",
    "clusterName": "2",
    "clusterId": "598bd034f5528800083d58a2",
    "operateType": "create_component",
    "queryType": "service",
    "userName": "sysadmin",
    "user_id": "5982dba001bf6000084d02ce",
    "tenant_id": "5982dba001bf6000084d02ca",
    "status": "success",
    "comments": "/1/1",
    "time_create": "2017-08-10T03:56:03Z",
    "time_update": "2017-08-10T03:56:03Z"
   },
   {
    "_id": "598bd969f5528800083d58b6",
    "clusterName": "2",
    "clusterId": "598bd034f5528800083d58a2",
    "operateType": "start_component",
    "queryType": "service",
    "userName": "sysadmin",
    "user_id": "5982dba001bf6000084d02ce",
    "tenant_id": "5982dba001bf6000084d02ca",
    "status": "success",
    "comments": "/1/1",
    "time_create": "2017-08-10T03:56:25Z",
    "time_update": "2017-08-10T03:56:25Z"
   },
   {
    "_id": "598bec3cf5528800083d58b7",
    "clusterName": "2",
    "clusterId": "598bd034f5528800083d58a2",
    "operateType": "create_service",
    "queryType": "service",
    "userName": "sysadmin",
    "user_id": "5982dba001bf6000084d02ce",
    "tenant_id": "5982dba001bf6000084d02ca",
    "status": "success",
    "comments": "9",
    "time_create": "2017-08-10T05:16:44Z",
    "time_update": "2017-08-10T05:16:44Z"
   },
   {
    "_id": "598bec7ef5528800083d58b8",
    "clusterName": "2",
    "clusterId": "598bd034f5528800083d58a2",
    "operateType": "create_component",
    "queryType": "service",
    "userName": "sysadmin",
    "user_id": "5982dba001bf6000084d02ce",
    "tenant_id": "5982dba001bf6000084d02ca",
    "status": "success",
    "comments": "/9/4",
    "time_create": "2017-08-10T05:17:50Z",
    "time_update": "2017-08-10T05:17:50Z"
   },
   {
    "_id": "598bec98f5528800083d58b9",
    "clusterName": "2",
    "clusterId": "598bd034f5528800083d58a2",
    "operateType": "start_component",
    "queryType": "service",
    "userName": "sysadmin",
    "user_id": "5982dba001bf6000084d02ce",
    "tenant_id": "5982dba001bf6000084d02ca",
    "status": "success",
    "comments": "/9/4",
    "time_create": "2017-08-10T05:18:16Z",
    "time_update": "2017-08-10T05:18:16Z"
   },
   {
    "_id": "598bf82b53db6100084041cc",
    "clusterName": "2",
    "clusterId": "598bd034f5528800083d58a2",
    "operateType": "stop_component",
    "queryType": "service",
    "userName": "sysadmin",
    "user_id": "5982dba001bf6000084d02ce",
    "tenant_id": "5982dba001bf6000084d02ca",
    "status": "success",
    "comments": "/9/4",
    "time_create": "2017-08-10T06:07:39Z",
    "time_update": "2017-08-10T06:07:39Z"
   },
   {
    "_id": "598c23dbe68b1f00089cd4a3",
    "clusterName": "2",
    "clusterId": "598bd034f5528800083d58a2",
    "operateType": "create_service",
    "queryType": "service",
    "userName": "sysadmin",
    "user_id": "5982dba001bf6000084d02ce",
    "tenant_id": "5982dba001bf6000084d02ca",
    "status": "success",
    "comments": "a",
    "time_create": "2017-08-10T09:14:03Z",
    "time_update": "2017-08-10T09:14:03Z"
   }
  ]
 }
```

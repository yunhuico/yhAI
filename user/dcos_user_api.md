#用户模块api更改

##一 创建用户api更改

REQUEST

```
Url :
Header: X_Auth_Token
POST /v1/user
Body {}
```

EXAMPLE

```
curl -X POST -H "Content-Type:  application/json" -H "X-Auth-Token: 598d0bbf88a87300081c8a26" http://192.168.10.117:10001/v1/user -d @use.json
```
  
修改前的json：   
```
{
"username":"test",
"email":"test@qq.com",
"company": "company”
}
```

修改后的use.json如下：   
```
{
"username":"test",
"email":"test@qq.com",
"company": "company”,
"roleType":"sysadmin"
}
```

RESPONSE:

```
{
  "success": true,
  "data": {
   "created": true,
   "url": "/v1/user/598d0bf188a87300081c8a28"
  }
}
```

##二 更改用户api修改

REQUEST

```
Url :
Header: X_Auth_Token
PUT /v1/user/{user_id}
Body {}
```

EXAMPLE

```
Curl -X PUT -H "Content-Type: application/json" -H "X-Auth-Token: 598d0bbf88a87300081c8a26" http://192.168.10.117:10001/v1/user/5982dba001bf6000084d02ce -d @userA.json
```

其中userA.json 如下
```
{
"username":"test",
"email":"test@qq.com",
"company": "company”,
"roleType":"admin"
}
```

RESPONSE

```
{
  "success": true,
  "data": {
   "created": false,
   "url": "/v1/user/598d14a7fed77500088f6fd8"
  }
}
```
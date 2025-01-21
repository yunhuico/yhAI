# linker DCOS-ui

## preinstall

 * node.js
 * npm
 * docker
 * docker-compose
 * grunt

### Initial local envorinment

```
cd docker
docker-compose -f locallinkerdcos.yaml up -d
```

You will see

```
docker_mongodb_1 is up-to-date
Recreating docker_deployer_1
docker_usermgmt_1 is up-to-date
Creating docker_clustermgmt_1
```

### Start server

```
cd portal-server
node server.js
```

the message is shown,

```
linker portal started, listen port 3000.
```

Then open browser, and input account and password

sysadmin / password

```
open http://localhost:3000/
```

### Check account

Then you can test account is created sucsessful.

```
curl -X POST -d"{\"username\":\"sysadmin\",\"password\":\"password\"}" http://localhost:10001/v1/user/login
```

Result is

```
{
  "success": true,
  "data": {
   "id": "5857a02f731c25000864d226",
   "userid": "5857a023731c25000864d225"
  }
}
```

* Summary of set up
* Configuration
* Dependencies
* Database configuration
* How to run tests
* Deployment instructions

### Contribution guidelines ###

* Writing tests
* Code review
* Other guidelines

### Who do I talk to? ###

* Repo owner or admin
* Other community or team contact
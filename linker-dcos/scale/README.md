linker scale ------ A tool to do scale out and scale in operation to a specified linker dcos


Build binary
--------------------
build.sh

the binary file(named cluster_scale) and configuration file are located on parent's bin folder



Usage
--------------------

## Check all command destription
cluster_scale -help
 Usage of ./cluster_scale:
  -addnumber int
    	The amount of added node
  -clusterlb string
    	The destination management cluster lb address, such as: 172.17.0.10 (default "127.0.0.1")
  -clustername string
    	The destination cluster name
  -config string (option)
    	The configuration file (default "./linkerdcos_scale.properties")
  -operation string
    	Operation type: add or remove
  -password string
    	The password of user for linkerdcos platform
  -removenodes string
    	The nodes' ip that will be removed from linkerdcos platform. Multiple values are seperated by comma
  -username string
    	The valid username of linkerdcos platform


## scale out operation
sudo ./cluster_scale -clusterlb 10.140.0.18 -clustername linkerdcos  -addnumber 1 -operation add -username sysadmin -password password

## scale in operation
./cluster_scale -clusterlb 10.140.0.18 -clustername linkerdcos  -removenodes 10.140.0.20  -operation remove -username sysadmin -password password


Note
-----------------
1. only support linkerdcos whose provider type is aws and google, other provider type is not supported
2. since the backend "add node" and "remove node" operations are asynchronous, a success return does not mean operation completed, you have to
   check it from UI or backend log

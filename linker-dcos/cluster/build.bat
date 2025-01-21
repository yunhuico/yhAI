@echo off

set GO_PATH=%cd%
echo GO_PATH=%GO_PATH%

echo Start to go third party code from github.com ...
echo Downloading logrus ...
go get -v -u github.com/Sirupsen/logrus
echo Downloading properties
go get -v -u github.com/magiconair/properties
echo Downloading go-restful ...
go get -v -u github.com/emicklei/go-restful
echo Downloading mejson ...
go get -v -u github.com/compose/mejson
echo Downloading mgo ...
go get -v -u gopkg.in/mgo.v2
echo Downloading go-dockerclient ...
go get -v -u github.com/fsouza/go-dockerclient
echo Downloading jsonq ...
go get -v -u github.com/jmoiron/jsonq
echo Downloading uuid ...
go get -v -u github.com/pborman/uuid
echo Downloading gomail ...
go get -v -u gopkg.in/gomail.v2
echo "Downloading assert ..."
go get -v -u github.com/bmizerany/assert

echo Copying linkernetworks's libs ...
xcopy /e /y /r /i ..\linker_common_lib\linkernetworks.com src\linkernetworks.com\

rem echo Chean bin
rem del /s /f /q bin\*.*

echo Start to build linker cluster ...
go build -a -o bin/cluster_mgmt.exe linkernetworks.com/linker_cluster

echo Copying properties file to bin/ ...
copy /y .\cluster_mgmt.properties .\bin\cluster_mgmt.properties
copy /y .\clusterpolicy.json .\bin\clusterpolicy.json

rem fetch or update Swagger UI
if exist bin\swagger-ui (git clone https://github.com/wordnik/swagger-ui.git bin\swagger-ui)
pause

echo cp bin/cluster_mgmt 
rem root@ansible:/root/Linker_Ansible/linker_ansible_repo/Linker_Mesos_Cluster/roles/controller/files/



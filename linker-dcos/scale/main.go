package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/magiconair/properties"

	"linkernetworks.com/dcos-backend/scale/common"
	"linkernetworks.com/dcos-backend/scale/scale"
)

var (
	Props *properties.Properties

	PropertiesFile  = flag.String("config", "./linkerdcos_scale.properties", "The configuration file")
	ClusterLBFlag   = flag.String("clusterlb", "127.0.0.1", "The management cluster lb address, such as: 172.17.0.10")
	UsernameFlag    = flag.String("username", "sysadmin", "The valid username of linkerdcos platform")
	PasswordFlag    = flag.String("password", "password", "The password of user for linkerdcos platform")
	ClusterNameFlag = flag.String("clustername", "", "The destination cluster name")
	OperationFlag   = flag.String("operation", "", "Operation type: add or remove")
	RemoveNodesFlag = flag.String("removenodes", "", "The nodes' ip that will be removed from linkerdcos platform. Values are seperated by comma")
	AddNumberFlag   = flag.Int("addnumber", 0, "The amount of added node")

	//current only support new mode, does support add customized node
	// AddModeFlag = flag.String("addmode", "new", "Add node mode. create a new node or use exist node, current only support new mode")

	//EnginOpt and NodeAttribute can be stored in config file as it's related with business
	RemoveNodeIpList = []string{}
)

func init() {
	// get configuration
	flag.Parse()

	fmt.Printf("PropertiesFile is %s\n", *PropertiesFile)
	var err error
	if Props, err = properties.LoadFile(*PropertiesFile, properties.UTF8); err != nil {
		fmt.Printf("[error] Unable to read properties:%v\n", err)
	}

	// set log configuration
	// Log as JSON instead of the default ASCII formatter.
	switch Props.GetString("logrus.formatter", "") {
	case "text":
		logrus.SetFormatter(&logrus.TextFormatter{})
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{})
	default:
		logrus.SetFormatter(&logrus.TextFormatter{})
	}

	// Output to stderr instead of stdout, could also be a file.
	logrus.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	level, err := logrus.ParseLevel(Props.GetString("logrus.level", "info"))
	if err != nil {
		fmt.Printf("parse log level err is %v\n", err)
		fmt.Printf("using default level is %v \n", logrus.InfoLevel)
		level = logrus.InfoLevel
	}

	logrus.SetLevel(level)

}

//validat request parameters
func validateParameter() (err error) {
	logrus.Infof("validate the paramters of current request")
	if len(*UsernameFlag) <= 0 || len(*PasswordFlag) <= 0 || len(*ClusterNameFlag) <= 0 || len(*OperationFlag) <= 0 {
		logrus.Errorf("invalid parameter! username, password, clustername and operation can not be null!")
		return errors.New("invalid parameter")
	}

	if strings.EqualFold(*OperationFlag, common.OPERATION_REMOVE) {
		if len(*RemoveNodesFlag) <= 0 {
			errMsg := "remove node can not be null!"
			logrus.Errorf(errMsg)
			return errors.New(errMsg)
		}

		RemoveNodeIpList = strings.Split(*RemoveNodesFlag, ",")
		logrus.Infof("the requested removed nodes are %v", RemoveNodeIpList)

	} else if strings.EqualFold(*OperationFlag, common.OPERATION_ADD) {
		// if !strings.EqualFold(*AddModeFlag, common.OPERATION_ADDMODE_NEW){
		// 	 errMsg:= "currently only support new add mode! please try again with add mode --- new"
		// 	 logrus.Errorln(errMsg)
		// 	 return errors.New(errMsg)
		// }

		if *AddNumberFlag <= 0 {
			errMsg := "add node number can not less than 0!"
			logrus.Errorln(errMsg)
			return errors.New(errMsg)
		}
	} else {
		logrus.Errorf("not supported operation %s", *OperationFlag)
		return errors.New("not supported operation")
	}

	return

}

func main() {

	err := validateParameter()
	if err != nil {
		logrus.Errorln("invalid parameters! please check it again!")
		os.Exit(1)
	}

	if strings.EqualFold(*OperationFlag, common.OPERATION_ADD) {
		err := scale.ScaleOut(*UsernameFlag, *PasswordFlag, *ClusterLBFlag, *ClusterNameFlag, *AddNumberFlag)
		if err != nil {
			logrus.Errorf("send scale out request error %v", err)
			os.Exit(1)
		}
	} else if strings.EqualFold(*OperationFlag, common.OPERATION_REMOVE) {
		err := scale.ScaleIn(*UsernameFlag, *PasswordFlag, *ClusterLBFlag, *ClusterNameFlag, RemoveNodeIpList)
		if err != nil {
			logrus.Errorf("send scale out request error %v", err)
			os.Exit(1)
		}
	} else {
		logrus.Errorf("not supported operation mode %s", *OperationFlag)
		os.Exit(1)
	}

}

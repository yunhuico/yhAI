package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
	swagger "github.com/emicklei/go-restful-swagger12"
	"github.com/magiconair/properties"
	"linkernetworks.com/dcos-backend/common/persistence/dao"
	"linkernetworks.com/dcos-backend/common/persistence/session"
	"linkernetworks.com/dcos-backend/user/api/usermgmt"
	"linkernetworks.com/dcos-backend/user/common"
)

var (
	PROPS          *properties.Properties
	PROPERTIESFILE = flag.String("config", "usermgmt.properties", "the configuration file")
	MONGOALIAS     string
	SWAGGERPATH    string
	LINKERICON     string
)

func init() {
	// get configuration
	flag.Parse()
	fmt.Printf("propertiesFile is %s\n", *PROPERTIESFILE)
	var err error
	if PROPS, err = properties.LoadFile(*PROPERTIESFILE, properties.UTF8); err != nil {
		fmt.Printf("[error] Unable to read properties:%v\n", err)
	}

	// set log configuration
	// Log as JSON instead of the default ASCII formatter.
	switch PROPS.GetString("logrus.formatter", "") {
	case "text":
		logrus.SetFormatter(&logrus.TextFormatter{})
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{})
	default:
		logrus.SetFormatter(&logrus.TextFormatter{})
	}

	// Output to stderr instead of stdout, could also be a file.
	logFile := PROPS.GetString("logrus.file", "/var/log/linkerdcos_userMgmt.log")
	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		fmt.Printf("error opening file %v\n", err)
		f, err = os.OpenFile("linkerdcos_userMgmt.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
		if err != nil {
			fmt.Printf("still failed to open log file linkerdcos_userMgmt.log %v\n", err)
		}
	}
	logrus.SetOutput(f)

	// Only log the warning severity or above.
	level, err := logrus.ParseLevel(PROPS.GetString("logrus.level", "info"))
	if err != nil {
		fmt.Printf("parse log level err is %v\n", err)
		fmt.Printf("using default level is %v \n", logrus.InfoLevel)
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)

}

func main() {
	// Swagger configuration
	SWAGGERPATH = PROPS.GetString("swagger.path", "")
	LINKERICON = filepath.Join(SWAGGERPATH, "images/mora.ico")

	// New, shared session manager, seprate DAO layer
	MONGOALIAS = PROPS.GetString("db.alias", "dev")
	sessMng := session.NewSessionManager(PROPS.FilterPrefix("mongod."), MONGOALIAS)
	defer sessMng.CloseAll()
	dao.DAO = &dao.Dao{SessMng: sessMng, MongoAlias: MONGOALIAS}
	common.UTIL = &common.Util{Props: PROPS}

	// accept and respond in JSON unless told otherwise
	restful.DefaultRequestContentType(restful.MIME_JSON)
	restful.DefaultResponseContentType(restful.MIME_JSON)
	// gzip if accepted
	restful.DefaultContainer.EnableContentEncoding(true)
	// faster router
	restful.DefaultContainer.Router(restful.CurlyRouter{})
	// no need to access body more than once
	// restful.SetCacheReadEntity(false)
	// API Cross-origin requests
	apiCors := PROPS.GetBool("http.server.cors", false)

	//UserMgmt API
	usermgmt.Register(restful.DefaultContainer, apiCors)

	hostname, err := os.Hostname()
	if err != nil {
		logrus.Errorf("get hostname err is %+v", err)
	}
	endpoint := hostname + ":" + PROPS.MustGet("http.server.port")
	// Serve favicon.ico
	http.HandleFunc("/favion.ico", icon)

	basePath := PROPS.MustGet("http.server.host") + ":" + PROPS.MustGet("http.server.port")
	isHttpsEnabled := PROPS.GetBool("http.server.https.enabled", false)
	logrus.Debugf("httpEnabled=%v", isHttpsEnabled)
	if isHttpsEnabled {
		// Register Swagger UI
		swagger.InstallSwaggerService(swagger.Config{
			WebServices:     restful.RegisteredWebServices(),
			WebServicesUrl:  "https://" + endpoint,
			ApiPath:         "/apidocs.json",
			SwaggerPath:     SWAGGERPATH,
			SwaggerFilePath: PROPS.GetString("swagger.file.path", ""),
		})

		// If swagger is not on `/` redirect to it
		if SWAGGERPATH != "/" {
			http.HandleFunc("/", index)
		}
		crtFile := PROPS.MustGet("http.server.https.crt")
		keyFile := PROPS.MustGet("http.server.https.key")
		logrus.Infof("ready to serve on %s by using TLS", basePath)
		logrus.Fatal(http.ListenAndServeTLS(basePath, crtFile, keyFile, nil))

	} else {
		// Register Swagger UI
		swagger.InstallSwaggerService(swagger.Config{
			WebServices:     restful.RegisteredWebServices(),
			WebServicesUrl:  "http://" + endpoint,
			ApiPath:         "/apidocs.json",
			SwaggerPath:     SWAGGERPATH,
			SwaggerFilePath: PROPS.GetString("swagger.file.path", ""),
		})

		// If swagger is not on `/` redirect to it
		if SWAGGERPATH != "/" {
			http.HandleFunc("/", index)
		}
		logrus.Infof("ready to serve on %s", basePath)
		logrus.Fatal(http.ListenAndServe(basePath, nil))
	}
}

// If swagger is not on `/` redirect to it
func index(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, SWAGGERPATH, http.StatusMovedPermanently)
}
func icon(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, LINKERICON, http.StatusMovedPermanently)
}

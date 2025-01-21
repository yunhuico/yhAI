package main

import (
	"flag"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
	swagger "github.com/emicklei/go-restful-swagger12"
	"github.com/magiconair/properties"
	"linkernetworks.com/dcos-backend/deployer/api/documents"
	"linkernetworks.com/dcos-backend/deployer/common"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

var (
	Props          *properties.Properties
	PropertiesFile = flag.String("config", "dcos_deploy.properties", "the configuration file")
	ZkFlag         = flag.String("zk", "", "zk url")
	HostnameFlag   = flag.String("hostname", "", "hostname")
	MongoAlias     string
	SwaggerPath    string
	LinkerIcon     string
	ZK             string
	Hostname       string
	// TODO:  Mongo          string
)

func init() {
	// get configuration
	flag.Parse()
	ZK = *ZkFlag
	Hostname = *HostnameFlag
	fmt.Printf("PropertiesFile is %s\n", *PropertiesFile)
	var err error
	if Props, err = properties.LoadFile(*PropertiesFile, properties.UTF8); err != nil {
		fmt.Printf("[error] Unable to read properties:%v\n", err)
	}

	common.UTIL = &common.Util{Props}

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
	// Use the Airbrake hook to report errors that have Error severity or above to
	// an exception tracker. You can create custom hooks, see the Hooks section.
	// log.AddHook(airbrake.NewHook("https://example.com", "xyz", "development"))

	// Output to stderr instead of stdout, could also be a file.
	logFile := Props.GetString("logrus.file", "/var/log/linkerdcos_deployMgmt.log")
	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		fmt.Printf("error opening file %v", err)
		f, err = os.OpenFile("linkerdcos_deployMgmt.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
		if err != nil {
			fmt.Printf("still failed to open log file linkerdcos_deployMgmt.log %v", err)
		}
	}
	logrus.SetOutput(f)

	// Only log the warning severity or above.
	level, err := logrus.ParseLevel(Props.GetString("logrus.level", "info"))
	if err != nil {
		fmt.Printf("parse log level err is %v\n", err)
		fmt.Printf("using default level is %v \n", logrus.InfoLevel)
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)

	//set GOMAXPROCS for goroutine parallel
	runtime.GOMAXPROCS(runtime.NumCPU())

}

func main() {
	// Swagger configuration
	SwaggerPath = Props.GetString("swagger.path", "")
	LinkerIcon = filepath.Join(SwaggerPath, "images/mora.ico")

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
	apiCors := Props.GetBool("http.server.cors", false)
	// Documents API
	documents.Register(restful.DefaultContainer, apiCors)
	endpoint := Hostname + ":" + Props.MustGet("http.server.port")
	// Serve favicon.ico
	http.HandleFunc("/favion.ico", icon)
	basePath := Props.MustGet("http.server.host") + ":" + Props.MustGet("http.server.port")
	// basePath := ":" + Props.MustGet("http.server.port")
	isHttpsEnabled := Props.GetBool("http.server.https.enabled", false)
	logrus.Debugf("httpEnabled=%v", isHttpsEnabled)
	if isHttpsEnabled {
		// Register Swagger UI
		swagger.InstallSwaggerService(swagger.Config{
			WebServices:     restful.RegisteredWebServices(),
			WebServicesUrl:  "https://" + endpoint,
			ApiPath:         "/apidocs.json",
			SwaggerPath:     SwaggerPath,
			SwaggerFilePath: Props.GetString("swagger.file.path", ""),
		})

		// If swagger is not on `/` redirect to it
		if SwaggerPath != "/" {
			http.HandleFunc("/", index)
		}
		crtFile := Props.MustGet("http.server.https.crt")
		keyFile := Props.MustGet("http.server.https.key")
		logrus.Infof("ready to serve on %s by using TLS", basePath)
		logrus.Fatal(http.ListenAndServeTLS(basePath, crtFile, keyFile, nil))

	} else {
		// Register Swagger UI
		swagger.InstallSwaggerService(swagger.Config{
			WebServices:     restful.RegisteredWebServices(),
			WebServicesUrl:  "http://" + endpoint,
			ApiPath:         "/apidocs.json",
			SwaggerPath:     SwaggerPath,
			SwaggerFilePath: Props.GetString("swagger.file.path", ""),
		})

		// If swagger is not on `/` redirect to it
		if SwaggerPath != "/" {
			http.HandleFunc("/", index)
		}
		logrus.Infof("ready to serve on %s", basePath)
		logrus.Fatal(http.ListenAndServe(basePath, nil))
	}
	// router := NewRouter().StrictSlash(true)
	// logrus.Fatal(http.ListenAndServe(Props["http.server.host"]+":"+Props["http.server.port"], router))
}

// If swagger is not on `/` redirect to it
func index(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, SwaggerPath, http.StatusMovedPermanently)
}
func icon(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, LinkerIcon, http.StatusMovedPermanently)
}

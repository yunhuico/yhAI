package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
	swagger "github.com/emicklei/go-restful-swagger12"
	"github.com/magiconair/properties"
	"linkernetworks.com/dcos-backend/cluster/api/documents"
	"linkernetworks.com/dcos-backend/cluster/common"
	"linkernetworks.com/dcos-backend/common/persistence/dao"
	"linkernetworks.com/dcos-backend/common/persistence/session"
)

var (
	Props          *properties.Properties
	PropertiesFile = flag.String("config", "cluster_mgmt.properties", "the configuration file")
	HostnameFlag   = flag.String("hostname", "", "hostname")
	MongoAlias     string
	SwaggerPath    string
	LinkerIcon     string
	Hostname       string
	// TODO:  Mongo          string
)

func init() {
	// get configuration
	flag.Parse()
	Hostname = *HostnameFlag
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
	default:
		logrus.SetFormatter(&logrus.TextFormatter{})
	}
	// Use the Airbrake hook to report errors that have Error severity or above to
	// an exception tracker. You can create custom hooks, see the Hooks section.
	// log.AddHook(airbrake.NewHook("https://example.com", "xyz", "development"))

	// Output to stderr instead of stdout, could also be a file.
	// logrus.SetOutput(f)
	logFile := Props.GetString("logrus.file", "/var/log/clusterMgmt.log")
	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		fmt.Printf("error opening file %v", err)
		f, err = os.OpenFile("linkerdcos_clusterMgmt.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
		if err != nil {
			fmt.Printf("still failed to open log file clusterMgmt.log %v", err)
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

}

func main() {
	// Swagger configuration
	SwaggerPath = Props.GetString("swagger.path", "")
	LinkerIcon = filepath.Join(SwaggerPath, "images/mora.ico")

	// New, shared session manager, seprate DAO layer
	MongoAlias = Props.GetString("db.alias", "dev")
	sessMng := session.NewSessionManager(Props.FilterPrefix("mongod."), MongoAlias)
	defer sessMng.CloseAll()
	dao.DAO = &dao.Dao{SessMng: sessMng, MongoAlias: MongoAlias}
	fmt.Println(dao.DAO.MongoAlias)

	lbClient := common.LbClient{
		Host: Props.MustGetString("lb.host"),
	}
	common.UTIL = &common.Util{LbClient: &lbClient, Props: Props}

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

	// Check hostname flag, if hostname is set by flag, using hostname
	if strings.TrimSpace(Hostname) == "" {
		hostname, err := os.Hostname()
		if err != nil {
			logrus.Errorf("get hostname err is %+v", err)
		}
		Hostname = hostname
	}
	endpoint := Hostname + ":" + Props.MustGet("http.server.port")
	// Serve favicon.ico
	http.HandleFunc("/favion.ico", icon)

	basePath := Props.MustGet("http.server.host") + ":" + Props.MustGet("http.server.port")
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
}

// If swagger is not on `/` redirect to it
func index(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, SwaggerPath, http.StatusMovedPermanently)
}
func icon(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, LinkerIcon, http.StatusMovedPermanently)
}

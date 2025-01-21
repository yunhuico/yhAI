package session

import (
	"bytes"
	"errors"
	"github.com/Sirupsen/logrus"
	"github.com/magiconair/properties"
	"gopkg.in/mgo.v2"
	"strconv"
	"strings"
	"sync"
	"time"
)

// MongoDB Session Manager
type SessionManager struct {
	defaultAlias string
	configMap    map[string]*properties.Properties
	sessions     map[string]*mgo.Session
	accessLock   *sync.RWMutex
}

// Creates a new Session Manager using `props` as configuration.
// For more info about properties check `mongod.*` section in `mora.properties`
func NewSessionManager(props *properties.Properties, defaultAlias string) *SessionManager {
	sess := &SessionManager{
		defaultAlias: defaultAlias,
		configMap:    make(map[string]*properties.Properties),
		sessions:     make(map[string]*mgo.Session),
		accessLock:   &sync.RWMutex{},
	}
	sess.SetConfig(props)
	return sess
}

// Returns slice containing all configured aliases
func (s *SessionManager) GetAliases() []string {
	aliases := []string{}
	for k := range s.configMap {
		aliases = append(aliases, k)
	}
	return aliases
}

func (s *SessionManager) GetDefault() (*mgo.Session, bool, error) {
	return s.Get(s.defaultAlias)
}

// Gets session for alias
func (s *SessionManager) Get(alias string) (*mgo.Session, bool, error) {
	// Get alias configurations
	config, err := s.GetConfig(alias)
	if err != nil {
		return nil, false, err
	}

	var uri string
	var hostport string
	var sessionId string
	if uriConfig := strings.Trim(config.GetString("uri", ""), " "); len(uriConfig) != 0 {
		// uri = config.GetString("uri", "")
		sessionId = config.GetString("uri", "")
		uri = buildURI(config)
	} else {
		hostport = config.MustGet("host") + ":" + config.MustGet("port")
		sessionId = hostport
	}

	// Check if session already exists
	s.accessLock.RLock()
	existing := s.sessions[sessionId]
	s.accessLock.RUnlock()

	// Clone and return if sessions exists
	if existing != nil {
		return existing.Copy(), true, nil
	}

	// Get timeout from configuration
	s.accessLock.Lock()
	timeout := 0
	if timeoutConfig := strings.Trim(config.GetString("timeout", ""), " "); len(timeoutConfig) != 0 {
		timeout, err = strconv.Atoi(timeoutConfig)
		if err != nil {
			return nil, false, err
		}
	}

	// Connect to database server
	info("connecting to [%s=%s] with timeout [%d seconds]", config.GetString("alias", ""), sessionId, timeout)
	var newSession *mgo.Session
	if uri != "" {
		newSession, err = mgo.DialWithTimeout(uri, time.Duration(timeout)*time.Second)
	} else {
		dialInfo := mgo.DialInfo{
			Addrs:    []string{hostport},
			Direct:   true,
			Database: config.GetString("database", ""),
			Username: config.GetString("username", ""),
			Password: config.GetString("password", ""),
			Timeout:  time.Duration(timeout) * time.Second,
		}
		newSession, err = mgo.DialWithInfo(&dialInfo)
	}
	if err != nil {
		info("unable to connect to [%s] because:%v", sessionId, err)
		newSession = nil
	} else {
		s.sessions[sessionId] = newSession
	}
	s.accessLock.Unlock()
	return newSession, false, err
}

// Closes session based on `uri` or `host:port`
func (s *SessionManager) Close(sessionId string) {
	s.accessLock.Lock()
	if existing := s.sessions[sessionId]; existing != nil {
		existing.Close()
		delete(s.sessions, sessionId)
	}
	s.accessLock.Unlock()
}

// Closes all sessions.
func (s *SessionManager) CloseAll() {
	info("closing all sessions: ", len(s.sessions))
	s.accessLock.Lock()
	for _, each := range s.sessions {
		each.Close()
	}
	s.accessLock.Unlock()
}

// Set's session manager configuration.
func (s *SessionManager) SetConfig(props *properties.Properties) {
	for _, k := range props.Keys() {
		parts := strings.Split(k, ".")
		alias := parts[1]
		config := s.configMap[alias]
		if config == nil {
			config = properties.NewProperties()
			config.Set("alias", alias)
			s.configMap[alias] = config
		}
		config.Set(parts[2], props.MustGet(k))
	}
}

// Get's session configurations by alias.
func (s *SessionManager) GetConfig(alias string) (*properties.Properties, error) {
	if config := s.configMap[alias]; config != nil {
		return config, nil
	}
	return nil, errors.New("Unknown alias: " + alias)
}

//"mongodb://linker:password@localhost:27017,172.17.2.44:27017/linker"
func buildURI(config *properties.Properties) string {
	uri := config.GetString("uri", "")
	if len(uri) <= 0 {
		return uri
	}

	var commandTextBuffer bytes.Buffer
	username := config.GetString("username", "")
	password := config.GetString("password", "")
	// port := config.GetString("port", "27017")
	database := config.GetString("database", "")

	prefix := "mongodb://"
	commandTextBuffer.WriteString(prefix)
	commandTextBuffer.WriteString(username + ":")
	commandTextBuffer.WriteString(password + "@")

	rs := []rune(uri)
	ipPart := string(rs[len(prefix):len(uri)])
	ips := strings.Split(ipPart, ",")
	lenth := len(ips)
	for i := 0; i < lenth-1; i++ {
		ip := ips[i] + ","
		commandTextBuffer.WriteString(ip)
	}
	lastIP := ips[lenth-1] + "/" + database
	commandTextBuffer.WriteString(lastIP)

	return commandTextBuffer.String()
}

// Log wrapper
func info(template string, values ...interface{}) {
	logrus.Infof("[mongo] "+template+"\n", values...)
	// log.Printf("[mongo] "+template+"\n", values...)
}

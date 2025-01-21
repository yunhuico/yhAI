package config

import (
	"errors"
	"fmt"
	"net/mail"
	"os"
	"strings"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/featureflag"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/smtp"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/trigger"

	"github.com/pelletier/go-toml/v2"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/cache"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/sentrygo"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/work"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/worker"
)

// Config defines UltraFox config fields.
//
// When adding or modifying Config, please, modify ExampleConfig() with sane default values,
// which contributes to the default config file.
//
// Use comment struct tag to generate field comment during `ultrafox init`
// For reference on struct tags:
// https://pkg.go.dev/github.com/pelletier/go-toml/v2#hdr-Struct_tags
type Config struct {
	// Enable profiler? It listens at port 6060
	Profiler bool `comment:"Enable profiler? It listens at port 6060"`
	// Used to encrypt/decrypt user credentials, must be of 32 characters and generated from random source of good quality
	CryptoKey string `comment:"used to encrypt/decrypt user credentials, must be of 32 characters and generated from random source of good quality"`
	DocsHost  string `comment:"There are many links to document website, so must configure a document host."`
	// Development only: expose local instance to public internet via frp
	DevelopmentTunnel Tunnel `comment:"Development only: expose local instance to public internet via frp"`
	// Database type and DSN
	Database model.DBConfig `comment:"Database type and DSN"`
	// Redis config
	Redis cache.RedisConfig `comment:"redis related config"`
	// API server service related config
	APIServer APIServer `comment:"API server related config"`
	// OAuth2 providers to log UltraFox users in
	PassportVendors model.PassportVendors `comment:"OAuth2 providers to log UltraFox users in"`
	// Webhook service related config
	Webhook Webhook `comment:"Webhook service related config"`
	// Dkron supports cron triggers of user workflows
	Dkron trigger.DkronConfig `comment:"Dkron supports cron triggers of user workflows"`
	// OAuth2 credentials to third party APPs that UltraFox officially operates
	Credentials model.OfficialCredentials `comment:"OAuth2 credentials to third party APPs that UltraFox officially operates"`
	// Where pending workflow execution(aka, work) should go, typically used by webhook service. The other end is WorkSource.
	WorkSink work.ProducerConfig `comment:"Where pending workflow execution(aka, work) should go, typically used by webhook service. The other end is WorkSource"`
	// Where work gets actually executed
	Worker worker.Config `comment:"Where work gets actually executed"`
	// SMTP service to enable UltraFox to send notification mail
	NotificationSMTP smtp.SenderConfig `comment:"SMTP service to enable UltraFox to send notification mail"`
	// Error tracing
	Sentry sentrygo.Config `comment:"Error tracing"`
	// Private Beta related configs, which should be removed after the beta period
	BetaConfig apiserver.BetaConfig `comment:"Private Beta related configs, which should be removed after the beta period"`
	// Swagger endpoints related configs
	SwaggerConfig apiserver.SwaggerConfig `comment:"Swagger endpoints related configs"`
	// FeatureFlagConfig, ultrafox integrate https://github.com/Unleash/unleash as basic component.
	FeatureFlagConfig featureflag.Config `comment:"Feature flag config, ultrafox integrate https://github.com/Unleash/unleash as basic component."`
}

// ExampleConfig contributes to the default config file.
func ExampleConfig() (cfg Config, err error) {
	const workerTopicName = "work"
	var kafkaAddr = []string{"kafka:9092"}

	cryptoKey, err := utils.RandStr(32) // AES key size
	if err != nil {
		err = fmt.Errorf("randomizing cryptoKey: %w", err)
		return
	}
	userApproveBearerToken, err := utils.RandStr(40)
	if err != nil {
		err = fmt.Errorf("randomizing userApproveBearerToken: %w", err)
		return
	}
	subdomainSuffix, err := utils.RandStr(8)
	if err != nil {
		err = fmt.Errorf("randomizing subdomainSuffix: %w", err)
		return
	}
	swaggerPassword, err := utils.RandStr(40)
	if err != nil {
		err = fmt.Errorf("randomizing swaggerPassword: %w", err)
		return
	}

	cfg = Config{
		Profiler:  false,
		CryptoKey: cryptoKey,
		DocsHost:  "http://docs.ultrafox.jihulab.com",
		DevelopmentTunnel: Tunnel{
			Server:          "ultrafox-dev-tunnel.jihulab.com:7000",
			Domain:          "ultrafox-dev-tunnel.jihulab.com",
			Token:           "ultrafox-is-awesome!!!",
			SubdomainSuffix: strings.ToLower(subdomainSuffix),
		},
		Database: model.DBConfig{
			Dialect: "sqlite",
			DSN:     "file:/tmp/ultrafox_sqlite.db?_busy_timeout=5000",
		},
		Redis: cache.RedisConfig{
			Addr: "redis:6379",
			DB:   0,
		},
		APIServer: APIServer{
			Port:          8010,
			AuditResponse: false,
			ExternalHost:  "http://localhost:8010",
		},
		PassportVendors: model.PassportVendors{
			{
				Enabled:      true,
				Name:         "Jihulab",
				ClientID:     "client id here",
				ClientSecret: "client secret here",
				BaseURL:      "https://jihulab.com",
			},
			{
				Enabled:      true,
				Name:         "Gitlab",
				ClientID:     "client id here",
				ClientSecret: "client secret here",
				BaseURL:      "https://gitlab.com",
			},
		},
		Webhook: Webhook{
			ExternalHost:             "http://localhost:8011",
			Port:                     8011,
			TimeoutSecondsPerRequest: 10,
			SlackWebhook: SlackWebhookConfig{
				SigningSecret: "slack signing secret here",
			},
		},
		Dkron: trigger.DkronConfig{
			DkronInternalHost:   "http://localhost:18100",
			WebhookInternalHost: "http://localhost:18200",
			JobTags: map[string]string{
				"dc": "dc1:1",
			},
		},
		Credentials: model.OfficialCredentials{
			{
				Name:    "jihulab",
				Adapter: "ultrafox/gitlab",
				Type:    model.CredentialTypeOAuth2,
				MetaData: map[string]string{
					"server": "https://jihulab.com",
				},
				OAuth2Config: model.OfficialOAuth2Config{
					ClientID:     "client id here",
					ClientSecret: "client secret here",
					RedirectURL:  "http://localhost:8010/api/v1/credentials/oauth2/callback",
				},
			},
			{
				Name:    "dingtalkCorpBot",
				Adapter: "ultrafox/dingtalkCorpBot",
				Type:    model.CredentialTypeCustom,
				MetaData: map[string]string{
					"appKey":    "appKey here",
					"appSecret": "appSecret here",
				},
			},
		},
		WorkSink: work.ProducerConfig{
			KafkaTopic:     workerTopicName,
			KafkaAddresses: kafkaAddr,
			SASL: work.SASLCredential{
				Username: "",
				Password: "",
			},
			Compression: "none",
		},
		Worker: worker.Config{
			Concurrency:              50,
			GlobalMaxSteps:           100,
			GlobalMaxDurationSeconds: 60,
			WorkSource: work.ConsumerConfig{
				KafkaTopic:     workerTopicName,
				KafkaAddresses: kafkaAddr,
				SASL: work.SASLCredential{
					Username: "",
					Password: "",
				},
				ConsumerGroup: "worker",
			},
		},
		NotificationSMTP: smtp.SenderConfig{
			Auth: smtp.AuthOpt{
				SecureMethod: smtp.SSL,
				Host:         "localhost",
				Port:         465,
				Username:     "jim@localhost",
				Password:     "jim's super secret password",
			},
			From: mail.Address{
				Name:    "Jim",
				Address: "jim@localhost",
			},
		},
		Sentry: sentrygo.Config{
			Enabled:     false,
			Debug:       false,
			DSN:         "https://example.ingest.sentry.io/123456789",
			Environment: "devel",
		},
		BetaConfig: apiserver.BetaConfig{
			InvitationSignUpSheetURL: "https://example.com",
			APIBearerToken:           userApproveBearerToken,
		},
		SwaggerConfig: apiserver.SwaggerConfig{
			BasicAuth: false,
			Accounts: map[string]string{
				"Tesco": swaggerPassword,
			},
		},
		FeatureFlagConfig: featureflag.Config{
			APPName:  "default",
			BaseURL:  "http://localhost:4242/api",
			APIToken: "",
		},
	}

	return
}

// Tunnel returns the tunnel config
type Tunnel struct {
	Server string
	Domain string
	Token  string
	// must be lowercase or numeric, like 25ac43e
	SubdomainSuffix string `comment:"must be lowercase or numeric, like 25ac43e"`
}

// Webhook define webhook settings
type Webhook struct {
	// public Internet address of webhook service like https://example.com, for webhook call address forging
	ExternalHost string `comment:"public Internet address of webhook service like https://example.com, for webhook call address forging"`
	// listening port
	Port int `comment:"listening port"`
	// timeout for webhook request handling, in seconds
	TimeoutSecondsPerRequest int `comment:"timeout for webhook request handling, in seconds"`

	// SlackWebhook config
	SlackWebhook SlackWebhookConfig `comment:"Slack webhook config"`
}

// APIServer defines api server settings
type APIServer struct {
	Port int
	// whether to print response body in logs, must be used with PayloadAuditLog middleware
	AuditResponse bool `comment:"whether to print response body in logs, must be used with PayloadAuditLog middleware"`
	// public Internet address of API server like https://example.com, for callback address forging
	ExternalHost string `comment:"public Internet address of API server like https://example.com"`
}

type SlackWebhookConfig struct {
	// confirm that each request comes from Slack by verifying its unique signature.
	SigningSecret string `comment:"confirm that each request comes from Slack by verifying its unique signature."`
}

// Load reads config from toml file
func Load(fileName string) (cfg Config, err error) {
	file, err := os.Open(fileName)
	if err != nil {
		err = fmt.Errorf("opening config file: %w", err)
		return
	}
	defer file.Close()

	decoder := toml.NewDecoder(file)
	err = decoder.Decode(&cfg)
	if err != nil {
		err = fmt.Errorf("decoding toml from file %s: %w", file.Name(), err)
		return
	}

	if err = cfg.Validate(); err != nil {
		err = fmt.Errorf("config invalid: %w", err)
		return
	}

	cfg.DevelopmentTunnel.SubdomainSuffix = strings.ToLower(cfg.DevelopmentTunnel.SubdomainSuffix)
	return
}

func (c *Config) Validate() (err error) {
	adapterManager := adapter.GetAdapterManager()
	existsMap := map[string]struct{}{}
	for _, credential := range c.Credentials {
		if credential.Name == "" {
			return fmt.Errorf("credential name cannot be empty")
		}
		meta := adapterManager.LookupAdapter(credential.Adapter)
		if meta == nil {
			return fmt.Errorf("unknown adapter %s in official oauth2 config", credential.Adapter)
		}
		if !credential.Type.IsValid() {
			return errors.New("invalid credential type")
		}
		if _, ok := existsMap[credential.Name]; ok {
			return fmt.Errorf("credential %q already exists", credential.Name)
		}
		existsMap[credential.Name] = struct{}{}
	}

	return
}

// UpdateExternalHost updates all external host by using the tunnel server address.
func (c *Config) UpdateExternalHost(apiServerSubdomain, webhookSubdomain string) error {
	if c.DevelopmentTunnel.Server == "" {
		return fmt.Errorf("set tunnel server in config file")
	}
	if c.DevelopmentTunnel.Domain == "" {
		return fmt.Errorf("set tunnel domain in config file")
	}

	if c.DevelopmentTunnel.SubdomainSuffix == "" {
		return fmt.Errorf("set tunnel subdomainsuffix in config file")
	}

	c.APIServer.ExternalHost = fmt.Sprintf("https://%s.%s", apiServerSubdomain, c.DevelopmentTunnel.Domain)
	c.Webhook.ExternalHost = fmt.Sprintf("https://%s.%s", webhookSubdomain, c.DevelopmentTunnel.Domain)
	return nil
}

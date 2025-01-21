package constant

const (
	// EnvCADVISORS is the env key of cAdvisor addresses
	// Must set when metrics daemon config Updater is not enabled
	EnvCADVISORS = "CADVISORS"
	// EnvPollingSec is the env key of metrics fetching period
	// If not set, 5 by default, in second
	EnvPollingSec = "POLLING_SEC"
	// EnvCadvisorTimeout is the env key of timeout GET cAdvisor/metrics
	// If not set, 5000 by default, in millisecond
	EnvCadvisorTimeout = "CADVISOR_TIMEOUT"
	// EnvDaemonMode is the env key of metrics retrieving mode,
	// the value is supposed to be "onrequest" (default value if not set) or "polling"
	EnvDaemonMode = "DAEMON_MODE"
	// EnvEnableUpdater is the env key of whether enable configuration updater
	EnvEnableUpdater = "ENABLE_UPDATER"
	// EnvAddrUpdateSec is the env key of interval daemon refresh cadvisor addresses
	// If not set, 300 by default, in second
	EnvAddrUpdateSec = "ADDR_UPDATE_SEC"
	// EnvMesosEndpoint is the env key of mesos leader IP (or domain name) and port
	// If not set, "mesos.master/mesos" by default
	EnvMesosEndpoint = "MESOS_ENDPOINT"
	// EnvCadvisorPort is the env key of cadvisor port
	// If not set, 10000 by default
	EnvCadvisorPort = "CADVISOR_PORT"
	// EnvEnableHostMonitor is the env key of whether enable host monitor
	EnvEnableHostMonitor = "ENABLE_HOST_MONITOR"
)

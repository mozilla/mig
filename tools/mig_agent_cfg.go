package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"text/template"
)

const configTemplate string = `[agent]
    ; connection string to the MIG relay. must contain credentials.
    relay            = "{{.RelayAddress}}"

    ; location of the mig api
    api             = "{{.APIURL}}"

    proxies         = "{{.Proxies}}"

    ; location of the local stat socket
    socket           = "127.0.0.1:{{.StatSocketPort}}"

    ; frequency at which heartbeat messages are sent to the MIG relay
    heartbeatfreq    = "120s"

    ; timeout after which a module that has not finished is killed by the agent
    moduletimeout    = "300s"

    ; in immortal mode, the agent that encounter a fatal error
    ; will attempt to restart itself instead of just shutting down
    isimmortal       = {{.Immortal}}

    ; installservice orders the agent to deploy a service init configuration
    ; and start itself during the endpoint's boot process
    installservice   = {{.InstallAsService}}

    ; attempt to retrieve the public IP behind which the agent is running
    discoverpublicip = on

    ; in check-in mode, the agent connects to the relay, runs all pending commands
    ; and exits. this mode is used to run the agent as a cron job, not a daemon.
    checkin = {{.CronMode}}

    refreshenv = "5m"

    ; enable privacy mode
    extraprivacymode = {{.ExtraPrivacy}}

[stats]
    maxactions = 15

[certs]
    ca  = "/etc/mig/ca.crt"
    cert = "/etc/mig/agent.crt"
    key = "/etc/mig/agent.key"

[logging]
    mode    = "file" ; stdout | file | syslog
    level   = "debug"
    file    = "/var/log/mig-agent.log"
    maxfilesize = 10485760
`

// Defaults
const (
	defaultAPIURL           = "https://api.mig.mozilla.org/api/v1/"
	defaultStatSocketPort   = 51664
	defaultImmortal         = true
	defaultInstallAsService = true
	defaultCronMode         = false
	defaultExtraPrivacy     = true
)

type proxyList []string

// Config contains user-supplied configurable values that will be translated
// and injected into `release/mig-agent.cfg.template` to produce a valid
// `mig-agent.cfg`.
type Config struct {
	RelayAddress     string
	APIURL           string
	Proxies          proxyList
	StatSocketPort   uint
	Immortal         bool
	InstallAsService bool
	CronMode         bool
	ExtraPrivacy     bool
}

// translatedConfig contains the same data as `Config` with some
// transformations to make the output of this program consistent
// with what mig agents expect.
type translatedConfig struct {
	RelayAddress     string
	APIURL           string
	Proxies          string
	StatSocketPort   uint16
	Immortal         string
	InstallAsService string
	CronMode         string
	ExtraPrivacy     string
}

func onOrOff(on bool) string {
	if on {
		return "on"
	} else {
		return "off"
	}
}

func translate(config Config) translatedConfig {
	return translatedConfig{
		RelayAddress:     config.RelayAddress,
		APIURL:           config.APIURL,
		Proxies:          strings.Join([]string(config.Proxies), ","),
		StatSocketPort:   uint16(config.StatSocketPort),
		Immortal:         onOrOff(config.Immortal),
		InstallAsService: onOrOff(config.InstallAsService),
		CronMode:         onOrOff(config.CronMode),
		ExtraPrivacy:     onOrOff(config.ExtraPrivacy),
	}
}

// IsValid verifies that values that must be supplied for a configuration have
// been supplied and that those whose values cannot be guaranteed to be
// correct by the type system and flag API are valid.
func (config Config) IsValid() bool {
	return config.RelayAddress != "" && config.StatSocketPort <= 65535
}

// String is required by the `flag.Value` interface.
func (proxies *proxyList) String() string {
	return strings.Join([]string(*proxies), ",")
}

// Set is required by the `flag.Value` interface. We use it to enable users to
// supply multiple proxy addresses, which we collect into an array.
func (proxies *proxyList) Set(value string) error {
	*proxies = append(*proxies, value)
	return nil
}

func main() {
	relayAddr := flag.String(
		"relay",
		"",
		"Connection string to the MIG relay. Must contain credentials")
	apiUrl := flag.String(
		"api",
		defaultAPIURL,
		"URL of the MIG API")
	var proxies proxyList
	flag.Var(
		&proxies,
		"proxy",
		"Address of a proxy to use. Multiple -proxy arguments are accepted")
	statPort := flag.Uint(
		"statport",
		defaultStatSocketPort,
		"Port to bind the agent's stat socket to on localhost")
	immortal := flag.Bool(
		"immortal",
		defaultImmortal,
		"Instruct the agent to automatically recover from fatal errors")
	service := flag.Bool(
		"service",
		defaultInstallAsService,
		"Instruct the agent to install itself as a service")
	cronMode := flag.Bool(
		"cron",
		defaultCronMode,
		"Instruct the agent to run all queued actions before terminating instead of running as a daemon")
	extraPrivacy := flag.Bool(
		"extraprivate",
		defaultExtraPrivacy,
		"Instruct the agent to run with extra privacy controls")
	flag.Parse()

	template := template.Must(template.New("mig-agent.cfg").Parse(configTemplate))

	config := Config{
		RelayAddress:     *relayAddr,
		APIURL:           *apiUrl,
		Proxies:          proxies,
		StatSocketPort:   *statPort,
		Immortal:         *immortal,
		InstallAsService: *service,
		CronMode:         *cronMode,
		ExtraPrivacy:     *extraPrivacy,
	}
	if !config.IsValid() {
		fmt.Fprintf(os.Stderr, "Invalid configuration data supplied. Check that the -relay argument is present and that the -statport argument is a valid port number.\n")
		os.Exit(1)
	}

	translated := translate(config)
	err := template.Execute(os.Stdout, translated)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error filling configuration template: %s\n", err.Error())
		os.Exit(1)
	}
}

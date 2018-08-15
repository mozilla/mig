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

    ; Tags help investigators to target specific agents for queries.
    ; Multiple tags can be supplied as demonstrated below and the
    ; tag name and value are separated by a colon.
    ; tags = "operator:example"
    ; tags = "exampleTag:other"{{ range $tag, $value := .Tags }}
    tags = "{{ $tag }}:{{ $value }}"{{ end }}

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
type tagList []string

// Config contains user-supplied configurable values that will be translated
// and injected into `release/mig-agent.cfg.template` to produce a valid
// `mig-agent.cfg`.
type Config struct {
	RelayAddress     string
	APIURL           string
	Proxies          proxyList
  Tags             tagList
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
  Tags             map[string]string
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

// kvPairsToMap converts strings formatted like `key=value` to a map.
func kvPairsToMap(kvs []string) (map[string]string, error) {
  mapping := map[string]string{}
  for _, pair := range kvs {
    parts := strings.Split(pair, "=")
    if len(parts) != 2 {
      return map[string]string{}, fmt.Errorf("encountered invalidly formatted tag \"%s\". Expected format key=value")
    }
    mapping[parts[0]] = parts[1]
  }
  return mapping, nil
}

func translate(config Config) (translatedConfig, error) {
  tags, err := kvPairsToMap(config.Tags)
  if err != nil {
    return translatedConfig{}, err
  }

  tx := translatedConfig{
		RelayAddress:     config.RelayAddress,
		APIURL:           config.APIURL,
		Proxies:          strings.Join([]string(config.Proxies), ","),
    Tags:             tags,
		StatSocketPort:   uint16(config.StatSocketPort),
		Immortal:         onOrOff(config.Immortal),
		InstallAsService: onOrOff(config.InstallAsService),
		CronMode:         onOrOff(config.CronMode),
		ExtraPrivacy:     onOrOff(config.ExtraPrivacy),
	}

  return tx, nil
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

// String is required by the `flag.Value` interface.
func (tags *tagList) String() string {
  return strings.Join([]string(*tags), ",")
}

// Set is required by the `flag.Value` interface. We use it to enable users to
// supply multiple tag key=value pairs, which we collect into an array.
func (tags *tagList) Set(kvPair string) error {
  if !strings.Contains(kvPair, "=") {
    return fmt.Errorf("expected tag argument to be formatted key=value")
  }
  *tags = append(*tags, kvPair)
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
  var tags tagList
  flag.Var(
    &tags,
    "tag",
    "A tagName=tagValue pair to tag the agent with. Multuple -tag arguments are accepted")
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
    Tags:             tags,
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

	translated, err := translate(config)
  if err != nil {
    fmt.Fprintf(os.Stderr, "Error creating configuration: %s\n", err.Error())
    os.Exit(1)
  }

	err = template.Execute(os.Stdout, translated)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error filling configuration template: %s\n", err.Error())
		os.Exit(1)
	}
}

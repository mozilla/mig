package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"strings"
	"text/template"
)

const (
	configTemplateDir  = "release"
	configTemplateFile = "mig-agent.cfg.template"
)

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
	defaultTemplateFilePath := path.Join(configTemplateDir, configTemplateFile)

	templateFilePath := flag.String(
		"template",
		defaultTemplateFilePath,
		"Path to the mig-agent.cfg template to fill")
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

	fmt.Fprintf(os.Stderr, "Trying to load template %s\n", *templateFilePath)
	template, err := template.ParseFiles(*templateFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing template configuration: %s\n", err.Error())
		os.Exit(1)
	}

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
	err = template.Execute(os.Stdout, translated)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error filling configuration template: %s\n", err.Error())
		os.Exit(1)
	}
}

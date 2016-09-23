package service

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log/syslog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/kardianos/osext"
)

const (
	initSystemV = initFlavor(iota)
	initUpstart
	initSystemd
)

// the default flavor is initSystemV. we lookup the command line of
// process 1 to detect systemd or upstart
func getFlavor() (initFlavor, error) {
	initCmd, err := ioutil.ReadFile("/proc/1/cmdline")
	if err != nil {
		return initSystemV, err
	}
	// Trim any nul bytes from the result, which are present with some
	// kernels but not others
	init := string(bytes.TrimRight(initCmd, "\x00"))
	if strings.Contains(init, "init [") {
		return initSystemV, nil
	}
	if strings.Contains(init, "systemd") {
		return initSystemd, nil
	}
	if strings.Contains(init, "init") {
		// not so fast! you may think this is upstart, but it may be
		// a symlink to systemd... yeah, debian does that... ( x )
		var target string
		if len(init) > 9 && init[0:10] == "/sbin/init" {
			target, err = filepath.EvalSymlinks("/sbin/init")
		} else {
			target, err = filepath.EvalSymlinks(init)
		}
		if err == nil && strings.Contains(target, "systemd") {
			return initSystemd, nil
		}
		return initUpstart, nil
	}
	// failed to detect init system, falling back to sysvinit
	return initSystemV, nil
}

func newService(c *Config) (Service, error) {
	var err error
	flavor, err := getFlavor()
	if err != nil {
		return nil, err
	}
	s := &linuxService{
		flavor:      flavor,
		name:        c.Name,
		displayName: c.DisplayName,
		description: c.Description,
	}
	s.logger, err = syslog.New(syslog.LOG_INFO, s.name)
	if err != nil {
		return nil, err
	}
	return s, nil
}

type linuxService struct {
	flavor                         initFlavor
	name, displayName, description string
	logger                         *syslog.Writer
}

func (ls *linuxService) String() string {
	return fmt.Sprintf("Linux %s", ls.flavor.String())
}

type initFlavor uint8

func (f initFlavor) String() string {
	switch f {
	case initSystemV:
		return "sysvinit"
	case initUpstart:
		return "upstart"
	case initSystemd:
		return "systemd"
	default:
		return "unknown"
	}
}

func (f initFlavor) ConfigPath(name string) string {
	switch f {
	case initSystemd:
		return "/etc/systemd/system/" + name + ".service"
	case initSystemV:
		return "/etc/init.d/" + name
	case initUpstart:
		return "/etc/init/" + name + ".conf"
	default:
		return ""
	}
}

func (f initFlavor) GetTemplate() *template.Template {
	var templ string
	switch f {
	case initSystemd:
		templ = systemdScript
	case initSystemV:
		templ = systemVScript
	case initUpstart:
		templ = upstartScript
	}
	return template.Must(template.New(f.String() + "Script").Parse(templ))
}

func (s *linuxService) Install() error {
	confPath := s.flavor.ConfigPath(s.name)
	_, err := os.Stat(confPath)
	if err == nil {
		return fmt.Errorf("Init already exists: %s", confPath)
	}

	f, err := os.Create(confPath)
	if err != nil {
		return err
	}
	defer f.Close()

	path, err := osext.Executable()
	if err != nil {
		return err
	}

	var to = &struct {
		Display     string
		Description string
		Path        string
	}{
		s.displayName,
		s.description,
		path,
	}

	err = s.flavor.GetTemplate().Execute(f, to)
	if err != nil {
		return err
	}

	if s.flavor == initSystemV {
		if err = os.Chmod(confPath, 0755); err != nil {
			return err
		}
		for _, i := range [...]string{"2", "3", "4", "5"} {
			if err = os.Symlink(confPath, "/etc/rc"+i+".d/S50"+s.name); err != nil {
				continue
			}
		}
		for _, i := range [...]string{"0", "1", "6"} {
			if err = os.Symlink(confPath, "/etc/rc"+i+".d/K02"+s.name); err != nil {
				continue
			}
		}
	}

	if s.flavor == initSystemd {
		err = exec.Command("systemctl", "enable", s.name+".service").Run()
		if err != nil {
			return err
		}
		return exec.Command("systemctl", "daemon-reload").Run()
	}

	return nil
}

func (s *linuxService) Remove() error {
	if s.flavor == initSystemd {
		exec.Command("systemctl", "disable", s.name+".service").Run()
	}
	if err := os.Remove(s.flavor.ConfigPath(s.name)); err != nil {
		return err
	}
	return nil
}

func (s *linuxService) Run(onStart, onStop func() error) (err error) {
	err = onStart()
	if err != nil {
		return err
	}
	defer func() {
		err = onStop()
	}()

	sigChan := make(chan os.Signal, 3)

	signal.Notify(sigChan, os.Interrupt, os.Kill)

	<-sigChan

	return nil
}

func (s *linuxService) Start() error {
	switch s.flavor {
	case initSystemd:
		return exec.Command("systemctl", "start", s.name+".service").Run()
	case initUpstart:
		return exec.Command("initctl", "start", s.name).Run()
	default:
		return exec.Command("service", s.name, "start").Run()
	}
}

func (s *linuxService) Stop() error {
	switch s.flavor {
	case initSystemd:
		return exec.Command("systemctl", "stop", s.name+".service").Start()
	case initUpstart:
		return exec.Command("initctl", "stop", s.name).Start()
	default:
		return exec.Command("service", s.name, "stop").Start()
	}
}

func (s *linuxService) IntervalMode(interval int) error {
	return fmt.Errorf("interval mode service only supported on darwin")
}

func (s *linuxService) Error(format string, a ...interface{}) error {
	return s.logger.Err(fmt.Sprintf(format, a...))
}
func (s *linuxService) Warning(format string, a ...interface{}) error {
	return s.logger.Warning(fmt.Sprintf(format, a...))
}
func (s *linuxService) Info(format string, a ...interface{}) error {
	return s.logger.Info(fmt.Sprintf(format, a...))
}

const systemVScript = `#!/bin/sh
# For RedHat and cousins:
# chkconfig: - 99 01
# description: {{.Description}}
# processname: {{.Path}}

### BEGIN INIT INFO
# Provides:          {{.Path}}
# Required-Start:
# Required-Stop:
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: {{.Display}}
# Description:       {{.Description}}
### END INIT INFO

cmd="{{.Path}}"

name=$(basename $0)
pid_file="/var/run/$name.pid"
stdout_log="/var/log/$name.log"
stderr_log="/var/log/$name.err"

get_pid() {
    cat "$pid_file"
}

is_running() {
    [ -f "$pid_file" ] && ps $(get_pid) > /dev/null 2>&1
}

case "$1" in
    start)
        if is_running; then
            echo "Already started"
        else
            echo "Starting $name"
            $cmd >> "$stdout_log" 2>> "$stderr_log" &
            echo $! > "$pid_file"
            if ! is_running; then
                echo "Unable to start, see $stdout_log and $stderr_log"
                exit 1
            fi
        fi
    ;;
    stop)
        if is_running; then
            echo -n "Stopping $name.."
            kill $(get_pid)
            for i in {1..10}
            do
                if ! is_running; then
                    break
                fi
                echo -n "."
                sleep 1
            done
            echo
            if is_running; then
                echo "Not stopped; may still be shutting down or shutdown may have failed"
                exit 1
            else
                echo "Stopped"
                if [ -f "$pid_file" ]; then
                    rm "$pid_file"
                fi
            fi
        else
            echo "Not running"
        fi
    ;;
    restart)
        $0 stop
        if is_running; then
            echo "Unable to stop, will not attempt to start"
            exit 1
        fi
        $0 start
    ;;
    status)
        if is_running; then
            echo "Running"
        else
            echo "Stopped"
            exit 1
        fi
    ;;
    *)
    echo "Usage: $0 {start|stop|restart|status}"
    exit 1
    ;;
esac
exit 0`

const upstartScript = `# {{.Description}}

description     "{{.Display}}"

start on filesystem or runlevel [2345]
stop on runlevel [!2345]

#setuid username

# stop the respawn is process fails to start 5 times within 5 minutes
respawn
respawn limit 5 300
umask 022

console none

pre-start script
    test -x {{.Path}} || { stop; exit 0; }
end script

# Start
exec {{.Path}}
`

const systemdScript = `[Unit]
Description={{.Description}}
ConditionFileIsExecutable={{.Path}}
After=network.target

[Service]
ExecStart={{.Path}}
# respawn process on crash after a 3s wait
# if fails to start 5 times within 5 minutes, stop trying
Restart=on-failure
RestartSec=3s
StartLimitInterval=300
StartLimitBurst=5

[Install]
WantedBy=multi-user.target
`

; Sample MIG Agent configuration file

[agent]
    ; connection string to the MIG relay. must contain credentials.
    relay            = "amqp://guest:guest@localhost:5672/"

    ; location of the mig api
    api             = "http://localhost:1664/api/v1/"

    ; location of the local stat socket
    socket           = "127.0.0.1:51664"

    ; frequency at which heartbeat messages are sent to the MIG relay
    heartbeatfreq    = "300s"

    ; timeout after which a module that has not finished is killed by the agent
    moduletimeout    = "300s"

    ; in immortal mode, the agent that encounter a fatal error
    ; will attempt to restart itself instead of just shutting down
    isimmortal       = on

    ; comma delimited list of host:port proxies to use, if desired
    ; the agent will attempt to try to proxies for public ip retrieval
    ; and the relay connection before a direct connection
    ; proxies          = "proxy1:8888,proxy2:8888"

    ; installservice orders the agent to deploy a service init configuration
    ; and start itself during the endpoint's boot process
    installservice   = on

    ; attempt to retrieve the public IP behind which the agent is running
    discoverpublicip = off

    ; in check-in mode, the agent connects to the relay, runs all pending commands
    ; and exits. this mode is used to run the agent as a cron job, not a daemon.
    checkin = off

    ; how often should the agent refresh its own environment (for example, to update
    ; the environment if the system changes ip addresses. leave unset or "" to disable
    ; periodic environment refresh. on workstations or other devices that may be
    ; obtaining dhcp leases and moving around, you might want to start with something
    ; like "5m"
    refreshenv = ""

[certs]
    ca  = "/path/to/ca/cert"
    cert= "/path/to/client/cert"
    key = "/path/to/private/key"

[logging]
    mode    = "stdout" ; stdout | file | syslog
    level   = "debug"

; for file logging
;   file = "mig_scheduler.log"
;   maxfilesize = 0 ; if > 0, log file will be rotated once it reaches size

; for syslog, logs go into local3
;    host   = "localhost"
;    port   = 514
;    protocol = "udp"

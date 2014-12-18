; Sample MIG Agent configuration file

[agent]
    ; connection string to the MIG relay. must contain credentials.
    relay            = "amqp://guest:guest@localhost:5672/"

    ; location of the local stat socket
    socket           = "127.0.0.1:51664"

    ; frequency at which heartbeat messages are sent to the MIG relay
    heartbeatfreq    = "300s"

    ; timeout after which a module that has not finished is killed by the agent
    moduletimeout    = "300s"

    ; in immortal mode, the agent that encounter a fatal error
    ; will attempt to restart itself instead of just shutting down
    isimmortal       = on

    ; installservice orders the agent to deploy a service init configuration
    ; and start itself during the endpoint's boot process
    installservice   = on

    ; attempt to retrieve the public IP behind which the agent is running
    discoverpublicip = off

[certs]
    ca  = "/path/to/ca/cert"
    cert= "/path/to/client/cert"
    key = "/path/to/private/key"

[logging]
    mode    = "stdout" ; stdout | file | syslog
    level   = "debug"

; for file logging
;   file = "mig_scheduler.log"

; for syslog, logs go into local3
;    host   = "localhost"
;    port   = 514
;    protocol = "udp"

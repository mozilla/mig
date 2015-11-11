; Sample MIG Agent configuration file

[agent]
    ; connection string to the MIG relay. must contain credentials.
    relay            = "amqp://guest:guest@localhost:5672/"

    ; location of the mig api
    api             = "http://localhost:1664/api/v1/"

    ; a list of proxies to try if direct connection doesn't work
    proxy           = "proxy.example.net:3128"
    proxy           = "proxy2.example.net:8080"

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

    ; in check-in mode, the agent connects to the relay, runs all pending commands
    ; and exits. this mode is used to run the agent as a cron job, not a daemon.
    checkin          = off

[acl]
    ; Master Keys are used to verify ACL Signatures
    key = "E60892BB9BD89A69F759A1A0A3D652173B763E8F"
    key = "FA5D79F95F7AF7097C3E83DA26A86D5E5885AC11"

    ; Signatures Minimum indicates how many signatures from ACL Keys
    ; are required to verify and load an ACL
    signatureminimum = 1

    ; ACL indicates which Access Control Lists the agent should load.
    ; For each group, an ACL will be requested from the API, verified
    ; and loaded into the agent.
    acl = default
    acl = mygroup1

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

; Sample MIG configuration file

[agent]
    timeout = "20m"
    heartbeatfreq = "5m"
    whitelist = "/var/cache/mig/agents_whitelist.txt"
    detectmultiagents = true
    killdupagents = false

[collector]
; frequency at which the job collector runs
    freq = "60s"
; period during which done actions and commands, and invalid actions are kept
    deleteafter = "72h"

[directories]
    spool = "/var/cache/mig/"
    tmp = "/var/tmp/"

[postgres]
    host = "127.0.0.1"
    port = 5432
    dbname = "mig"
    user = "migscheduler"
    password = "123456"
    sslmode = "disable"
    maxconn = 10

[mq]
    host  = "127.0.0.1"
    port  = 5672
    user  = "guest"
    pass  = "guest"
    vhost = "/"

; TLS options
;   usetls  = true
;   cacert  = "/path/to/cacert.pem"
;   tlscert = "/path/to/cert.pem"
;   tlskey  = "/path/to/key.pem"

; AMQP options
; timeout defaults to 10 minutes
; keep this higher than the agent heartbeat value
;   timeout = "10m"

[logging]
    mode = "stdout" ; stdout | file | syslog
    level = "debug"

; for file logging
;   file = "mig_scheduler.log"

; for syslog, logs go into local3
;    host = "localhost"
;    port = 514
;    protocol = "udp"

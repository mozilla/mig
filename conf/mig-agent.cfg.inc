; Sample MIG Agent configuration file

[agent]
    isimmortal      = on
    installservice  = on
    relay           = "amqp://guest:guest@localhost:5672/"
    socket          = "127.0.0.1:51664"
    heartbeatfreq   = "300s"
    moduletimeout   = "300s"

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

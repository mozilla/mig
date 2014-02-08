; Sample MIG configuration file

[agent]
    timeout = "20m"
    whitelist = "/var/cache/mig/agents_whitelist.txt"

[directories]
    spool = "/var/cache/mig/"
    tmp = "/var/tmp/"

[db]
    host = "127.0.0.1"
    port = 27017

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

[logging]
    mode = "stdout" ; stdout | file | syslog
    level = "debug"

; for file logging
;   file = "mig_scheduler.log"

; for syslog, logs go into local3
;    host = "localhost"
;    port = 514
;    protocol = "udp"

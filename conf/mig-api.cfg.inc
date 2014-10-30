; Sample MIG API configuration file

[server]
    ip = "127.0.0.1"
    port = 12345
    host = "http://localhost:12345"
    baseroute = "/api/v1"

[directories]
    spool = "/var/cache/mig/"
    tmp = "/var/tmp/"

[postgres]
    host = "127.0.0.1"
    port = 5432
    dbname = "mig"
    user = "migapi"
    password = "123456"
    sslmode = "disable"

[logging]
    mode = "stdout" ; stdout | file | syslog
    level = "debug"

; for file logging
;   file = "mig_scheduler.log"

; for syslog, logs go into local3
;    host = "localhost"
;    port = 514
;    protocol = "udp"

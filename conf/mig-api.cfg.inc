; Sample MIG API configuration file

[server]
    listening_ip = "127.0.0.1"
    listening_port = 12345

[openpgp]
    pubring = "/var/cache/mig/.gnupg/pubring.gpg"

[directories]
    spool = "/var/cache/mig/"
    tmp = "/var/tmp/"

[db]
    host = "127.0.0.1"
    port = 27017

[logging]
    mode = "stdout" ; stdout | file | syslog
    level = "debug"

; for file logging
;   file = "mig_scheduler.log"

; for syslog, logs go into local3
;    host = "localhost"
;    port = 514
;    protocol = "udp"

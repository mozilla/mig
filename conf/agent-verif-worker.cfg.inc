[mq]
    host = "localhost"
    port = 5672
    user = "worker"
    pass = "secretpassphrase"
    vhost = "mig"
    usetls  = false
    cacert  = "/etc/mig/certs/ca.crt"
    tlscert = "/etc/mig/certs/worker.crt"
    tlskey  = "/etc/mig/certs/worker.key"
    timeout = "10s"
[logging]
    mode = "stdout" ; stdout | file | syslog
    level = "debug"
    ;host = "localhost"
    ;port = 514
    ;protocol = "udp"

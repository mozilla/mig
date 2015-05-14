[mozdef]
    host = "mozdef-relay.example.net"
    port = 5671
    user = "mig-agent-intel-worker"
    pass = "secretpassphrase"
    vhost = "prod"
    exchange = "eventtask"
    routingkey = "eventtask"
    usetls  = true
    cacertpath  = "/etc/mig/certs/ca.crt"
    clientcertpath = "/etc/mig/certs/worker.crt"
    clientkeypath  = "/etc/mig/certs/worker.key"
    timeout = "10s"
[mq]
    host = "somerelay.somedomain.example.net"
    port = 443
    user = "worker"
    pass = "secretpassphrase2"
    vhost = "mig"
    usetls  = true
    cacert  = "/etc/mig/certs/ca.crt"
    tlscert = "/etc/mig/certs/worker.crt"
    tlskey  = "/etc/mig/certs/worker.key"
    timeout = "10s"
[vmintgr]
    bin = "/opt/vmintgr/bin/grouptest"
[logging]
    mode = "syslog" ; stdout | file | syslog
    level = "info"
    host = "localhost"
    port = 514
    protocol = "udp"

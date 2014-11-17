#!/usr/bin/env bash
echo Standalone MIG demo deployment script
which sudo 2>&1 1>/dev/null || (echo Install sudo and try again && exit 1)

echo -e "\n---- Shutting down existing Scheduler and API tmux sessions\n"
sudo tmux -S /tmp/tmux-$(id -u mig)/default kill-session -t mig

echo -e "\n---- Destroying existing investigator conf & key\n"
rm -rf -- ~/.migrc ~/.mig
sudo killall /sbin/mig-agent

# packages dependencies
pkglist=""
distrib=$(head -1 /etc/issue|awk '{print $1}')
case $distrib in
    Amazon|Fedora|Red|CentOS|Scientific)
        PKG="yum"
    [ ! -r "/usr/include/readline/readline.h" ] && pkglist="$pkglist readline-devel"
    [ ! -d "/var/lib/rabbitmq" ] && echo RabbitMQ is not installed and your distribution does not ship it. Install rabbitmq-server manually and rerun this script. http://www.rabbitmq.com/download.html
    [ ! -r "/usr/bin/postgres" ] && pkglist="$pkglist postgresql"
    ;;
    Debian|Ubuntu)
        PKG="apt-get"
    [ ! -e "/usr/include/readline/readline.h" ] && pkglist="$pkglist libreadline-dev"
    [ ! -d "/var/lib/rabbitmq" ] && pkglist="$pkglist rabbitmq-server"
    ls /usr/lib/postgresql/*/bin/postgres 2>&1 1>/dev/null || pkglist="$pkglist postgresql"
    ;;
esac

which go   2>&1 1>/dev/null || pkglist="$pkglist golang"
which git  2>&1 1>/dev/null || pkglist="$pkglist git"
which hg   2>&1 1>/dev/null || pkglist="$pkglist mercurial"
which make 2>&1 1>/dev/null || pkglist="$pkglist make"
which gcc  2>&1 1>/dev/null || pkglist="$pkglist gcc"
which tmux 2>&1 1>/dev/null || pkglist="$pkglist tmux"
which curl 2>&1 1>/dev/null || pkglist="$pkglist tmux"

if [ "$pkglist" != "" ]; then
    echo "missing packages: $pkglist"
    echo -n "would you list to install the missing packages? (need sudo) y/n> "
    read yesno
    if [ $yesno = "y" ]; then
        sudo $PKG install $pkglist || ( echo FAILED && exit 1 )
    fi
fi

echo -e "\n---- Checking out MIG source code\n"
if [ -d mig ]; then
    cd mig
    git pull origin master || ( echo FAILED && exit 1 )
else
    git clone https://github.com/mozilla/mig.git || ( echo FAILED && exit 1 )
    cd mig
fi

echo -e "\n---- Retrieving build dependencies\n"
make go_get_deps || ( echo FAILED && exit 1 )

echo -e "\n---- Building MIG Scheduler\n"
make mig-scheduler || ( echo FAILED && exit 1 )
sudo cp bin/linux/amd64/mig-scheduler /usr/local/bin/ || ( echo FAILED && exit 1 )
sudo chown mig /usr/local/bin/mig-scheduler || ( echo FAILED && exit 1 )
sudo chmod 550 /usr/local/bin/mig-scheduler || ( echo FAILED && exit 1 )

echo -e "\n---- Building MIG API\n"
make mig-api || ( echo FAILED && exit 1 )
sudo cp bin/linux/amd64/mig-api /usr/local/bin/ || ( echo FAILED && exit 1 )
sudo chown mig /usr/local/bin/mig-api || ( echo FAILED && exit 1 )
sudo chmod 550 /usr/local/bin/mig-api || ( echo FAILED && exit 1 )

echo -e "\n---- Building MIG Console\n"
make mig-console || ( echo FAILED && exit 1 )
sudo cp bin/linux/amd64/mig-console /usr/local/bin/ || ( echo FAILED && exit 1 )
sudo chown mig /usr/local/bin/mig-console || ( echo FAILED && exit 1 )
sudo chmod 555 /usr/local/bin/mig-console || ( echo FAILED && exit 1 )

echo -e "\n---- Building Database\n"
cd doc/.files
dbpass=$(cat /dev/urandom | tr -dc _A-Z-a-z-0-9 | head -c${1:-32})
sudo su - postgres -c "psql -c 'drop database mig'"
sudo su - postgres -c "psql -c 'drop role migadmin'"
sudo su - postgres -c "psql -c 'drop role migapi'"
sudo su - postgres -c "psql -c 'drop role migscheduler'"
sudo su - postgres -c "psql -c 'drop role migreadonly'"
bash createdb.sh $dbpass || ( echo FAILED && exit 1 )

echo -e "\n---- Creating system user and folders\n"
id mig || sudo useradd -r mig || ( echo FAILED && exit 1 )
sudo mkdir -p /var/cache/mig/{action/new,action/done,action/inflight,action/invalid,command/done,command/inflight,command/ready,command/returned} || ( echo FAILED && exit 1 )
sudo hostname > /tmp/agents_whitelist.txt || ( echo FAILED && exit 1 )
sudo echo localhost >> /tmp/agents_whitelist.txt || ( echo FAILED && exit 1 )
sudo mv /tmp/agents_whitelist.txt /var/cache/mig/
sudo chown mig /var/cache/mig -R || ( echo FAILED && exit 1 )
sudo mkdir /etc/mig || ( echo FAILED && exit 1 )
sudo chown mig /etc/mig || ( echo FAILED && exit 1 )

echo -e "\n---- Configuring RabbitMQ\n"
mqpass=$(cat /dev/urandom | tr -dc _A-Z-a-z-0-9 | head -c${1:-32})
sudo rabbitmqctl delete_user admin
sudo rabbitmqctl add_user admin $mqpass || ( echo FAILED && exit 1 )
sudo rabbitmqctl set_user_tags admin administrator || ( echo FAILED && exit 1 )
sudo rabbitmqctl delete_user scheduler
sudo rabbitmqctl add_user scheduler $mqpass || ( echo FAILED && exit 1 )
sudo rabbitmqctl delete_user agent
sudo rabbitmqctl add_user agent $mqpass || ( echo FAILED && exit 1 )
sudo rabbitmqctl add_vhost mig
sudo rabbitmqctl list_vhosts
sudo rabbitmqctl set_permissions -p mig scheduler \
    '^mig(|\.(heartbeat|sched\..*))' \
    '^mig.*' \
    '^mig(|\.(heartbeat|sched\..*))' || ( echo FAILED && exit 1 )
sudo rabbitmqctl set_permissions -p mig agent \
    "^mig\.agt\.*" \
    "^mig*" \
    "^mig(|\.agt\..*)" || ( echo FAILED && exit 1 )

echo -e "\n---- Creating Scheduler configuration\n"
cat > /tmp/mig-scheduler.cfg << EOF
[agent]
    timeout = "12h"
    heartbeatfreq = "30s"
    whitelist = "/var/cache/mig/agents_whitelist.txt"
    detectmultiagents = true
[collector]
    freq = "7s"
    deleteafter = "168h"
[directories]
    spool = "/var/cache/mig/"
    tmp = "/var/tmp/"
[postgres]
    host = "127.0.0.1"
    port = 5432
    dbname = "mig"
    user = "migscheduler"
    password = "$dbpass"
    sslmode = "disable"
    maxconn = 10
[mq]
    host = "127.0.0.1"
    port = 5672
    user = "scheduler"
    pass = "$mqpass"
    vhost = "mig"
; TLS options
    usetls  = false
; AMQP options
    timeout = "600s"
[logging]
    mode = "file"
    level = "info"
    file = "/var/cache/mig/mig-scheduler.log"
EOF
sudo mv /tmp/mig-scheduler.cfg /etc/mig/ || ( echo FAILED && exit 1 )
sudo chown mig /etc/mig/mig-scheduler.cfg || ( echo FAILED && exit 1 )
sudo chmod 750 /etc/mig/mig-scheduler.cfg || ( echo FAILED && exit 1 )

echo -e "\n---- Creating API configuration\n"
cat > /tmp/mig-api.cfg << EOF
[authentication]
    enabled = off
    tokenduration = 10m
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
    password = "$dbpass"
    sslmode = "disable"
[logging]
    mode = "file" ; stdout | file | syslog
    level = "info"
    file = "/var/cache/mig/mig-api.log"
EOF
sudo mv /tmp/mig-api.cfg /etc/mig/ || ( echo FAILED && exit 1 )
sudo chown mig /etc/mig/mig-api.cfg || ( echo FAILED && exit 1 )
sudo chmod 750 /etc/mig/mig-api.cfg || ( echo FAILED && exit 1 )

echo -e "\n---- Starting Scheduler and API in TMUX under mig user\n"
sudo su mig -c "/usr/bin/tmux new-session -s 'mig' -d"
sudo su mig -c "/usr/bin/tmux new-window -t 'mig' -n '0' '/usr/local/bin/mig-scheduler -c /etc/mig/mig-scheduler.cfg'"
sudo su mig -c "/usr/bin/tmux new-window -t 'mig' -n '0' '/usr/local/bin/mig-api -c /etc/mig/mig-api.cfg'"

ret=$(curl -s http://localhost:12345/api/v1/heartbeat)
[ "$ret" != "gatorz say hi" ] && echo API Startup FAILED && exit 1

echo -e "\n---- Creating GnuPG key pair for new investigator in ~/.mig/\n"
[ ! -d ~/.mig ] && mkdir ~/.mig
gpg --batch --no-default-keyring --keyring ~/.mig/pubring.gpg --secret-keyring ~/.mig/secring.gpg --gen-key << EOF
Key-Type: 1
Key-Length: 1024
Subkey-Type: 1
Subkey-Length: 1024
Name-Real: $(whoami) Investigator
Name-Email: $(whoami)@$(hostname)
Expire-Date: 12m
EOF

echo -e "\n---- Creating client configuration in ~/.migrc\n"
keyid=$(gpg --no-default-keyring --keyring ~/.mig/pubring.gpg \
    --secret-keyring ~/.mig/secring.gpg --fingerprint \
    --with-colons $(whoami)@$(hostname) | grep '^fpr' | cut -f 10 -d ':')
cat > ~/.migrc << EOF
[api]
    url = "http://localhost:12345/api/v1/"
    skipverifycert = on
[gpg]
    home = "/home/$(whoami)/.mig/"
    keyid = "$keyid"
EOF

echo -e "\n---- Creating investigator $(whoami) in database\n"
gpg --no-default-keyring --keyring ~/.mig/pubring.gpg \
    --secret-keyring ~/.mig/secring.gpg \
    --export -a $(whoami)@$(hostname) \
    > ~/.mig/$(whoami)-pubkey.asc || ( echo FAILED && exit 1 )
echo -e "create investigator\n$(whoami)\n$HOME/.mig/$(whoami)-pubkey.asc\ny\n" | \
    /usr/local/bin/mig-console -q || ( echo FAILED && exit 1 )

echo -e "\n---- Creating agent configuration\n"
cd; cd mig
cat > conf/mig-agent-conf.go << EOF
package main
import(
    "mig"
    "time"
    _ "mig/modules/filechecker"
    _ "mig/modules/netstat"
    _ "mig/modules/upgrade"
    _ "mig/modules/agentdestroy"
    _ "mig/modules/example"
)
var TAGS = struct {
    Operator string \`json:"operator"\`
}{
    "MIGDemo",
}
var ISIMMORTAL bool = true
var MUSTINSTALLSERVICE bool = true
var DISCOVERPUBLICIP = false
var LOGGINGCONF = mig.Logging{
    Mode:   "file",
    Level:  "debug",
    File:   "//var/cache/mig/mig-agent.log",
}
var AMQPBROKER string = "amqp://agent:$mqpass@localhost:5672/mig"
var PROXIES = [...]string{``}
var SOCKET = "127.0.0.1:51664"
var HEARTBEATFREQ time.Duration = 300 * time.Second
var MODULETIMEOUT time.Duration = 300 * time.Second
var AGENTACL = [...]string{
\`{
    "default": {
        "minimumweight": 2,
        "investigators": {
            "$(whoami)": {
                "fingerprint": "$keyid",
                "weight": 2
            }
        }
    }
}\`}
var PUBLICPGPKEYS = [...]string{
\`
$(cat $HOME/.mig/$(whoami)-pubkey.asc)
\`}
var CACERT = []byte("")
var AGENTCERT = []byte("")
var AGENTKEY = []byte("")
EOF

echo -e "\n---- Building and running local agent\n"
make mig-agent || ( echo FAILED && exit 1 )
sudo cp bin/linux/amd64/mig-agent-latest /sbin/mig-agent || ( echo FAILED && exit 1 )
sudo chown root /sbin/mig-agent || ( echo FAILED && exit 1 )
sudo chmod 500 /sbin/mig-agent || ( echo FAILED && exit 1 )
sudo /sbin/mig-agent

sleep 3
echo exit | /usr/local/bin/mig-console

cat << EOF
MIG is up and running with one local agent. Try /usr/local/bin/mig-console

This configuration is insecure, do not use it in production yet.
To make it secure, do the following:

  1. Create a PKI, give a server cert to Rabbitmq, and client certs
     to the scheduler and the agents. See doc at
     http://mig.mozilla.org/doc/configuration.rst.html#rabbitmq-tls-configuration

  2. Create real investigators and disable investigator id 2 when done.

  3. Enable HTTPS and active authentication in the API. Do not open the API
     to the world in the current state!

  4. You may want to add TLS to Postgres and tell the scheduler and API to use it.

  5. Change database password of users 'migapi' and 'migscheduler'. Change rabbitmq
     passwords of users 'scheduler' and 'agent';
EOF

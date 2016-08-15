#!/usr/bin/env bash
fail() {
    echo configuration failed
    exit 1
}

if [ -z "$BASH_SOURCE" ]; then
    echo "This script *must* run under bash. Please rerun with '$ bash $0'"
    fail
fi

echo Standalone MIG demo deployment script
which sudo 2>&1 1>/dev/null || (echo Install sudo and try again && exit 1)

MAKEGOPATH=false
if [ "$1" == "makegopath" ]; then
    MAKEGOPATH=true
fi

echo -e "\n---- Checking build environment\n"
if [[ -z $GOPATH && $MAKEGOPATH == "true" ]]; then
    echo "GOPATH env variable is not set. setting it to '$HOME/go'"
    export GOPATH="$HOME/go"
    mkdir -p "$GOPATH/src/mig.ninja/"
    savepath=$(pwd)
    cd ..
    mv "$savepath" "$GOPATH/src/mig.ninja/"
    cd "$GOPATH/src/mig.ninja/"
fi
if [[ -z $GOPATH && $MAKEGOPATH == "false" ]]; then
    echo "GOPATH env variable is not set. either set it, or ask this script to create it using: $ $0 makegopath"
    fail
fi
if [[ "$GOPATH/src/mig.ninja/mig" != "$(pwd)" ]]; then
    echo "GOPATH error: This repository needs to be located inside of GOPATH for compilation to work."
    echo "current GOPATH is '$GOPATH'. current dir is '$(pwd)'."
    echo "You should run 'go get mig.ninja/mig' and work from '$GOPATH/src/mig.ninja/mig'."
    fail
fi

echo -e "\n---- Shutting down existing Scheduler and API tmux sessions\n"
sudo tmux -S /tmp/tmux-$(id -u mig)/default kill-session -t mig || echo "OK - No running MIG session found"

echo -e "\n---- Destroying existing investigator conf & key\n"
rm -rf -- ~/.migrc ~/.mig || echo "OK"
sudo /sbin/mig-agent -q=shutdown || echo "OK"

# packages dependencies
pkglist=""
installRabbitRPM=false
isRPM=false
distrib=$(head -1 /etc/issue|awk '{print $1}')
case $distrib in
    Amazon|Fedora|Red|CentOS|Scientific)
        isRPM=true
        PKG="yum"
        [ ! -r "/usr/include/readline/readline.h" ] && pkglist="$pkglist readline-devel"
        [ ! -d "/var/lib/rabbitmq" ] && pkglist="$pkglist erlang" && installRabbitRPM=true
        [ ! -r "/usr/bin/postgres" ] && pkglist="$pkglist postgresql-server"
    ;;
    Debian|Ubuntu)
        PKG="apt-get"
        [ ! -e "/usr/include/readline/readline.h" ] && pkglist="$pkglist libreadline-dev"
        [ ! -d "/var/lib/rabbitmq" ] && pkglist="$pkglist rabbitmq-server"
        ls /usr/lib/postgresql/*/bin/postgres 2>&1 1>/dev/null || pkglist="$pkglist postgresql"
    ;;
esac

echo -e "\n---- Checking the installed version of go\n"
# Make sure the correct version of go is installed. We need at least version
# 1.5.
if [ ! $(which go) ]; then
    echo "go doesn't seem to be installed, or is not in PATH; at least version 1.5 is required"
    exit 1
fi
go_version=$(go version)
echo $go_version | grep -E -q --regexp="go1\.[0-4]" && echo -e "installed version of go is ${go_version}\nwe need at least version 1.5" && fail

which go   2>&1 1>/dev/null || pkglist="$pkglist golang"
which git  2>&1 1>/dev/null || pkglist="$pkglist git"
which hg   2>&1 1>/dev/null || pkglist="$pkglist mercurial"
which make 2>&1 1>/dev/null || pkglist="$pkglist make"
which gcc  2>&1 1>/dev/null || pkglist="$pkglist gcc"
which tmux 2>&1 1>/dev/null || pkglist="$pkglist tmux"
which curl 2>&1 1>/dev/null || pkglist="$pkglist curl"
which rngd 2>&1 1>/dev/null || pkglist="$pkglist rng-tools"

if [ "$pkglist" != "" ]; then
    echo "missing packages: $pkglist"
    echo -n "would you like to install the missing packages? (need sudo) y/n> "
    read yesno
    if [ $yesno = "y" ]; then
        [ "$isRPM" != true ] && (sudo apt-get update || fail)
        sudo $PKG install $pkglist || fail
    fi
fi
if [ "$installRabbitRPM" = true ]; then
    sudo rpm -Uvh http://www.rabbitmq.com/releases/rabbitmq-server/v3.5.1/rabbitmq-server-3.5.1-1.noarch.rpm
fi
if [ "$isRPM" = true ]; then
    sudo service rabbitmq-server stop
    sudo service rabbitmq-server start || fail
    sudo service postgresql initdb
    PGHBA=$(sudo find /var/lib -name pg_hba.conf | tail -1)
    echo -e "\n---- Adding password authorization to $PGHBA\n"
    echo 'host    all             all             127.0.0.1/32            password' > /tmp/hba
    sudo grep -Ev "^#|^$" $PGHBA  >> /tmp/hba
    sudo mv /tmp/hba $PGHBA
    sudo service postgresql restart
fi

echo -e "\n---- Building MIG Scheduler\n"
make mig-scheduler || fail
id mig || sudo useradd -r mig || fail
sudo cp bin/linux/amd64/mig-scheduler /usr/local/bin/ || fail
sudo chown mig /usr/local/bin/mig-scheduler || fail
sudo chmod 550 /usr/local/bin/mig-scheduler || fail

echo -e "\n---- Building MIG API\n"
make mig-api || fail
sudo cp bin/linux/amd64/mig-api /usr/local/bin/ || fail
sudo chown mig /usr/local/bin/mig-api || fail
sudo chmod 550 /usr/local/bin/mig-api || fail

echo -e "\n---- Building MIG Worker\n"
make worker-agent-verif || fail
sudo cp bin/linux/amd64/mig-worker-agent-verif /usr/local/bin/ || fail
sudo chown mig /usr/local/bin/mig-worker-agent-verif || fail
sudo chmod 550 /usr/local/bin/mig-worker-agent-verif || fail

echo -e "\n---- Building MIG Clients\n"
make mig-console || fail
sudo cp bin/linux/amd64/mig-console /usr/local/bin/ || fail
sudo chown mig /usr/local/bin/mig-console || fail
sudo chmod 555 /usr/local/bin/mig-console || fail

make mig-cmd || fail
sudo cp bin/linux/amd64/mig /usr/local/bin/ || fail
sudo chown mig /usr/local/bin/mig || fail
sudo chmod 555 /usr/local/bin/mig || fail

echo -e "\n---- Building Database\n"
cd database/
dbpass=$(cat /dev/urandom | tr -dc _A-Z-a-z-0-9 | head -c${1:-32})
sudo su - postgres -c "psql -c 'drop database mig'"
sudo su - postgres -c "psql -c 'drop role migadmin'"
sudo su - postgres -c "psql -c 'drop role migapi'"
sudo su - postgres -c "psql -c 'drop role migscheduler'"
sudo su - postgres -c "psql -c 'drop role migreadonly'"
bash createlocaldb.sh $dbpass || fail
cd ..

echo -e "\n---- Creating system user and folders\n"
sudo mkdir -p /var/cache/mig/{action/new,action/done,action/inflight,action/invalid,command/done,command/inflight,command/ready,command/returned} || fail
hostname > /tmp/agents_whitelist.txt
hostname --fqdn >> /tmp/agents_whitelist.txt
echo localhost >> /tmp/agents_whitelist.txt
sudo mv /tmp/agents_whitelist.txt /var/cache/mig/
sudo chown mig /var/cache/mig -R || fail
[ ! -d /etc/mig ] && sudo mkdir /etc/mig
sudo chown mig /etc/mig || fail

echo -e "\n---- Configuring RabbitMQ\n"
(ps faux|grep "/var/lib/rabbitmq"|grep -v grep) 2>&1 1> /dev/null
if [ $? -gt 0 ]; then
    sudo service rabbitmq-server restart || fail
fi
mqpass=$(cat /dev/urandom | tr -dc _A-Z-a-z-0-9 | head -c${1:-32})
sudo rabbitmqctl delete_user admin
sudo rabbitmqctl add_user admin $mqpass || fail
sudo rabbitmqctl set_user_tags admin administrator || fail

sudo rabbitmqctl delete_vhost mig
sudo rabbitmqctl add_vhost mig || fail
sudo rabbitmqctl list_vhosts || fail

sudo rabbitmqctl delete_user scheduler
sudo rabbitmqctl add_user scheduler $mqpass || fail
sudo rabbitmqctl set_permissions -p mig scheduler \
    '^(toagents|toschedulers|toworkers|mig\.agt\..*)$' \
    '^(toagents|toworkers|mig\.agt\.(heartbeats|results))$' \
    '^(toagents|toschedulers|toworkers|mig\.agt\.(heartbeats|results))$' || fail

sudo rabbitmqctl delete_user agent
sudo rabbitmqctl add_user agent $mqpass || fail
sudo rabbitmqctl set_permissions -p mig agent \
    '^mig\.agt\..*$' \
    '^(toschedulers|mig\.agt\..*)$' \
    '^(toagents|mig\.agt\..*)$' || fail

sudo rabbitmqctl delete_user worker
sudo rabbitmqctl add_user worker $mqpass || fail
sudo rabbitmqctl set_permissions -p mig worker \
    '^migevent\..*$' \
    '^migevent(|\..*)$' \
    '^(toworkers|migevent\..*)$'

echo -e "\n---- Creating Scheduler configuration\n"
cp conf/scheduler.cfg.inc /tmp/scheduler.cfg
sed -i "s|whitelist = \"/var/cache/mig/agents_whitelist.txt\"|whitelist = \"\"|" /tmp/scheduler.cfg || fail
sed -i "s/freq = \"87s\"/freq = \"3s\"/" /tmp/scheduler.cfg || fail
sed -i "s/password = \"123456\"/password = \"$dbpass\"/" /tmp/scheduler.cfg || fail
sed -i "s/user  = \"guest\"/user = \"scheduler\"/" /tmp/scheduler.cfg || fail
sed -i "s/pass  = \"guest\"/pass = \"$mqpass\"/" /tmp/scheduler.cfg || fail
sudo mv /tmp/scheduler.cfg /etc/mig/scheduler.cfg || fail
sudo chown mig /etc/mig/scheduler.cfg || fail
sudo chmod 750 /etc/mig/scheduler.cfg || fail
echo OK

echo -e "\n---- Creating API configuration\n"
cp conf/api.cfg.inc /tmp/api.cfg
sed -i "s/password = \"123456\"/password = \"$dbpass\"/" /tmp/api.cfg || fail
sudo mv /tmp/api.cfg /etc/mig/api.cfg || fail
sudo chown mig /etc/mig/api.cfg || fail
sudo chmod 750 /etc/mig/api.cfg || fail
echo OK

echo -e "\n---- Creating Worker configuration\n"
cp conf/agent-verif-worker.cfg.inc /tmp/agent-verif-worker.cfg
sed -i "s/pass = \"secretpassphrase\"/pass = \"$mqpass\"/" /tmp/agent-verif-worker.cfg || fail
sudo mv /tmp/agent-verif-worker.cfg /etc/mig/agent-verif-worker.cfg || fail
sudo chown mig /etc/mig/agent-verif-worker.cfg || fail
sudo chmod 750 /etc/mig/agent-verif-worker.cfg || fail
echo OK

echo -e "\n---- Starting Scheduler and API in TMUX under mig user\n"
sudo su mig -c "/usr/bin/tmux new-session -s 'mig' -d"
sudo su mig -c "/usr/bin/tmux new-window -t 'mig' -n '0' '/usr/local/bin/mig-scheduler'"
sudo su mig -c "/usr/bin/tmux new-window -t 'mig' -n '0' '/usr/local/bin/mig-api'"
sudo su mig -c "/usr/bin/tmux new-window -t 'mig' -n '0' '/usr/local/bin/mig_agent_verif_worker'"
echo OK

# Unset proxy related environment variables from this point on, since we want to ensure we are
# directly accessing MIG resources locally.
if [ ! -z "$http_proxy" ]; then
    unset http_proxy
fi
if [ ! -z "$https_proxy" ]; then
    unset https_proxy
fi

echo -e "\n---- Testing API status\n"
sleep 2
ret=$(curl -s http://localhost:12345/api/v1/heartbeat | grep "gatorz say hi")
[ "$?" -gt 0 ] && fail
echo OK - API is running

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
    home = "$HOME/.mig/"
    keyid = "$keyid"
EOF

echo -e "\n---- Creating investigator $(whoami) in database\n"
gpg --no-default-keyring --keyring ~/.mig/pubring.gpg \
    --secret-keyring ~/.mig/secring.gpg \
    --export -a $(whoami)@$(hostname) \
    > ~/.mig/$(whoami)-pubkey.asc || fail
echo -e "create investigator\n$(whoami)\nyes\nyes\nyes\n$HOME/.mig/$(whoami)-pubkey.asc\ny\n" | \
    /usr/local/bin/mig-console -q || fail

echo -e "\n---- Creating agent configuration\n"
cat > conf/mig-agent-conf.go << EOF
package main
import(
    "mig.ninja/mig"
    "time"
)
var TAGS = struct {
    Operator string \`json:"operator"\`
}{
    "MIGDemo",
}
var ISIMMORTAL bool = false
var MUSTINSTALLSERVICE bool = true
var DISCOVERPUBLICIP bool = false
var DISCOVERAWSMETA bool = true
var CHECKIN bool = false
var APIURL string = "http://localhost:1664/api/v1/"
var LOGGINGCONF = mig.Logging{
    Mode:   "file",
    Level:  "debug",
    File:   "/var/cache/mig/mig-agent.log",
}
var AMQPBROKER string = "amqp://agent:$mqpass@localhost:5672/mig"
var PROXIES = []string{}
var SOCKET string = "127.0.0.1:51664"
var HEARTBEATFREQ time.Duration = 30 * time.Second
var REFRESHENV time.Duration = 60 * time.Second
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
make mig-agent AGTCONF=conf/mig-agent-conf.go BUILDENV=demo || fail
sudo cp bin/linux/amd64/mig-agent-latest /sbin/mig-agent || fail
sudo chown root /sbin/mig-agent || fail
sudo chmod 500 /sbin/mig-agent || fail
sudo /sbin/mig-agent

sleep 5
/usr/local/bin/mig -i actions/integration_tests.json

cat << EOF

        -------------------------------------------------

MIG is up and running with one local agent.

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

Now to get started, launch /usr/local/bin/mig-console or /usr/local/bin/mig
EOF

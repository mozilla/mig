#!/usr/bin/env bash

# Default entry point for standard MIG docker image. When run with no specific environment
# variables, this configures the docker image to run as a standalone MIG demo. If MIGMODE
# is set to test, the docker image is used to execute integration tests.

fail() {
    echo configuration failed
    exit 1
}

# Configure the docker container for standalone execution of a MIG demo environment.
standalone_configure() {
	echo Performing initial container configuration...

	go install github.com/mozilla/mig/mig-scheduler || fail
	go install github.com/mozilla/mig/mig-api || fail
	go install -tags 'modmemory' github.com/mozilla/mig/client/mig || fail
	go install -tags 'modmemory' github.com/mozilla/mig/client/mig-console || fail
	go install -tags 'modmemory' github.com/mozilla/mig/mig-agent || fail

	sudo sh -c "echo 'host all all 127.0.0.1/32 password' >> /etc/postgresql/9.5/main/pg_hba.conf"
	sudo service postgresql restart || fail
	dbpass=$(cat /dev/urandom | tr -dc _A-Z-a-z-0-9 | head -c${1:-32})

	for user in 'migadmin' 'migapi' 'migscheduler'; do
		sudo -u postgres sh -c "psql -c 'CREATE ROLE $user;'" || fail
		sudo -u postgres sh -c "psql -c \"ALTER ROLE $user WITH NOSUPERUSER INHERIT NOCREATEROLE NOCREATEDB LOGIN PASSWORD '$dbpass';\"" || fail
	done
	sudo -u postgres sh -c "psql -c 'CREATE DATABASE mig';" || fail
	sudo -u postgres sh -c "psql -f /go/src/mig.ninja/mig/database/schema.sql mig;"

	sudo sh -c "cat > /etc/supervisor/conf.d/mig-scheduler.inactive << EOF
[program:mig-scheduler]
command=/go/bin/mig-scheduler
startretries=9999
autorestart=true
EOF"

	sudo sh -c "cat > /etc/supervisor/conf.d/mig-api.inactive << EOF
[program:mig-api]
command=/go/bin/mig-api
startretries=9999
autorestart=true
EOF"

	sudo sh -c "cat > /etc/supervisor/conf.d/mig-agent.inactive << EOF
[program:mig-agent]
command=/go/bin/mig-agent -d
startretries=9999
autorestart=true
EOF"

	echo 'NODENAME=rabbit@localhost' | sudo tee --append /etc/rabbitmq/rabbitmq-env.conf
	sudo service rabbitmq-server start
	mqpass=$(cat /dev/urandom | tr -dc _A-Z-a-z-0-9 | head -c${1:-32})

	sudo rabbitmqctl add_user admin $mqpass || fail
	sudo rabbitmqctl set_user_tags admin administrator || fail

	sudo rabbitmqctl delete_user guest || fail

	sudo rabbitmqctl add_vhost mig || fail
	sudo rabbitmqctl list_vhosts || fail

	sudo rabbitmqctl add_user scheduler $mqpass || fail
	sudo rabbitmqctl set_permissions -p mig scheduler \
		'^(toagents|toschedulers|toworkers|mig\.agt\..*)$' \
		'^(toagents|toworkers|mig\.agt\.(heartbeats|results))$' \
		'^(toagents|toschedulers|toworkers|mig\.agt\.(heartbeats|results))$' || fail

	sudo rabbitmqctl add_user agent $mqpass || fail
	sudo rabbitmqctl set_permissions -p mig agent \
		'^mig\.agt\..*$' \
		'^(toschedulers|mig\.agt\..*)$' \
		'^(toagents|mig\.agt\..*)$' || fail

	sudo rabbitmqctl add_user worker $mqpass || fail
	sudo rabbitmqctl set_permissions -p mig worker \
		'^migevent\..*$' \
		'^migevent(|\..*)$' \
		'^(toworkers|migevent\..*)$'

	sudo service rabbitmq-server stop
	sudo service postgresql stop

	sudo mkdir -p /etc/mig || fail
	sudo sh -c "cat /go/src/mig.ninja/mig/tools/api.cfg.demo | \
		sed \"s,APIPASS,${dbpass},\" > /etc/mig/api.cfg.demo"
	sudo sh -c "cat /go/src/mig.ninja/mig/tools/scheduler.cfg.demo | \
		sed \"s,SCHEDULERDBPASS,${dbpass},\" | \
		sed \"s,SCHEDULERMQPASS,${mqpass},\" > /etc/mig/scheduler.cfg.demo"
	sudo sh -c "cat /go/src/mig.ninja/mig/tools/mig-agent.cfg.demo | \
		sed \"s,AGENTPASS,${mqpass},\" > /etc/mig/mig-agent.cfg.demo"
}

# Generate a demonistration investigator and associated key material and configure the
# standalone docker image to make use of it.
standalone_userconfig() {
	sudo service rabbitmq-server restart
	sudo service postgresql restart

	# Configure a demo investigator
	mkdir -p ~/.mig
	gpg --batch --no-default-keyring --keyring ~/.mig/pubring.gpg --secret-keyring \
		~/.mig/secring.gpg --gen-key << EOF
Key-Type: 1
Key-Length: 1024
Subkey-Type: 1
Subkey-Length: 1024
Name-Real: $(whoami) Investigator
Name-Email: $(whoami)@localhost
Expire-Date: 12m
EOF
	keyid=$(gpg --no-default-keyring --keyring ~/.mig/pubring.gpg \
		--secret-keyring ~/.mig/secring.gpg --fingerprint \
		--with-colons $(whoami)@localhost | grep '^fpr' | cut -f 10 -d ':')
	cat > ~/.migrc << EOF
[api]
    url = "http://localhost:1664/api/v1/"
    skipverifycert = on
[gpg]
    home = "$HOME/.mig/"
    keyid = "$keyid"
[targets]
    macro = all:status='online'
EOF
	# Temporarily start the API up with API authentication disabled to add the initial
	# investigator
	sudo sh -c "cat /etc/mig/api.cfg.demo | \
		sed 's,enabled = on,enabled = off,' > /etc/mig/api.cfg"
	sudo mv /etc/supervisor/conf.d/mig-api.inactive \
		/etc/supervisor/conf.d/mig-api.conf
	sudo service supervisor start
	gpg --no-default-keyring --keyring ~/.mig/pubring.gpg \
		--secret-keyring ~/.mig/secring.gpg \
		--export -a $(whoami)@localhost \
		> ~/.mig/$(whoami)-pubkey.asc
	echo -e "create investigator\n$(whoami)\nyes\nyes\nyes\nyes\n$HOME/.mig/$(whoami)-pubkey.asc\ny\n" | \
		/go/bin/mig-console -q
	# Install the newly created pubkey in the agents keychain
	sudo mkdir -p /etc/mig/agentkeys
	sudo cp ~/.mig/$(whoami)-pubkey.asc /etc/mig/agentkeys/$(whoami)-pubkey.asc
	sudo sh -c "cat > /etc/mig/acl.cfg << EOF
{
  \"default\": {
    \"minimumweight\": 1,
    \"investigators\": {
      \"mig\": {
        \"fingerprint\": \"${keyid}\",
        \"weight\": 1
      }
    }
  }
}
EOF
"

	# Use the configurations installed for standalone mode and enable MIG
	# daemons
	sudo service supervisor stop
	sudo mv /etc/mig/api.cfg.demo /etc/mig/api.cfg
	sudo mv /etc/mig/scheduler.cfg.demo /etc/mig/scheduler.cfg
	sudo mv /etc/mig/mig-agent.cfg.demo /etc/mig/mig-agent.cfg

	sudo mv /etc/supervisor/conf.d/mig-scheduler.inactive \
		/etc/supervisor/conf.d/mig-scheduler.conf
	sudo mv /etc/supervisor/conf.d/mig-agent.inactive \
		/etc/supervisor/conf.d/mig-agent.conf
	sudo service supervisor start
}

# Start integration tests.
start_test() {
	# Sleep a number of seconds to give the agent time to register before we run the
	# test, the heartbeat interval is 30 seconds so 45 should be sufficient
	sleep 45
	mig -i /go/src/mig.ninja/mig/actions/integration_tests.json || exit 1
}

# Start demo environment, just spawns a shell.
start_demo() {
	bash
}

PATH=/go/bin:$PATH; export PATH
GOPATH=/go; export GOPATH

if [[ ! -f /.migconfigured ]]; then
	# The container hasn't been configured with a standalone configuration yet, apply
	# the configuration and note it as having completed.
	standalone_configure
	standalone_userconfig
	sudo touch /.migconfigured
fi

sudo service rabbitmq-server stop
sudo service postgresql stop
sudo service supervisor stop
sudo mkdir -p /var/run/rabbitmq
sudo chown rabbitmq:rabbitmq /var/run/rabbitmq
sudo rm -f /var/run/supervisor.sock /var/run/supervisord.pid
sudo service rabbitmq-server restart
sudo service postgresql start
echo Waiting for Postgres to be ready...
while true; do
	pg_isready
	if [[ $? -eq 0 ]]; then
		break
	fi
	sleep 1
done
sudo service supervisor start

if [[ $MIGMODE = "test" ]]; then
	start_test
else
	start_demo
fi

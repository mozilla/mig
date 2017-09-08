#!/usr/bin/env bash

fail() {
    echo configuration failed
    exit 1
}

go install mig.ninja/mig/mig-scheduler || fail
go install mig.ninja/mig/mig-api || fail
go install -tags 'modmemory' mig.ninja/mig/client/mig || fail
go install -tags 'modmemory' mig.ninja/mig/client/mig-console || fail
go install -tags 'modmemory' mig.ninja/mig/mig-agent || fail

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
startretries=20
EOF"

sudo sh -c "cat > /etc/supervisor/conf.d/mig-api.inactive << EOF
[program:mig-api]
command=/go/bin/mig-api
startretries=20
EOF"

sudo sh -c "cat > /etc/supervisor/conf.d/mig-agent.inactive << EOF
[program:mig-agent]
command=/go/bin/mig-agent -d
startretries=20
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


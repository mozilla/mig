#!/bin/bash

echo 'NODENAME=rabbit@localhost' | sudo tee --append /etc/rabbitmq/rabbitmq-env.conf

sudo service rabbitmq-server start

mqpass=$(cat /dev/urandom | tr -dc _A-Z-a-z-0-9 | head -c${1:-32})

# Configure RabbitMQ with the users we want, and some random passwords for now in the image. Note we do
# not agent agent users here; these are created on image execution.
sudo rabbitmqctl add_user admin $mqpass || exit 1
sudo rabbitmqctl set_user_tags admin administrator || exit 1

sudo rabbitmqctl delete_user guest || exit 1

sudo rabbitmqctl add_vhost mig || exit 1

sudo rabbitmqctl add_user scheduler $mqpass || exit 1
sudo rabbitmqctl set_permissions -p mig scheduler \
	'^(toagents|toschedulers|toworkers|mig\.agt\..*)$' \
	'^(toagents|toworkers|mig\.agt\.(heartbeats|results))$' \
	'^(toagents|toschedulers|toworkers|mig\.agt\.(heartbeats|results))$' || exit 1

sudo rabbitmqctl add_user worker $mqpass || exit 1
sudo rabbitmqctl set_permissions -p mig worker \
	'^migevent\..*$' \
	'^migevent(|\..*)$' \
	'^(toworkers|migevent\..*)$'

sudo service rabbitmq-server stop

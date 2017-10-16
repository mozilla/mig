#!/bin/bash

sudo service rabbitmq-server start || exit 1


sudo rabbitmqctl change_password admin $MIGRELAYADMINPASSWORD || exit 1
sudo rabbitmqctl change_password scheduler $MIGRELAYSCHEDULERPASSWORD || exit 1
sudo rabbitmqctl change_password worker $MIGRELAYWORKERPASSWORD || exit 1

for agent in $MIGRELAYAGENTS; do
	username=`echo $agent | awk -F: '{print $1}'`
	pw=`echo $agent | awk -F: '{print $2}'`

	# It's possible if the container was restarted the user could already exist, dont
	# bail if the user add fails
	sudo rabbitmqctl add_user $username $pw
	sudo rabbitmqctl change_password $username $pw || exit 1

	sudo rabbitmqctl set_permissions -p mig $username \
	'^mig\.agt\..*$' \
	'^(toschedulers|mig\.agt\..*)$' \
	'^(toagents|mig\.agt\..*)$' || exit 1

done

sudo service rabbitmq-server stop || exit 1

sudo /usr/sbin/rabbitmq-server

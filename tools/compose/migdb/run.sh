#!/bin/bash

sudo service postgresql start || exit 1
while true; do
	pg_isready
	if [[ $? -eq 0 ]]; then
		break
	fi
	sleep 1
done
sudo -u postgres sh -c "psql -c \"ALTER ROLE migadmin PASSWORD '$MIGDBADMINPASSWORD';\"" || exit 1
sudo -u postgres sh -c "psql -c \"ALTER ROLE migapi PASSWORD '$MIGDBAPIPASSWORD';\"" || exit 1
sudo -u postgres sh -c "psql -c \"ALTER ROLE migscheduler PASSWORD '$MIGDBSCHEDULERPASSWORD';\"" || exit 1
sudo service postgresql stop || exit 1

sudo -u postgres /usr/lib/postgresql/9.5/bin/postgres -D /var/lib/postgresql/9.5/main \
	-h '*' -c 'config_file=/etc/postgresql/9.5/main/postgresql.conf'

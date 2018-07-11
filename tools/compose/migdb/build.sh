#!/bin/bash

# Make sure we allow connections from external hosts
sudo sh -c 'echo "host all all samenet password" >> /etc/postgresql/9.5/main/pg_hba.conf'

sudo service postgresql start || exit 1

# Add our users, just assign a random password during image build so we can add the schema
dbpass=$(cat /dev/urandom | tr -dc _A-Z-a-z-0-9 | head -c${1:-32})
sudo -u postgres sh -c "psql -c 'CREATE ROLE migadmin;'" || exit 1
sudo -u postgres sh -c "psql -c \"ALTER ROLE migadmin WITH NOSUPERUSER INHERIT NOCREATEROLE NOCREATEDB LOGIN PASSWORD '$dbpass';\"" || exit 1

sudo -u postgres sh -c "psql -c 'CREATE ROLE migapi;'" || exit 1
sudo -u postgres sh -c "psql -c \"ALTER ROLE migapi WITH NOSUPERUSER INHERIT NOCREATEROLE NOCREATEDB LOGIN PASSWORD '$dbpass';\"" || exit 1

sudo -u postgres sh -c "psql -c 'CREATE ROLE migscheduler;'" || exit 1
sudo -u postgres sh -c "psql -c \"ALTER ROLE migscheduler WITH NOSUPERUSER INHERIT NOCREATEROLE NOCREATEDB LOGIN PASSWORD '$dbpass';\"" || exit 1

# Add the schema
sudo -u postgres sh -c "psql -c 'CREATE DATABASE mig';" || exit 1
sudo -u postgres sh -c "psql -f /go/src/github.com/mozilla/mig/database/schema.sql mig;"

sudo service postgresql stop || exit 1

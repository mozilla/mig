#! /usr/bin/env bash
[ ! -x $(which sudo) ] && echo "sudo isn't available, that won't work" && exit 1

genpass=1
pass=""
[ ! -z $1 ] && pass=$1 && echo "using predefined password '$pass'" && genpass=0

for user in "migadmin" "migapi" "migscheduler"; do
    [ $genpass -gt 0 ] && pass=$(cat /dev/urandom | tr -dc _A-Z-a-z-0-9 | head -c${1:-32})
    sudo su postgres -c "psql -c 'CREATE ROLE $user;'" 1>/dev/null
    [ $? -ne 0 ] && echo "ERROR: user creation failed." && exit 123
    sudo su postgres -c "psql -c \"ALTER ROLE $user WITH NOSUPERUSER INHERIT NOCREATEROLE NOCREATEDB LOGIN PASSWORD '$pass';\"" 1>/dev/null
    [ $? -ne 0 ] && echo "ERROR: user creation failed." && exit 123
    echo "Created user $user with password '$pass'"
done
sudo su postgres -c "psql -c 'CREATE DATABASE mig OWNER migadmin;'" 1>/dev/null
[ $? -ne 0 ] && echo "ERROR: database creation failed." && exit 123

sudo su postgres -c "psql -d mig -f schema.sql" 1>/dev/null
[ $? -ne 0 ] && echo "ERROR: tables creation failed." && exit 123

echo "MIG Database created successfully."

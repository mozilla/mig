#!/bin/bash

# Update scheduler configuration using the environment
sed -i "s/MIGDBHOST/$MIGDBHOST/g" /etc/mig/scheduler.cfg
sed -i "s/MIGDBSCHEDULERPASSWORD/$MIGDBSCHEDULERPASSWORD/g" /etc/mig/scheduler.cfg
sed -i "s/MIGRELAYHOST/$MIGRELAYHOST/g" /etc/mig/scheduler.cfg
sed -i "s/MIGRELAYSCHEDULERPASSWORD/$MIGRELAYSCHEDULERPASSWORD/g" /etc/mig/scheduler.cfg

/usr/bin/supervisord -c /etc/supervisor/supervisord.conf -n

#!/usr/bin/env bash
for t in $(tail -10000 /var/log/daemon.log | grep 'is not authorized' |awk '{print $12}'|sort|uniq|sed "s/'//g")
do
        if [ "$(grep $t /etc/mig/agents_whitelist.txt)" == "" ]
        then
                echo $t
        fi
done

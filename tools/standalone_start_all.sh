#!/usr/bin/env bash
sudo service rabbitmq-server restart
sudo service postgresql restart
sleep 10
sudo tmux -S /tmp/tmux-$(id -u mig)/default kill-session -t mig || echo "OK - No running MIG session found"
tmux new-session -s 'mig' -d
tmux new-window -t 'mig' -n '0' '/usr/local/bin/mig-scheduler'
tmux new-window -t 'mig' -n '1' '/usr/local/bin/mig-api'
tmux new-window -t 'mig' -n '2' 'sudo /sbin/mig-agent -d'
echo 'scheduler, api and agent started in tmux session, use "tmux attach" to open it'

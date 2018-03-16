# Even though we have a dependency established between the scheduler and RabbitMQ in
# our docker-compose.yml file, the scheduler is started before RabbitMQ actually has
# time to get up and running.
# By sleeping for two seconds, we have a reasonable enough guarantee RabbitMQ will be
# ready when the scheduler starts.  This prevents the scheduler from failing to
# initialize and crashing immediately.
sleep 2
/opt/mig/bin/mig-scheduler

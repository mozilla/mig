FROM ubuntu

WORKDIR /queue

# Install dependencies
RUN apt update &&\
    apt install -y rabbitmq-server

# Set up RabbitMQ with all of the users and such that we need.
# The admin, scheduler, agent, and worker users all have a password of "docker".
RUN echo "NODENAME=rabbit@localhost" | tee --append /etc/rabbitmq/rabbitmq-env.conf &&\
    service rabbitmq-server start &&\
    rabbitmqctl add_user admin docker &&\
    rabbitmqctl set_user_tags admin administrator &&\
    rabbitmqctl delete_user guest &&\
    rabbitmqctl add_vhost mig &&\
    rabbitmqctl list_vhosts &&\
    rabbitmqctl add_user scheduler docker &&\
    rabbitmqctl set_permissions -p mig scheduler \
		"^(toagents|toschedulers|toworkers|mig\.agt\..*)$" \
		"^(toagents|toworkers|mig\.agt\.(heartbeats|results))$" \
		"^(toagents|toschedulers|toworkers|mig\.agt\.(heartbeats|results))$" &&\
	rabbitmqctl add_user agent docker &&\
	rabbitmqctl set_permissions -p mig agent \
		"^mig\.agt\..*$" \
		"^(toschedulers|mig\.agt\..*)$" \
		"^(toagents|mig\.agt\..*)$" &&\
	rabbitmqctl add_user worker docker &&\
	rabbitmqctl set_permissions -p mig worker \
		"^migevent\..*$" \
		"^migevent(|\..*)$" \
		"^(toworkers|migevent\..*)$" &&\
    service rabbitmq-server stop


# Set the default command to run when starting the container
CMD ["/usr/sbin/rabbitmq-server"]

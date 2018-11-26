# Testing

This directory contains files that support testing MIG services such as the API against dockerized
backend infrastructure.  In particular, the `docker-compose.yml` file sets up Postgres and RabbitMQ
instances that we can connect to from outside Docker.

## Usage

First create the Docker containers

```
docker-compose -f ./docker-compose.yml up -d
```

Next, get a bash session in the Postgres container.  Assuming the container name for Postgres
(obtained by running `docker ps`) is testing_postgres_1, run the following to enter the container and then
set up Postgres.

```
docker exec -it testing_postgres_1 bash
psql -c "CREATE DATABASE mig;"
psql -f /var/lib/db/init_migapi_db.sql mig
exit
```

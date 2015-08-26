#! /usr/bin/env bash

PGDATABASE='mig'
PGUSER='migadmin'
PGPASS='MYDATABASEPASSWORD'
PGHOST='192.168.0.1'
PGPORT=5432

qfile=$(mktemp)
schedpass=$(< /dev/urandom tr -dc _A-Z-a-z-0-9 | head -c${1:-32})
apipass=$(< /dev/urandom tr -dc _A-Z-a-z-0-9 | head -c${1:-32})

# pgpass file follow 'hostname:port:database:username:password'
echo "$PGHOST:$PGPORT:$PGDATABASE:$PGUSER:$PGPASS" > ~/.pgpass
chmod 400 ~/.pgpass

cat > $qfile << EOF
\c postgres
CREATE ROLE migscheduler;
ALTER ROLE migscheduler LOGIN PASSWORD '$schedpass';

CREATE ROLE migapi;
ALTER ROLE migapi LOGIN PASSWORD '$apipass';
EOF

psql -U $PGUSER -d $PGDATABASE -h $PGHOST -p $PGPORT -c "\i $qfile"
psql -U $PGUSER -d $PGDATABASE -h $PGHOST -p $PGPORT -d mig -c "\i schema.sql"
echo "created users: migscheduler/$schedpass migapi/$apipass"
rm $qfile
rm -f ~/.pgpass

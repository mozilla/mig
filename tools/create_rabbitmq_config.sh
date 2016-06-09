#! /usr/bin/env bash

adminpass=$(< /dev/urandom tr -dc _A-Z-a-z-0-9 | head -c${1:-32})
schedpass=$(< /dev/urandom tr -dc _A-Z-a-z-0-9 | head -c${1:-32})
agentpass=$(< /dev/urandom tr -dc _A-Z-a-z-0-9 | head -c${1:-32})
workrpass=$(< /dev/urandom tr -dc _A-Z-a-z-0-9 | head -c${1:-32})

echo "creating rabbitmq users"
sudo rabbitmqctl add_user admin $adminpass
sudo rabbitmqctl set_user_tags admin administrator
sudo rabbitmqctl add_user scheduler $schedpass
sudo rabbitmqctl add_user agent $agentpass
sudo rabbitmqctl add_user worker $workrpass

echo "deleting guest user"
sudo rabbitmqctl delete_user guest

echo "creating 'mig' vhost"
sudo rabbitmqctl add_vhost mig

echo "creating ACLs for scheduler user"
sudo rabbitmqctl set_permissions -p mig scheduler \
        '^(toagents|toschedulers|toworkers|mig\.agt\..*)$' \
        '^(toagents|toworkers|mig\.agt\.(heartbeats|results))$' \
	'^(toagents|toschedulers|toworkers|mig\.agt\.(heartbeats|results))$'

echo "creating ACLs for agent user"
sudo rabbitmqctl set_permissions -p mig agent \
        '^mig\.agt\..*$' \
        '^(toschedulers|mig\.agt\..*)$' \
        '^(toagents|mig\.agt\..*)$'

echo "creating ACLs for worker user"
sudo rabbitmqctl set_permissions -p mig worker \
	'^migevent\..*$' \
	'^migevent(|\..*)$' \
	'^(toworkers|migevent\..*)$'

echo "writing configuration to /etc/rabbitmq/rabbitmq.config"
[ -e /etc/rabbitmq/rabbitmq.config ] && sudo cp /etc/rabbitmq/rabbitmq.config{,.bkp}
mqconf=$(mktemp)
echo '[
  {rabbit, [
         {ssl_listeners, [5671]},
         {ssl_options, [{cacertfile,	"/etc/rabbitmq/ca.crt"},
                        {certfile,		"/etc/rabbitmq/rabbitmq.crt"},
                        {keyfile,		"/etc/rabbitmq/rabbitmq.key"},
                        {verify,		verify_peer},
                        {fail_if_no_peer_cert,	true},
                        {versions, ["tlsv1.2", "tlsv1.1"]},
                        {ciphers,  [{dhe_rsa,aes_256_cbc,sha256},
                                    {dhe_rsa,aes_128_cbc,sha256},
                                    {dhe_rsa,aes_256_cbc,sha},
                                    {rsa,aes_256_cbc,sha256},
                                    {rsa,aes_128_cbc,sha256},
                                    {rsa,aes_256_cbc,sha}]}
        ]}
  ]}
].' > $mqconf
sudo mv $mqconf /etc/rabbitmq/rabbitmq.config

echo "set mirroring policy"
sudo rabbitmqctl -p mig set_policy mig-mirror-all "^(toschedulers|toagents|toworkers|mig(|event))\." '{"ha-mode":"all"}'

sudo chown rabbitmq /etc/rabbitmq/*
echo
echo "rabbitmq configured with the following users:"
echo "  admin       $adminpass"
echo "  scheduler   $schedpass"
echo "  agent       $agentpass"
echo "  worker      $workrpass"
echo
echo "copy ca.crt and rabbitmq.{crt,key} into /etc/rabbitmq/"
echo "then run $ service rabbitmq-server restart"

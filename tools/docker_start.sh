#!/usr/bin/env bash

standalone_services() {
	sudo service rabbitmq-server restart
	sudo service postgresql restart

	# Configure a demo investigator
	mkdir -p ~/.mig
	gpg --batch --no-default-keyring --keyring ~/.mig/pubring.gpg --secret-keyring \
		~/.mig/secring.gpg --gen-key << EOF
Key-Type: 1
Key-Length: 1024
Subkey-Type: 1
Subkey-Length: 1024
Name-Real: $(whoami) Investigator
Name-Email: $(whoami)@localhost
Expire-Date: 12m
EOF
	keyid=$(gpg --no-default-keyring --keyring ~/.mig/pubring.gpg \
		--secret-keyring ~/.mig/secring.gpg --fingerprint \
		--with-colons $(whoami)@localhost | grep '^fpr' | cut -f 10 -d ':')
	cat > ~/.migrc << EOF
[api]
    url = "http://localhost:1664/api/v1/"
    skipverifycert = on
[gpg]
    home = "$HOME/.mig/"
    keyid = "$keyid"
[targets]
    macro = all:status='online'
EOF
	# Temporarily start the API up with API authentication disabled to add the initial
	# investigator
	sudo sh -c "cat /etc/mig/api.cfg.demo | \
		sed 's,enabled = on,enabled = off,' > /etc/mig/api.cfg"
	sudo mv /etc/supervisor/conf.d/mig-api.inactive \
		/etc/supervisor/conf.d/mig-api.conf
	sudo service supervisor start
	gpg --no-default-keyring --keyring ~/.mig/pubring.gpg \
		--secret-keyring ~/.mig/secring.gpg \
		--export -a $(whoami)@localhost \
		> ~/.mig/$(whoami)-pubkey.asc
	echo -e "create investigator\n$(whoami)\nyes\nyes\nyes\nyes\n$HOME/.mig/$(whoami)-pubkey.asc\ny\n" | \
		/go/bin/mig-console -q
	# Install the newly created pubkey in the agents keychain
	sudo mkdir -p /etc/mig/keys
	sudo cp ~/.mig/$(whoami)-pubkey.asc /etc/mig/keys/$(whoami)-pubkey.asc
	sudo sh -c "cat > /etc/mig/acl.cfg << EOF
{
  \"default\": {
    \"minimumweight\": 1,
    \"investigators\": {
      \"mig\": {
        \"fingerprint\": \"${keyid}\",
        \"weight\": 1
      }
    }
  }
}
EOF
"

	# Use the configurations installed for standalone mode and enable MIG
	# daemons
	sudo service supervisor stop
	sudo mv /etc/mig/api.cfg.demo /etc/mig/api.cfg
	sudo mv /etc/mig/scheduler.cfg.demo /etc/mig/scheduler.cfg
	sudo mv /etc/mig/mig-agent.cfg.demo /etc/mig/mig-agent.cfg

	sudo mv /etc/supervisor/conf.d/mig-scheduler.inactive \
		/etc/supervisor/conf.d/mig-scheduler.conf
	sudo mv /etc/supervisor/conf.d/mig-agent.inactive \
		/etc/supervisor/conf.d/mig-agent.conf
	sudo service supervisor start
}

start_demo() {
	standalone_services
	bash
}

PATH=/go/bin:$PATH; export PATH
start_demo

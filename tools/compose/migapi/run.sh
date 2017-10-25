#!/bin/bash

# Update API configuration using the environment
sudo sed -i "s/MIGDBHOST/$MIGDBHOST/g" /etc/mig/api.cfg
sudo sed -i "s/MIGDBAPIPASSWORD/$MIGDBAPIPASSWORD/g" /etc/mig/api.cfg

# If we have been asked to generate an investigator, do so and apply the new user
# to the database.
if [[ $GENERATEINVESTIGATOR == "yes" && ! -f /miginvestigator/fingerprint.txt ]]; then
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
	sudo sed -i 's/enabled = on/enabled = off/' /etc/mig/api.cfg
	sudo service supervisor start
	gpg --no-default-keyring --keyring ~/.mig/pubring.gpg \
		--secret-keyring ~/.mig/secring.gpg \
		--export -a $(whoami)@localhost \
		> ~/.mig/$(whoami)-pubkey.asc
	# Make sure the database is ready before we try this
	while true; do
		env PGHOST=$MIGDBHOST pg_isready
		if [[ $? -eq 0 ]]; then
			break
		fi
		sleep 1
	done

	echo -e "create investigator\n$(whoami)\nyes\nyes\nyes\nyes\n$HOME/.mig/$(whoami)-pubkey.asc\ny\n" | \
		/go/bin/mig-console -q
	sudo service supervisor stop
	sudo rm -f /var/run/supervisor.sock
	sudo sed -i 's/enabled = off/enabled = on/' /etc/mig/api.cfg
	# Populate /miginvestigator with the key material we created so other containers have access to it
	sudo cp ~/.mig/$(whoami)-pubkey.asc /miginvestigator/pubkey.asc
	sudo sh -c "echo $keyid >> /miginvestigator/fingerprint.txt"
	sudo cp -R ~/.mig /miginvestigator/mig
fi

sudo /usr/bin/supervisord -c /etc/supervisor/supervisord.conf -n

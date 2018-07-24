#!/bin/bash

# If AGENTMODULES is set, rebuild the agent with the indicated module tags
if [[ ! -z "$AGENTMODULES" ]]; then
	sudo env GOPATH=/go \
		go install -tags "$AGENTMODULES" github.com/mozilla/mig/mig-agent
fi

# If CLIENTMODULES is set, rebuild the clients with the indicated module tags
if [[ ! -z "$CLIENTMODULES" ]]; then
	sudo env GOPATH=/go \
		go install -tags "$CLIENTMODULES" github.com/mozilla/mig/client/mig-console
	sudo env GOPATH=/go \
		go install -tags "$CLIENTMODULES" github.com/mozilla/mig/client/mig
fi

# Update API configuration using the environment
sudo sed -i "s/AGENTUSER/$AGENTUSER/g" /etc/mig/mig-agent.cfg
sudo sed -i "s/AGENTPASSWORD/$AGENTPASSWORD/g" /etc/mig/mig-agent.cfg
sudo sed -i "s/MIGRELAYHOST/$MIGRELAYHOST/g" /etc/mig/mig-agent.cfg
sudo sed -i "s/MIGAPIHOST/$MIGAPIHOST/g" /etc/mig/mig-agent.cfg

# If the environment indicates investigator generation is enabled, stage the key material
# and an ACL in the agents keyring/configuration; we also build the command line tools
# configuration using the same data.
if [[ $GENERATEINVESTIGATOR == "yes" ]]; then
	while [[ ! -f /miginvestigator/fingerprint.txt ]]; do
		sleep 1
	done
	sudo mkdir -p /etc/mig/agentkeys
	sudo cp /miginvestigator/pubkey.asc /etc/mig/agentkeys/pubkey.asc
	keyid=`head -1 /miginvestigator/fingerprint.txt`
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
	sudo rm -rf ~/.mig
	sudo cp -R /miginvestigator/mig ~/.mig
	sudo chown -R $(whoami) ~/.mig
	cat > ~/.migrc << EOF
[api]
    url = "http://$MIGAPIHOST:1664/api/v1/"
    skipverifycert = on
[gpg]
    home = "$HOME/.mig/"
    keyid = "$keyid"
[targets]
    macro = all:status='online'
EOF
fi

sudo /usr/bin/supervisord -c /etc/supervisor/supervisord.conf -n

#!/usr/bin/env bash

loaderkeyfile="/etc/mig/mig-loader.key"

chk() {
	if [[ -z "$1" ]]; then
		return 0
	fi
	return 1
}

cnt=0
while true; do
	buf=$(osascript << EOF
	display dialog "Please enter MIG registration token" default answer ""
	set ret to text returned of result
	return ret
	EOF
	)
	cnt=`expr $cnt + 1`
	chk $buf
	if [[ $? == "1" ]]; then
		break
	fi
	if [[ $cnt -eq 3 ]]; then
		break
	fi
done
echo $buf > $loaderkeyfile

pl=/Library/LaunchAgents/com.mozilla.mig-loader.plist
launchctl unload $pl
launchctl load $pl
launchctl start mig-loader

exit 0

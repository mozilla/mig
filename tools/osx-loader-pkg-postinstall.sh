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
	if [ $? -eq 1 ]; then exit 1; fi
	cnt=`expr $cnt + 1`
	chk $buf
	if [[ $? == "1" ]]; then
		break
	fi
	if [[ $cnt -eq 3 ]]; then
		exit 1
	fi
done
echo $buf > $loaderkeyfile

# Run the loader once as part of startup
/usr/local/bin/mig-loader -i

exit 0

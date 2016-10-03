#!/usr/bin/env bash

loaderkeyfile="/etc/mig/mig-loader.key"

chk() {
	if [[ -z "$1" ]]; then
		return 0
	fi
	return 1
}

cnt=0
buf=$(osascript << EOF
set tries to 0
repeat
	set done to false
	repeat 1 times
		display dialog "Please enter MIG registration token" default answer ""
		set ret to text returned of result
		set keylen to length of ret
		set valid to "0" = (do shell script ¬
			"egrep -q '^\\\w{40}$' <<<" & quoted form of ret & "; printf \$?")
		if not valid then
			exit repeat
		end if
		set done to true
	end repeat
	if done then
		exit repeat
	end if
	set tries to tries + 1
	display alert "Invalid key format" as critical message "Your key should be 40 alphanumeric characters long, on a single line"
	if tries is greater than 2 then
		display alert "Giving up" as critical message "Verify your key is valid and try the installation again"
		error "invalidkey"
	end if
end repeat
return ret
EOF
)
if [ $? -eq 1 ]; then exit 1; fi

echo $buf > $loaderkeyfile

# Run the loader once as part of startup
/usr/local/bin/mig-loader -i

exit 0

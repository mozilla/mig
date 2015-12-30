#!/bin/bash
# Julien Vehent - 2014
# watch the MIG source code directory and rebuild the components
# when a file is saved.

echo "starting inotify listener on ./src/mig"
# feed the inotify events into a while loop that creates
# the variables 'date' 'time' 'dir' 'file' and 'event'
inotifywait -mr --timefmt '%d/%m/%y %H:%M' --format '%T %w %f %e' \
-e modify ./src/mig \
| while read date time dir file event
do
    if [[ "$file" =~ \.go$ && "$dir" =~ src\/mig ]]; then
        dontskip=true
    else
        #echo skipping $date $time $event $dir $file
        continue
    fi

    #echo NEW EVENT: $event $dir $file

    if [[ "$dir" =~ src\/mig\/$ ]]; then
        echo
        make mig-agent && \
        make mig-cmd && \
        make mig-api && \
        make mig-scheduler && \
        echo success $(date +%H:%M:%S)

    elif [[ "$dir" =~ agent && "$file" != "configuration.go" ]] ; then
        echo
        make mig-agent && echo success $(date +%H:%M:%S)

    elif [[ "$dir" =~ modules && "$file" != "configuration.go" ]] ; then
        echo
        make mig-agent && echo success $(date +%H:%M:%S)

    elif [[ "$dir" =~ api ]] ; then
        echo
        make mig-api && echo success $(date +%H:%M:%S)

    elif [[ "$dir" =~ client\/generator ]] ; then
        echo
        make mig-action-generator && echo success $(date +%H:%M:%S)

    elif [[ "$dir" =~ client\/verifier ]] ; then
        echo
        make mig-action-verifier && echo success $(date +%H:%M:%S)

    elif [[ "$dir" =~ client\/console ]] ; then
        echo
        make mig-console && echo success $(date +%H:%M:%S)

    elif [[ "$dir" =~ client\/cmd ]] ; then
        echo
        make mig-console && echo success $(date +%H:%M:%S)

    elif [[ "$dir" =~ client ]] ; then
        echo
        make mig-console && \
        make mig-cmd && \
        echo success $(date +%H:%M:%S)

    elif [[ "$dir" =~ workers ]] ; then
        echo
        make worker-agent-verif && \
        make worker-agent-intel && \
        echo success $(date +%H:%M:%S)

    elif [[ "$dir" =~ workers\/agent_intel ]] ; then
        echo
        make worker-agent-intel && echo success $(date +%H:%M:%S)

    elif [[ "$dir" =~ workers\/agent_verif ]] ; then
        echo
        make worker-agent-verif && echo success $(date +%H:%M:%S)

    elif [[ "$dir" =~ workers\/compliance_item ]] ; then
        echo
        make worker-compliance-item && echo success $(date +%H:%M:%S)

    elif [[ "$dir" =~ pgp ]] ; then
        echo
        make mig-agent && \
        make mig-cmd && \
        make mig-api && \
        make mig-scheduler && \
        echo success $(date +%H:%M:%S)

    elif [[ "$dir" =~ scheduler ]] ; then
        echo
        make mig-scheduler && echo success $(date +%H:%M:%S)

    fi
done

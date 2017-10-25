; Sample MIG Loader configuration file

[loader]
    ; location of the mig api
    api             = "http://localhost:1664/api/v1/"

    ; comma delimited list of host:port proxies to use, if desired
    ; the loader will attempt to try to proxies for manifest retrieval
    ; proxies          = "proxy1:8888,proxy2:8888"

    ; attempt to retrieve the public IP behind which the loader is running
    discoverpublicip = off

    ; attempt to retrieve AWS metadata from the metadata service to include in the environment
    ; sent to the API
    discoverawsmeta = off

    ; the number of signatures required in a manifest for the manifest to be considered
    ; valid
    requiredsignatures = 1

    ; Tags can be specified for a given loader at compile-time using the loader built-in
    ; configuration TAGS value. Additional tags can be included in the configuration file
    ; here if desired to override or extend the tags the loader has already been compiled
    ; with.
    ; tags = "tagname:tagvalue"

[logging]
    mode    = "stdout" ; stdout | file | syslog
    level   = "debug"

; for file logging
;   file = "mig_scheduler.log"
;   maxfilesize = 0 ; if > 0, log file will be rotated once it reaches size

; for syslog, logs go into local3
;    host   = "localhost"
;    port   = 514
;    protocol = "udp"

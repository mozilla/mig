; Sample MIG API configuration file

[authentication]
    # turn this on after initial setup, once you have at least
    # one investigator created
    enabled = off

    # when validating token timestamps, accept a timestamp that is
    # within this duration of the local clock
    tokenduration = 10m

[manifest]
    # used with mig manifests, this indicates the number of valid signatures
    # that must be applied to a manifest for the api to mark it as active
    requiredsignatures = 2

[server]
    # local listening ip
    ip = "127.0.0.1"

    # local listening port
    port = 12345

    # public location of the API endpoint
    host = "http://localhost:12345"

    # API base route, all endpoints are below this path
    # ex: http://localhost:12345/api/v1/action/create/
    #     |------<host>--------|<base>|--<endpoint>--|
    baseroute = "/api/v1"

    # informs the api where it should obtain the clients public ip address
    # from. the default if unset is "peer".
    #
    # use the X-Forwarded-For header, the trailing integer
    # indicates an offset from the end of the list of addresses in
    # x-forwarded-for to use as the client public ip:
    #clientpublicip = x-forwarded-for:0
    # use socket peer address:
    #clientpublicip = peer

[postgres]
    host = "127.0.0.1"
    port = 5432
    dbname = "mig"
    user = "migapi"
    password = "123456"
    sslmode = "disable"

[logging]
    mode = "stdout" ; stdout | file | syslog
    level = "debug"

; for file logging
;   file = "mig_api.log"

; for syslog, logs go into local3
;    host = "localhost"
;    port = 514
;    protocol = "udp"

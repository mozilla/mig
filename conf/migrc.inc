[api]
    # location of the api endpoint
    url = "http://localhost:12345/api/v1/"

    # set this to `on` if you use https with a self signed cert
    skipverifycert = off

[gpg]
    # location of the gnupg directory of the investigator
    home = "/home/bob/.gnupg/"

    # pgp fingerprint of the investigator
    keyid = "E60892BB9BD89A69F759A1A0A3D652173B763E8F"

# the targets section lets you define custom user macros you can use as an
# argument to the -t flag with mig, or settargets in mig-console rather
# than having to specify a full targeting string.
#[targets]
#    macro = all:status='online'
#    macro = onlineandidle:status='online' OR status='idle'

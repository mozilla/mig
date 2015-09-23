; Sample MIG runner configuration file

[runner]
    directory = "./runner" ; The path to the root runner directory
    checkdirectory = 30    ; How often to check runners/ for job changes

[logging]
    mode = "stdout" ; stdout | file | syslog
    level = "debug"

[client]
    clientconfpath = "default" ; Path to client conf, default for $HOME/.migrc
    ; If the private key has a passphrase associated with it, this must be
    ; set to the correct value.
    passphrase = "passphrase" ; Private key passphrase
    delayresults = "30s" ; Duration after action expiry to fetch action results

#!/usr/bin/env python
import os
import sys
import gnupg
from time import gmtime, strftime
import random
import requests
import json

def makeToken(gpghome, keyid):
    gpg = gnupg.GPG(gnupghome=gpghome)
    version = "1"
    timestamp = strftime("%Y-%m-%dT%H:%M:%SZ", gmtime())
    nonce = str(random.randint(10000, 18446744073709551616))
    token = version + ";" + timestamp + ";" + nonce
    sig = gpg.sign(token + "\n",
        keyid=keyid,
        detach=True, clearsign=True)
    token += ";"
    linectr=0
    for line in iter(str(sig).splitlines()):
        linectr+=1
        if linectr < 4 or line.startswith('-') or not line:
            continue
        token += line
    return token

if __name__ == '__main__':
    token = makeToken("/home/ulfr/.gnupg", "E60892BB9BD89A69F759A1A0A3D652173B763E8F")
    r = requests.get(sys.argv[1],
        headers={'X-PGPAUTHORIZATION': token},
        verify=True)
    if r.status_code == 200:
        print json.dumps(r.json(), sort_keys=True, indent=4, separators=(',', ': '))
    elif r.status_code == 500:
        print r.json()
        # api returns a 500 with an error body on failures
        migjson=r.json()
        raise Exception("API returned HTTP code %s and error '%s:%s'" %
                            (r.status_code,
                            migjson['collection']['error']['code'],
                            migjson['collection']['error']['message'])
                        )
    else:
        # another type of failure that's unlikely to have an error body
        raise Exception("Failed with HTTP code %s" % r.status_code)

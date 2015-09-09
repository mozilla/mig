echo "creating openssl config"
[ ! -e openssl.cnf ] && echo '[ ca ]
default_ca              = default_CA
[ default_CA ]
dir                     = ./ca
certs                   = $dir
new_certs_dir           = $dir/certs
database                = $dir/index
serial                  = $dir/serial
RANDFILE                = $dir/random-bits
certificate             = $dir/ca.crt
private_key             = $dir/ca.key
default_days            = 1095
default_crl_days        = 30
default_md              = sha256
preserve                = no
x509_extensions         = server_cert
policy                  = policy_anything

[ policy_anything ]
countryName             = optional
stateOrProvinceName     = optional
localityName            = optional
organizationName        = optional
organizationalUnitName  = optional
commonName              = supplied
emailAddress            = optional

[req]
distinguished_name 	= req_distinguished_name

[req_distinguished_name]
countryName                     = Country Name (2 letter code)
countryName_default             = US
countryName_min                 = 2
countryName_max                 = 2
stateOrProvinceName             = State or Province Name (full name)
stateOrProvinceName_default     = Florida
localityName                    = Locality Name (eg, city)
localityName_default            = Gator Town
0.organizationName              = Organization Name (eg, company)
0.organizationName_default      = Mozilla
organizationalUnitName          = Organizational Unit Name (eg, section)
organizationalUnitName_default  = MIG
commonName                      = Common Name (eg, YOUR name)
commonName_max                  = 64

[ root_ca ]
nsComment                       = "MIG Certificate Authority"
subjectKeyIdentifier            = hash
authorityKeyIdentifier          = keyid,issuer:always
basicConstraints                = critical,CA:TRUE,pathlen:1
keyUsage                        = keyCertSign, cRLSign

[v3_req]
basicConstraints 	= CA:FALSE
keyUsage 		= digitalSignature, keyEncipherment

[ server_cert ]
authorityKeyIdentifier  = keyid,issuer:always
issuerAltName           = issuer:copy
extendedKeyUsage        = serverAuth,clientAuth,msSGC,nsSGC
basicConstraints        = critical,CA:false
keyUsage                = digitalSignature,nonRepudiation,keyEncipherment
nsCertType              = server,client' > openssl.cnf

mkdir -p ca/certs
[ ! -e ca/serial ] && echo 1000 > ca/serial
[ ! -e ca/index ] && touch ca/index


echo "creating a key for the root certificate authority"
openssl genrsa -out ca/ca.key 2048
openssl req -new -x509 -days 3650 -key ca/ca.key -out ca/ca.crt -config openssl.cnf -extensions root_ca << EOF
US
Florida
Gator Town
MIG
CA
$(hostname --fqdn)
postmaster@$(hostname --fqdn)
EOF

echo "create certificate for rabbitmq"
echo -n "enter the public dns name of the rabbitmq server agents will connect to> "
read rabbitmqhostname
openssl req -new -newkey rsa:2048 -keyout rabbitmq.key -nodes -out rabbitmq.csr -config openssl.cnf << EOF
US
Florida
Gator Town
MIG
RabbitMQ Relay
${rabbitmqhostname}
postmaster@${rabbitmqhostname}
EOF
openssl ca -extensions server_cert -in rabbitmq.csr -out rabbitmq.crt -config openssl.cnf

echo "create certificates for scheduler, workers and agents"
for target in scheduler worker agent
do
	openssl req -new -newkey rsa:2048 -keyout $target.key -nodes -out $target.csr -config openssl.cnf << EOF
US
Florida
Gator Town
MIG
$target
${target}.$(hostname --fqdn)
postmaster@${target}.$(hostname --fqdn)
EOF
	yes | openssl ca -extensions server_cert -in $target.csr -out $target.crt -config openssl.cnf
done

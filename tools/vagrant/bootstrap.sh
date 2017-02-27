#!/usr/bin/env bash
set -x

user=${1:-$(whoami)}

# Install Go
goversion=1.7.4
echo "installing go $goversion ..."
gofile=go${goversion}.linux-amd64.tar.gz
gourl=https://storage.googleapis.com/golang/${gofile}
wget -q -O /usr/local/${gofile} ${gourl}
mkdir /usr/local/go
tar -xzf /usr/local/${gofile} -C /usr/local/go --strip 1

# Link the mig directory into the $GOPATH
export GOPATH=/home/${user}/go
mkdir -p $GOPATH/src/mig.ninja
chown -R ${user}.${user} $GOPATH
ln -s /mig $GOPATH/src/mig.ninja/mig
echo "export GOPATH=/home/${user}/go" >> /home/${user}/.profile
echo "PATH=/usr/local/go/bin:\$GOPATH/bin:\$PATH" >> /home/${user}/.profile

echo "ALL DONE!!!!"

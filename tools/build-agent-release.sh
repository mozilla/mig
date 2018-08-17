BUILDREV=$(date +%Y%m%d)-0.$(git log --pretty=format:'%h' -n 1).prod

# update_releases invokes the update_release_json tool to record an agent update.
# It expects to be called with three arguments:
# 1. The path to the releases.json file to update,
# 2. The tag for the new release and
# 3. The path to the new agent binary.
function update_releases() {
  releasejson=$(go run tools/update_release_json.go \
    -releases $1 \
    -component agent \
    -tag $2 \
    -sha256 $(cat $3 | openssl sha256))

  echo "$releasejson" > $1
}


GOOS=darwin GOARCH=386 go build -o releases/mig-agent-$BUILDREV github.com/mozilla/mig/mig-agent
if [ $? -eq 0 ]; then
  update_releases releases/releases.json $BUILDREV releases/mig-agent-$BUILDREV
fi

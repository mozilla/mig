releasejson=$(go run tools/update_release_json.go \
  -releases releases/releases.json \
  -component agent \
  -tag "$2$3" \
  -sha256 $(cat $1/mig-agent-$2$3 | openssl sha256))

echo "$releasejson" > releases/releases.json

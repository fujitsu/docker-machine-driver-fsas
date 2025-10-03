#!/bin/bash
if [[ ! $1 ]]; then
  echo "Pass tag as param e.g. $0 v1.2.3"
  exit 2
fi

tag=$1
url="https://github.com/fujitsu/docker-machine-driver-fsas/releases/download/$tag/docker-machine-driver-fsas"

sed -i 's/<sha256_checksum>/'$(sha256sum docker-machine-driver-fsas | cut -d" " -f 1)'/g' ./ci_tools/fsas_nodedriver_template.yml
sed -i 's#<fsas_nodedriver_binary_url>#'$url'#g' ./ci_tools/fsas_nodedriver_template.yml
mv ./ci_tools/fsas_nodedriver_template.yml ./docker-machine-driver-fsas.yml
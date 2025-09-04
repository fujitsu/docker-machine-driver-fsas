#!/bin/bash
sed -i 's/<sha256_checksum>/'$(sha256sum docker-machine-driver-fsas | cut -d" " -f 1)'/g' ./ci_tools/fsas_nodedriver_template.yml
sed -i 's#<fsas_nodedriver_binary_url>#'https://github.com/fujitsu/docker-machine-driver-fsas/releases/download/\<placeholder\>/docker-machine-driver-fsas'#g' ./ci_tools/fsas_nodedriver_template.yml
mv ./ci_tools/fsas_nodedriver_template.yml ./docker-machine-driver-fsas.yml
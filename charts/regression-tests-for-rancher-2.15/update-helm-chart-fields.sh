#!/usr/bin/env bash

set -euo pipefail

# variables for updating helm chart fields
declare -A pairs=(
    ['<cc-id>']='cc-123'
    ['<cc-id-ntp-url>']='cc-234'
    ['<cc-id-no-email>']='cc-345'
    ['<cc-id-no-reg-code>']='cc-456'
    ['<cc-id-reg-ok>']='cc-567'
    ['<netb-id>']='netb-345'
    ['<netp-id>']='netp-456'
    ['<netp-gw>']='netp-gw-567'
    ['<os-image-name>']='os-image-678'
    ['<ssh-user>']='ssh-user-789'
    ['<ssh-pwd>']='ssh-pwd-890'
    ['<image-ssh-pub-key>']='image-ssh-pub-key-901'
    ['<comp-cond-json>']='comp-cond-1-json-012'
    ['<dev-spec-json>']='dev-spec-1-json-123'
    ['<dev-spec-json-1-comp-nic-prov>']='dev-spec-1-comp-nic-234'
    ['<dev-spec-json-1-comp-nic-bare>']='dev-spec-1-comp-nic-235'
    ['<dev-spec-json-1-comp-nic-3rd-sub>']='dev-spec-1-comp-nic-236'

)

# update helm chart fields using the above variables

# for i in $(ls helm-chart*.yaml); do
for i in $(ls "helm-chart-test-3589.yaml"); do
    echo "--> updating file: $i";
    for key in "${!pairs[@]}"; do
        echo "--> updating key: $key with value: ${pairs[$key]}"
        # sed -i "s|\${$key}|${pairs[$key]}|g" "$i"
        sed "s|\${$key}|${pairs[$key]}|g" "$i"
    done
done
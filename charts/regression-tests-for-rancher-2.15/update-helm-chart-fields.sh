#!/usr/bin/env bash

set -euo pipefail

# variables for updating helm chart fields
declare -A pairs=(
    ['<cc-id>']='cc-222'
    ['<cc-id-ntp-url>']='cc-234'
    ['<cc-id-no-email>']='cc-345'
    ['<cc-id-no-reg-code>']='cc-456'
    ['<cc-id-reg-ok>']='cc-567'
    ['<netb-id>']='netb-345'
    ['<netp-id>']='netp-456'
    ['<netp-gw>']='netp-gw-2'
    ['<os-image-name>']='os-image-678'
    ['<ssh-user>']='ssh-user-789'
    ['<ssh-pwd>']='ssh-pwd-890'
    ['<image-ssh-pub-key>']='image-ssh-pub-key-901'
    ['<comp-cond-json>']='comp-cond-1-json-012'
    ['<dev-spec-json>']='dev-spec-1-json-123'
    ['<dev-spec-json-1-comp-nic-prov>']='dev-spec-1-comp-nic-234'
    ['<dev-spec-json-1-comp-nic-bare>']='dev-spec-1-comp-nic-235'
    ['<dev-spec-json-1-comp-nic-3rd-sub>']='dev-spec-1-comp-nic-236'
    ['<dev-spec-json-2-comp-nic-prov-bare>']='dev-spec-2-comp-nic-prov-bare-345'
    ['<dev-spec-json-2-comp-nic-3rd-sub>']='dev-spec-2-comp-nic-3rd-sub-456'
)

function print_red(){
    printf "\e[31m$1\e[0m"
}
function print_yellow(){
    printf "\e[33m$1\e[0m"
}
function print_green(){
    printf "\e[32m$1\e[0m"
}

keys=''

# update helm chart fields using the array 'pairs'
for i in helm-chart-*.yaml; do
    printf "=> updating file: $i ... ";
    for key in "${!pairs[@]}"; do
        # debug "#debug# updating key: $key with value: ${pairs[$key]}\n"
        sed -i "s#${key}#${pairs[$key]}#g" "$i"

        # collect keys
        keys+="$key|"
    done

    # delete last delimiter from collected keys (pipe |)
    keys="${keys%|}"
    #debug printf "\n### keys used for updating: $keys\n"

    # after replacing keys check if by accident any defined placeholders are left
    if [[ $(egrep "$keys" "$i" | wc -l) -ne 0 ]]; then
        print_red "Warning: defined placeholders left:\n"
        result=$(egrep -n "$keys" "$i")
        print_red "    $result\n"
    fi

    # after replacing keys check if by accident any generic placeholders are left
    if [[ $(grep '<[^>]*>' "$i" | wc -l) -ne 0 ]]; then
        print_yellow "    Warning: generic placeholders left:\n"
        print_yellow "    $(grep -n '<[^>]*>' "$i")\n"
    fi

    if [[ $(egrep "$keys" "$i" | wc -l) -eq 0 && $(grep '<[^>]*>' "$i" | wc -l) -eq 0 ]] ; then
        print_green "OK\n"
    fi

done
#!/usr/bin/env bash

set -euo pipefail

# variables for updating helm chart fields
declare -A pairs=(
    ['<cc-id>']='<cc-id>'
    ['<cc-id-ntp-url>']='<cc-id-ntp-url>'
    ['<cc-id-no-email>']='<cc-id-no-email>'
    ['<cc-id-no-reg-code>']='<cc-id-no-reg-code>'
    ['<cc-id-reg-ok>']='<cc-id-reg-ok>'
    ['<netb-id>']='<netb-id>'
    ['<netb-gw>']='<netb-gw>'
    ['<netp-id>']='<netp-id>'
    ['<netp-gw>']='<netp-gw>'
    ['<os-image-name>']='<os-image-name>'
    ['<ssh-user>']='<ssh-user>'
    ['<first-login-ssh-key>']='<first-login-ssh-key>'
    ['<image-ssh-pub-key>']='<image-ssh-pub-key>'
    ['<comp-cond-json>']='<comp-cond-json>'
    ['<dev-spec-json>']='<dev-spec-json>'
    ['<dev-spec-json-1-comp-nic-prov>']='<dev-spec-json-1-comp-nic-prov>'
    ['<dev-spec-json-1-comp-nic-bare>']='<dev-spec-json-1-comp-nic-bare>'
    ['<dev-spec-json-1-comp-nic-3rd-sub>']='<dev-spec-json-1-comp-nic-3rd-sub>'
    ['<dev-spec-json-2-comp-nic-prov-bare>']='<dev-spec-json-2-comp-nic-prov-bare>'
    ['<dev-spec-json-2-comp-nic-3rd-sub>']='<dev-spec-json-2-comp-nic-3rd-sub>'
)

# IMPORTANT: in array "pairs" adjust number of spaces in multiline fields e.g. firstLoginSshKey to match
# indentation in yaml files e.g.
# ['<first-login-ssh-key>']='-----BEGIN OPENSSH PRIVATE KEY-----
#      line1
#      line2
#      -----END OPENSSH PRIVATE KEY-----'


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
    # for key in "${!pairs[@]}"; do
    #     # debug "#debug# updating key: $key with value: ${pairs[$key]}\n"
    #     sed -i "s#${key}#${pairs[$key]}#g" "$i"
    #     # collect keys
    #     keys+="$key|"
    # done
    for key in "${!pairs[@]}"; do
        KEY="$key" VALUE="${pairs[$key]}" \
            perl -0pi -e 's/\Q$ENV{KEY}\E/$ENV{VALUE}/g' "$i"

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
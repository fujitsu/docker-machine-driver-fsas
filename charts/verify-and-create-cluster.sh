#!/usr/bin/env bash

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'
kube_version='v1.34.3'

function printf_red() {
  printf "${RED}${1}${NC}"
}

function printf_green() {
  printf "${GREEN}${1}${NC}"
}

if [[ -z $1 ]]; then
  echo "Pass config file as param e.g. values.custom.yaml. Abort"
  exit 1
fi

cfile="$1"

printf "==> Linting values file ... "
lint_response=$(helm lint . -f $cfile --kube-version "$kube_version")

echo $lint_response | egrep 'WARN|ERROR'
if [[ $? -eq 0 ]]; then
  printf_red "\n==> Linting erors. Abort\n"
  exit 1
fi
printf_green "OK\n"

printf "==> Rendering result file ... "
rfile='rendered-fsas-cluster.yaml'
if [[ -f $rfile ]]; then
  rm $rfile
fi

helm template fsas-cluster . -f $cfile --kube-version "$kube_version" > $rfile
if [[ $? -ne 0 ]]; then
  printf_red "\nRendering error. Abort\n"
  exit 1
fi
printf_green "OK\n"

printf "==> Verifying rendered file ... "
egrep "^kind: Cluster$" $rfile 1>/dev/null && egrep "^kind: FsasConfig$" rendered-fsas-cluster.yaml 1>/dev/null

if [[ $? -eq 0 ]]; then
  printf_green "OK\n"
else
  printf_red "\n==> Fail! Rendered file is not valid. Abort\n"
  exit 1
fi

 printf "==> Start building a cluster\n"
 kubectl apply -f $rfile
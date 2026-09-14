#!/usr/bin/env bash

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

function printf_red() {
  printf "${RED}${1}${NC}"
}

function printf_green() {
  printf "${GREEN}${1}${NC}"
}

cfile='values.custom.yaml'

printf "==> Linting values file ... "
lint_response=$(helm lint . -f $cfile)

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

helm template fsas-cluster . -f $cfile >$rfile
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
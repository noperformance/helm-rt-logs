#!/bin/sh

set -e

current_version=$(sed -n -e 's/version:[ "]*\([^"]*\).*/\1/p' $(dirname $0)/plugin.yaml)
HELM_RT_LOGS_VERSION=${HELM_RT_LOGS_VERSION:-$current_version}

dir=${HELM_PLUGIN_DIR:-"$(helm home)/plugins/helm-rt-logs"}
os=$(uname -s | tr '[:upper:]' '[:lower:]')
release_file="helm-rt-logs_${os}_${HELM_RT_LOGS_VERSION}.tar.gz"
url="https://github.com/noperformance/helm-rt-logs/releases/download/v${HELM_RT_LOGS_VERSION}/${release_file}"

mkdir -p $dir

if command -v wget
then
  wget -O ${dir}/${release_file} $url
elif command -v curl; then
  curl -L -o ${dir}/${release_file} $url
fi

tar xvf ${dir}/${release_file} -C $dir

chmod +x ${dir}/helm-rt-logs

rm ${dir}/${release_file}

#!/usr/bin/env bash

# Prevent direct run of the script
if [ "${BASH_SOURCE[0]}" -ef "$0" ]
then
    echo 'This script should not be executed directly, please run "deploy.sh" instead.'
    exit 1
fi

# Required ENVs
: ${RELEASE_ID?"Missing RELEASE_ID"}
: ${KEBOOLA_STACK?"Missing KEBOOLA_STACK"}
: ${HOSTNAME_SUFFIX?"Missing HOSTNAME_SUFFIX"}
: ${APP_PROXY_REPOSITORY?"Missing APP_PROXY_REPOSITORY"}
: ${APP_PROXY_IMAGE_TAG?"Missing APP_PROXY_IMAGE_TAG"}
: ${APP_PROXY_REPLICAS?"Missing APP_PROXY_REPLICAS"}

# Constants
export NAMESPACE="app-proxy"

# Common part of the deployment. Same for AWS/Azure/Local
./kubernetes/build.sh

# Namespace
kubectl apply -f ./kubernetes/deploy/namespace.yaml

# Proxy
kubectl apply -f ./kubernetes/deploy/proxy/config-map.yaml
kubectl apply -f ./kubernetes/deploy/proxy/pdb.yaml
kubectl apply -f ./kubernetes/deploy/proxy/network-policy.yaml
kubectl apply -f ./kubernetes/deploy/proxy/deployment.yaml

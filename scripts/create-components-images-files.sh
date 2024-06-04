#!/bin/bash
set -e

cd $(dirname $0)

source version
source export-config
source package-env

cd ../package

../scripts/k3s-images.sh

# Make sure the used data.json is a release artifact
cp ../bin/data.json ../bin/rancher-data.json

if [ ${ARCH} == arm64 ]; then
    ETCD_UNSUPPORTED_ARCH=arm64
fi
if [ ${ARCH} == s390x ]; then
    ETCD_UNSUPPORTED_ARCH=s390x
fi

mkdir -p ../dist
echo ${IMAGE} > ../dist/images
echo ${AGENT_IMAGE} >> ../dist/images
echo Built ${IMAGE} #${AGENT_IMAGE}
echo

cd ../bin

if [ ! -d build/system-charts ]; then
    mkdir -p build
    git clone --depth=1 --no-tags --branch $SYSTEM_CHART_DEFAULT_BRANCH https://github.com/rancher/system-charts $SYSTEM_CHART_REPO_DIR
fi

if [ ! -d $CHART_REPO_DIR ]; then
    git clone --branch $CHART_DEFAULT_BRANCH https://github.com/rancher/charts $CHART_REPO_DIR
fi

if [ ! -d $SMALL_FORK_REPO_DIR ]; then
    mkdir -p $SMALL_FORK_REPO_DIR
    git clone --branch main https://github.com/rancher/charts-small-fork $SMALL_FORK_REPO_DIR
fi

if [ ${ARCH} == amd64 ]; then
    # Move this out of ARCH check for local dev on non-amd64 hardware.
    TAG=$TAG REPO=${REPO} go run ../pkg/image/export/main.go $SYSTEM_CHART_REPO_DIR $CHART_REPO_DIR $IMAGE $AGENT_IMAGE $SYSTEM_AGENT_UPGRADE_IMAGE $WINS_AGENT_UPGRADE_IMAGE ${SYSTEM_AGENT_INSTALLER_RKE2_IMAGES[@]} ${SYSTEM_AGENT_INSTALLER_K3S_IMAGES[@]}
fi

# Create components file used for pre-release notes
../scripts/create-components-file.sh
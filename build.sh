#!/bin/bash

build_push(){
  nerdctl build --platform=${ARCHS} --output type=image,name=${REGISTRY}/${NAME}:latest,push=true .
}

helm_build_push(){
  FN=${NAME}-${VER}.tgz
  helm package ./install --version ${VER}
  curl --data-binary "@${FN}" http://helm.solenopsys.org/api/charts
 # rm ${FN}
}

REGISTRY=registry.solenopsys.org
NAME=solenopsys/ms-ipfs-revproxy
ARCHS="linux/amd64,linux/arm64"
VER=0.1.17


build_push
#helm_build_push
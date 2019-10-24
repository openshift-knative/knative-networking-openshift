#!/usr/bin/env bash

GOOS=linux GOARCH=amd64 CGOENABLED=0 go build -o build/_output/bin/knative-networking-openshift ./cmd/networking/openshift
docker build -f build/Dockerfile -t "$1" .
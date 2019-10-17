#!/usr/bin/env bash

GOOS=linux GOARCH=amd64 CGOENABLED=0 go build -o build/_output/bin/knative-networking-openshift ./cmd/networking/openshift
operator-sdk build "$1"
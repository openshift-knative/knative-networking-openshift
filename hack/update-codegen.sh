#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# Generate route injection code.
OUTPUT_PKG="github.com/openshift-knative/knative-serving-networking-openshift/pkg/client/injection/openshift" \
VERSIONED_CLIENTSET_PKG="github.com/openshift/client-go/route/clientset/versioned" \
EXTERNAL_INFORMER_PKG="github.com/openshift/client-go/route/informers/externalversions" \
  vendor/knative.dev/pkg/hack/generate-knative.sh "injection" \
    github.com/openshift/client-go \
    github.com/openshift/api \
    "route:v1" \
    --go-header-file hack/boilerplate.txt

# Generate maistra clients.
vendor/k8s.io/code-generator/generate-groups.sh "client,informer,lister" \
  github.com/openshift-knative/knative-serving-networking-openshift/pkg/client \
  github.com/maistra/istio-operator/pkg/apis \
  "maistra:v1" \
  --go-header-file hack/boilerplate.txt

# Generate maistra injection code.
OUTPUT_PKG="github.com/openshift-knative/knative-serving-networking-openshift/pkg/client/injection/maistra" \
vendor/knative.dev/pkg/hack/generate-knative.sh "injection" \
  github.com/openshift-knative/knative-serving-networking-openshift/pkg/client \
  github.com/maistra/istio-operator/pkg/apis \
  "maistra:v1" \
  --go-header-file hack/boilerplate.txt

hack/update-deps.sh
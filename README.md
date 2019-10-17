# Knative Networking Openshift

This is an implementation of Knative's `Ingress` resource, specific to Openshift needs. This has been "forked" from upstream's `networking-istio` code and as such, the `pkg/reconciler/ingress` package is a nearly identical copy of upstream's code. The goal is to keep this code up-to-speed with upstream advancements and enhance it where necessary to accomodate for Openshift's needs.

## Building and releasing a new image

To build a new image, use the `hack/build_image.sh` script. It wraps `go build` and `operator-sdk build` in a way that makes it look like an image build via operator-sdk. Push the image via `docker push` to quay.io to "release" it.

```bash
$ ./hack/build-image quay.io/openshift-knative/knative-networking-openshift:v0.10.0-1.3.0
$ docker push quay.io/openshift-knative/knative-networking-openshift:v0.10.0-1.3.0
```
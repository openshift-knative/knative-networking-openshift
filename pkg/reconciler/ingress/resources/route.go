package resources

import (
	"errors"
	"fmt"
	"strings"
	"time"

	routev1 "github.com/openshift/api/route/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"knative.dev/pkg/network"
	defaults "knative.dev/serving/pkg/apis/config"
	"knative.dev/serving/pkg/apis/networking"
	networkingv1alpha1 "knative.dev/serving/pkg/apis/networking/v1alpha1"
	presources "knative.dev/serving/pkg/resources"
)

const (
	TimeoutAnnotation      = "haproxy.router.openshift.io/timeout"
	DisableRouteAnnotation = "serving.knative.openshift.io/disableRoute"
	TerminationAnnotation  = "serving.knative.openshift.io/tlsMode"

	// TLSTerminationAnnotation is an annotation to configure routes.spec.tls.termination
	TLSTerminationAnnotation = "serving.knative.openshift.io/tlsTermination"
)

var (
	// ErrNotSupportedTLSTermination is an error when unsupported TLS termination is configured via annotation.
	ErrNotSupportedTLSTermination = errors.New("not supported tls termination is specified, only 'passthrough' is valid")

	// ErrNoValidLoadbalancerDomain indicates that the current ingress does not have a DomainInternal field, or
	// said field does not contain a value we can work with.
	ErrNoValidLoadbalancerDomain = errors.New("no parseable internal domain for ingresses found")
)

// MakeRoutes creates OpenShift Routes from a Knative Ingress
func MakeRoutes(ing networkingv1alpha1.IngressAccessor) ([]*routev1.Route, error) {
	// Skip making routes when the annotation is specified.
	if _, ok := ing.GetAnnotations()[DisableRouteAnnotation]; ok {
		return nil, nil
	}

	// Skip purely local ingresses.
	if ing.GetSpec().Visibility == networkingv1alpha1.IngressVisibilityClusterLocal {
		return nil, nil
	}

	service, err := findParseableInternalDomain(ing)
	if err != nil {
		return nil, err
	}

	var routes []*routev1.Route
	var index int
	for _, rule := range ing.GetSpec().Rules {
		// Skip generating routes for cluster-local rules.
		if rule.Visibility == networkingv1alpha1.IngressVisibilityClusterLocal {
			index += len(rule.Hosts)
			continue
		}

		timeout := defaults.DefaultMaxRevisionTimeoutSeconds * time.Second
		// We don't support multiple paths so just pick the first one here.
		if rule.HTTP != nil && len(rule.HTTP.Paths) > 0 && rule.HTTP.Paths[0].Timeout != nil {
			timeout = rule.HTTP.Paths[0].Timeout.Duration
		}

		for _, host := range rule.Hosts {
			// Ignore cluster-local domains.
			if strings.HasSuffix(host, network.GetClusterDomainName()) {
				index += 1
				continue
			}
			route, err := makeRoute(ing, index, host, service, timeout)
			index += 1
			if err != nil {
				return nil, err
			}
			routes = append(routes, route)
		}
	}

	return routes, nil
}

func findParseableInternalDomain(ing networkingv1alpha1.IngressAccessor) (types.NamespacedName, error) {
	loadbalancer := ing.GetStatus().LoadBalancer
	if loadbalancer == nil {
		return types.NamespacedName{}, ErrNoValidLoadbalancerDomain
	}
	for _, ingress := range loadbalancer.Ingress {
		if ingress.DomainInternal == "" {
			continue
		}
		fqn, err := parseInternalDomainToService(ingress.DomainInternal)
		if err != nil {
			continue
		}
		return fqn, nil
	}
	return types.NamespacedName{}, ErrNoValidLoadbalancerDomain
}

func parseInternalDomainToService(domainInternal string) (types.NamespacedName, error) {
	parts := strings.Split(domainInternal, ".")
	if len(parts) < 3 || parts[2] != "svc" {
		return types.NamespacedName{}, fmt.Errorf("could not extract namespace/name from %s", domainInternal)
	}
	return types.NamespacedName{
		Namespace: parts[1],
		Name:      parts[0],
	}, nil
}

func makeRoute(ing networkingv1alpha1.IngressAccessor,
	index int, host string, svc types.NamespacedName, timeout time.Duration) (*routev1.Route, error) {

	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("route-%s-%d", ing.GetUID(), index),
			Namespace: svc.Namespace,
			Labels: presources.UnionMaps(ing.GetLabels(), map[string]string{
				networking.IngressLabelKey: string(ing.GetUID()),
			}),
			Annotations: presources.UnionMaps(ing.GetAnnotations(), map[string]string{
				TimeoutAnnotation: fmt.Sprintf("%ds", int(timeout.Seconds())),
			}),
		},
		Spec: routev1.RouteSpec{
			Host: host,
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString("http2"),
			},
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: svc.Name,
			},
		},
	}

	if terminationType, ok := ing.GetAnnotations()[TLSTerminationAnnotation]; ok {
		switch strings.ToLower(terminationType) {
		case "passthrough":
			route.Spec.TLS = &routev1.TLSConfig{Termination: routev1.TLSTerminationPassthrough}
			route.Spec.Port = &routev1.RoutePort{TargetPort: intstr.FromString("https")}
		default:
			return nil, ErrNotSupportedTLSTermination
		}
	}

	return route, nil
}

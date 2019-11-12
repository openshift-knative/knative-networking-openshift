package resources

import (
	"crypto/sha256"
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
	// ErrNoValidLoadbalancerDomain indicates that the current ingress does not have a DomainInternal field, or
	// said field does not contain a value we can work with.
	ErrNoValidLoadbalancerDomain = errors.New("no parseable internal domain for ingresses found")
)

// MakeRoutes creates OpenShift Routes from a Knative Ingress
func MakeRoutes(ing networkingv1alpha1.IngressAccessor, lbs []networkingv1alpha1.LoadBalancerIngressStatus) ([]*routev1.Route, error) {
	// Skip making routes when the annotation is specified.
	if _, ok := ing.GetAnnotations()[DisableRouteAnnotation]; ok {
		return nil, nil
	}

	// Skip purely local ingresses.
	if ing.GetSpec().Visibility == networkingv1alpha1.IngressVisibilityClusterLocal {
		return nil, nil
	}

	service, err := findParseableInternalDomain(lbs)
	if err != nil {
		return nil, err
	}

	var routes []*routev1.Route
	for _, rule := range ing.GetSpec().Rules {
		// Skip generating routes for cluster-local rules.
		if rule.Visibility == networkingv1alpha1.IngressVisibilityClusterLocal {
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
				continue
			}
			routes = append(routes, MakeRoute(ing, host, service, timeout))
		}
	}

	return routes, nil
}

func findParseableInternalDomain(lbs []networkingv1alpha1.LoadBalancerIngressStatus) (types.NamespacedName, error) {
	for _, ingress := range lbs {
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

func MakeRoute(ing networkingv1alpha1.IngressAccessor, host string, svc types.NamespacedName, timeout time.Duration) *routev1.Route {
	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      routeName(string(ing.GetUID()), host),
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
		}
	}

	return route
}

func routeName(uid, host string) string {
	return fmt.Sprintf("route-%s-%x", uid, hashHost(host))
}

func hashHost(host string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(host)))[0:6]
}

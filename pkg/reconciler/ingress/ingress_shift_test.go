package ingress

import (
	"context"
	"testing"
	"time"

	// Inject our fakes
	fakesmmrclient "github.com/openshift-knative/knative-serving-networking-openshift/pkg/client/maistra/injection/client/fake"
	fakerouteclient "github.com/openshift-knative/knative-serving-networking-openshift/pkg/client/openshift/injection/client/fake"
	_ "knative.dev/pkg/client/injection/informers/istio/v1alpha3/gateway/fake"
	_ "knative.dev/pkg/client/injection/informers/istio/v1alpha3/virtualservice/fake"
	_ "knative.dev/pkg/client/injection/kube/informers/core/v1/endpoints/fake"
	_ "knative.dev/pkg/client/injection/kube/informers/core/v1/pod/fake"
	_ "knative.dev/pkg/client/injection/kube/informers/core/v1/secret/fake"
	_ "knative.dev/pkg/client/injection/kube/informers/core/v1/service/fake"
	_ "knative.dev/serving/pkg/client/injection/informers/networking/v1alpha1/ingress/fake"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	clientgotesting "k8s.io/client-go/testing"

	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"

	_ "knative.dev/pkg/system/testing"
	"knative.dev/serving/pkg/apis/networking/v1alpha1"
	"knative.dev/serving/pkg/reconciler"
	"knative.dev/serving/pkg/reconciler/ingress/resources"

	. "knative.dev/pkg/reconciler/testing"
	. "knative.dev/serving/pkg/reconciler/testing/v1alpha1"

	_ "github.com/openshift-knative/knative-serving-networking-openshift/pkg/client/maistra/injection/informers/maistra/v1/servicemeshmemberroll/fake"
	_ "github.com/openshift-knative/knative-serving-networking-openshift/pkg/client/openshift/injection/informers/route/v1/route/fake"
	oresources "github.com/openshift-knative/knative-serving-networking-openshift/pkg/reconciler/ingress/resources"

	maistrav1 "github.com/maistra/istio-operator/pkg/apis/maistra/v1"
)

// TestReconcileShift is an additional tests of TestReconcile in ingress_test.go.
// It tests extra resources like SMMR, NetworkPolicy for networking-openshift.
func TestReconcileShift(t *testing.T) {
	table := TableTest{{
		Name: "create new NetworkPolicy",
		Key:  "test-ns/route-tests",
		Objects: []runtime.Object{
			ingressWithStatus("route-tests", 1234, ingressReady),
			resources.MakeMeshVirtualService(insertProbe(ingress("route-tests", 1234))),
			resources.MakeIngressVirtualService(insertProbe(ingress("route-tests", 1234)),
				makeGatewayMap([]string{"knative-testing/knative-test-gateway", "knative-testing/knative-ingress-gateway"}, nil)),
			route(ingress("route-tests", 1234), "domain.com"),
			smmr([]string{"test-ns"}),
		},
		WantCreates: []runtime.Object{
			oresources.MakeNetworkPolicyAllowAll("test-ns"),
		},
		WantPatches: []clientgotesting.PatchActionImpl{
			patchAddFinalizerAction("route-tests", routeFinalizer),
		},
	}, {
		Name: "reconcile with existing istio-mesh NetworkPolicy",
		Key:  "test-ns/route-tests",
		Objects: []runtime.Object{
			ingressWithStatus("route-tests", 1234, ingressReady),
			resources.MakeMeshVirtualService(insertProbe(ingress("route-tests", 1234))),
			resources.MakeIngressVirtualService(insertProbe(ingress("route-tests", 1234)),
				makeGatewayMap([]string{"knative-testing/knative-test-gateway", "knative-testing/knative-ingress-gateway"}, nil)),
			route(ingress("route-tests", 1234), "domain.com"),
			smmr([]string{"test-ns"}),
			&networkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "istio-mesh",
					Namespace: "test-ns",
					Labels:    map[string]string{"maistra.io/owner": "knative-serving-ingress"},
				},
				Spec: networkingv1.NetworkPolicySpec{},
			},
		},
		WantCreates: []runtime.Object{
			oresources.MakeNetworkPolicyAllowAll("test-ns"),
		},
		WantPatches: []clientgotesting.PatchActionImpl{
			patchAddFinalizerAction("route-tests", routeFinalizer),
		},
	}, {
		Name: "reconcile with existing managed and istio-mesh NetworkPolicies",
		Key:  "test-ns/route-tests",
		Objects: []runtime.Object{
			ingressWithStatus("route-tests", 1234, ingressReady),
			resources.MakeMeshVirtualService(insertProbe(ingress("route-tests", 1234))),
			resources.MakeIngressVirtualService(insertProbe(ingress("route-tests", 1234)),
				makeGatewayMap([]string{"knative-testing/knative-test-gateway", "knative-testing/knative-ingress-gateway"}, nil)),
			route(ingress("route-tests", 1234), "domain.com"),
			smmr([]string{"test-ns"}),
			&networkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "istio-mesh",
					Namespace: "test-ns",
					Labels:    map[string]string{"maistra.io/owner": "knative-serving-ingress"},
				},
				Spec: networkingv1.NetworkPolicySpec{},
			},
			oresources.MakeNetworkPolicyAllowAll("test-ns"),
		},
		WantPatches: []clientgotesting.PatchActionImpl{
			patchAddFinalizerAction("route-tests", routeFinalizer),
		},
	}, {
		Name: "reconcile with user-added NetworkPolicy",
		Key:  "test-ns/route-tests",
		Objects: []runtime.Object{
			ingressWithStatus("route-tests", 1234, ingressReady),
			resources.MakeMeshVirtualService(insertProbe(ingress("route-tests", 1234))),
			resources.MakeIngressVirtualService(insertProbe(ingress("route-tests", 1234)),
				makeGatewayMap([]string{"knative-testing/knative-test-gateway", "knative-testing/knative-ingress-gateway"}, nil)),
			route(ingress("route-tests", 1234), "domain.com"),
			smmr([]string{"test-ns"}),
			&networkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-network-policy",
					Namespace: "test-ns",
				},
				Spec: networkingv1.NetworkPolicySpec{},
			},
		},
		WantPatches: []clientgotesting.PatchActionImpl{
			patchAddFinalizerAction("route-tests", routeFinalizer),
		},
	}, {
		Name: "reconcile with existing managed and user-added NetworkPolicies",
		Key:  "test-ns/route-tests",
		Objects: []runtime.Object{
			ingressWithStatus("route-tests", 1234, ingressReady),
			resources.MakeMeshVirtualService(insertProbe(ingress("route-tests", 1234))),
			resources.MakeIngressVirtualService(insertProbe(ingress("route-tests", 1234)),
				makeGatewayMap([]string{"knative-testing/knative-test-gateway", "knative-testing/knative-ingress-gateway"}, nil)),
			route(ingress("route-tests", 1234), "domain.com"),
			smmr([]string{"test-ns"}),
			&networkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-network-policy",
					Namespace: "test-ns",
				},
				Spec: networkingv1.NetworkPolicySpec{},
			},
			oresources.MakeNetworkPolicyAllowAll("test-ns"),
		},
		WantPatches: []clientgotesting.PatchActionImpl{
			patchAddFinalizerAction("route-tests", routeFinalizer),
		},
	}, {
		Name: "reconcile deletion with existing managed NetworkPolicy",
		Key:  "test-ns/route-tests",
		Objects: []runtime.Object{
			addRouteFinalizer(addDeletionTimestamp(ingress("route-tests", 1234))),
			resources.MakeMeshVirtualService(insertProbe(ingress("route-tests", 1234))),
			resources.MakeIngressVirtualService(insertProbe(ingress("route-tests", 1234)),
				makeGatewayMap([]string{"knative-testing/knative-test-gateway", "knative-testing/knative-ingress-gateway"}, nil)),
			smmr([]string{"ns-a", "ns-b", "test-ns"}),
			oresources.MakeNetworkPolicyAllowAll("test-ns"),
		},
		WantUpdates: []clientgotesting.UpdateActionImpl{{
			Object: addDeletionTimestamp(ingress("route-tests", 1234)),
		}, {
			Object: smmr([]string{"ns-a", "ns-b"}),
		}},
		WantDeletes: []clientgotesting.DeleteActionImpl{{
			ActionImpl: clientgotesting.ActionImpl{
				Namespace: "test-ns",
				Verb:      "delete",
				Resource: schema.GroupVersionResource{
					Group:    "networking.k8s.io",
					Version:  "v1",
					Resource: "networkpolicies"},
				Subresource: ""},
			Name: "knative-serving-allow-all",
		}},
	}, {
		Name: "reconcile deletion with existing istio-mesh NetworkPolicy",
		Key:  "test-ns/route-tests",
		Objects: []runtime.Object{
			addRouteFinalizer(addDeletionTimestamp(ingress("route-tests", 1234))),
			resources.MakeMeshVirtualService(insertProbe(ingress("route-tests", 1234))),
			resources.MakeIngressVirtualService(insertProbe(ingress("route-tests", 1234)),
				makeGatewayMap([]string{"knative-testing/knative-test-gateway", "knative-testing/knative-ingress-gateway"}, nil)),
			smmr([]string{"another-ns", "test-ns"}),
			oresources.MakeNetworkPolicyAllowAll("test-ns"),
			&networkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "istio-mesh",
					Namespace: "test-ns",
					Labels:    map[string]string{"maistra.io/owner": "knative-serving-ingress"},
				},
				Spec: networkingv1.NetworkPolicySpec{},
			},
		},
		WantUpdates: []clientgotesting.UpdateActionImpl{{
			Object: addDeletionTimestamp(ingress("route-tests", 1234)),
		}, {
			Object: smmr([]string{"another-ns"}),
		}},
		WantDeletes: []clientgotesting.DeleteActionImpl{{
			ActionImpl: clientgotesting.ActionImpl{
				Namespace: "test-ns",
				Verb:      "delete",
				Resource: schema.GroupVersionResource{
					Group:    "networking.k8s.io",
					Version:  "v1",
					Resource: "networkpolicies"},
				Subresource: ""},
			Name: "knative-serving-allow-all",
		}},
	}, {
		Name: "reconcile deletion with existing managed and istio-mesh NetworkPolicy",
		Key:  "test-ns/route-tests",
		Objects: []runtime.Object{
			addRouteFinalizer(addDeletionTimestamp(ingress("route-tests", 1234))),
			resources.MakeMeshVirtualService(insertProbe(ingress("route-tests", 1234))),
			resources.MakeIngressVirtualService(insertProbe(ingress("route-tests", 1234)),
				makeGatewayMap([]string{"knative-testing/knative-test-gateway", "knative-testing/knative-ingress-gateway"}, nil)),
			smmr([]string{"another-ns", "test-ns"}),
			oresources.MakeNetworkPolicyAllowAll("test-ns"),
			&networkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "istio-mesh",
					Namespace: "test-ns",
					Labels:    map[string]string{"maistra.io/owner": "knative-serving-ingress"},
				},
				Spec: networkingv1.NetworkPolicySpec{},
			},
		},
		WantUpdates: []clientgotesting.UpdateActionImpl{{
			Object: addDeletionTimestamp(ingress("route-tests", 1234)),
		}, {
			Object: smmr([]string{"another-ns"}),
		}},
		WantDeletes: []clientgotesting.DeleteActionImpl{{
			ActionImpl: clientgotesting.ActionImpl{
				Namespace: "test-ns",
				Verb:      "delete",
				Resource: schema.GroupVersionResource{
					Group:    "networking.k8s.io",
					Version:  "v1",
					Resource: "networkpolicies"},
				Subresource: ""},
			Name: "knative-serving-allow-all",
		}},
	}, {
		Name: "reconcile deletion with user-added NetworkPolicy",
		Key:  "test-ns/route-tests",
		Objects: []runtime.Object{
			addRouteFinalizer(addDeletionTimestamp(ingress("route-tests", 1234))),
			resources.MakeMeshVirtualService(insertProbe(ingress("route-tests", 1234))),
			resources.MakeIngressVirtualService(insertProbe(ingress("route-tests", 1234)),
				makeGatewayMap([]string{"knative-testing/knative-test-gateway", "knative-testing/knative-ingress-gateway"}, nil)),
			smmr([]string{"another-ns", "test-ns"}),
			&networkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-network-policy",
					Namespace: "test-ns",
				},
				Spec: networkingv1.NetworkPolicySpec{},
			},
		},
		WantUpdates: []clientgotesting.UpdateActionImpl{{
			Object: addDeletionTimestamp(ingress("route-tests", 1234)),
		}, {
			Object: smmr([]string{"another-ns"}),
		}},
	}, {
		Name: "reconcile deletion with existing managed and user-added NetworkPolicy",
		Key:  "test-ns/route-tests",
		Objects: []runtime.Object{
			addRouteFinalizer(addDeletionTimestamp(ingress("route-tests", 1234))),
			resources.MakeMeshVirtualService(insertProbe(ingress("route-tests", 1234))),
			resources.MakeIngressVirtualService(insertProbe(ingress("route-tests", 1234)),
				makeGatewayMap([]string{"knative-testing/knative-test-gateway", "knative-testing/knative-ingress-gateway"}, nil)),
			smmr([]string{"another-ns", "test-ns"}),
			oresources.MakeNetworkPolicyAllowAll("test-ns"),
			&networkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-network-policy",
					Namespace: "test-ns",
				},
				Spec: networkingv1.NetworkPolicySpec{},
			},
		},
		WantUpdates: []clientgotesting.UpdateActionImpl{{
			Object: addDeletionTimestamp(ingress("route-tests", 1234)),
		}, {
			Object: smmr([]string{"another-ns"}),
		}},
	}}

	table.Test(t, MakeFactory(func(ctx context.Context, listers *Listers, cmw configmap.Watcher) controller.Reconciler {
		return &Reconciler{
			Base:                 reconciler.NewBase(ctx, controllerAgentName, cmw),
			virtualServiceLister: listers.GetVirtualServiceLister(),
			gatewayLister:        listers.GetGatewayLister(),
			routeLister:          listers.GetOpenshiftRouteLister(),
			routeClient:          fakerouteclient.Get(ctx),
			smmrLister:           listers.GetServiceMeshMemberRollLister(),
			smmrClient:           fakesmmrclient.Get(ctx),
			finalizer:            ingressFinalizer,
			rfinalizer:           routeFinalizer,
			configStore: &testConfigStore{
				config: ReconcilerTestConfig(),
			},
			statusManager: &fakeStatusManager{
				FakeIsReady: func(ia *v1alpha1.Ingress, gw map[v1alpha1.IngressVisibility]sets.String) (bool, error) {
					return true, nil
				},
			},
			ingressLister: listers.GetIngressLister(),
		}
	}))
}

func addDeletionTimestamp(ing *v1alpha1.Ingress) *v1alpha1.Ingress {
	t := metav1.NewTime(time.Unix(1e9, 0))
	ing.SetDeletionTimestamp(&t)
	return ing
}

func addRouteFinalizer(ingress *v1alpha1.Ingress) *v1alpha1.Ingress {
	ingress.ObjectMeta.Finalizers = []string{routeFinalizer}
	return ingress
}

func smmr(namespaces []string) *maistrav1.ServiceMeshMemberRoll {
	smmr := maistrav1.ServiceMeshMemberRoll{
		ObjectMeta: metav1.ObjectMeta{
			Name:      smmrName,
			Namespace: smmrNamespace,
		},
		Spec: maistrav1.ServiceMeshMemberRollSpec{
			Members: namespaces,
		},
	}
	return &smmr
}

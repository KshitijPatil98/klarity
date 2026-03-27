package controller_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1alpha1 "github.com/KshitijPatil98/klarity/api/v1alpha1"
	"github.com/KshitijPatil98/klarity/internal/controller"
)

const (
	testConfigName      = "klarity"
	testOperatorNS      = "klarity-system"
	testAnthropicHeader = "2023-06-01"
)

func newTestReconciler(t *testing.T, httpClient *http.Client, objects ...client.Object) *controller.KlarityConfigReconciler {
	t.Helper()

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))

	builder := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&v1alpha1.KlarityConfig{})
	if len(objects) > 0 {
		builder = builder.WithObjects(objects...)
	}

	return &controller.KlarityConfigReconciler{
		Client:     builder.Build(),
		HTTPClient: httpClient,
	}
}

func configRequest() reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{Name: testConfigName},
	}
}

func newKlarityConfig(secretName, secretKey string) *v1alpha1.KlarityConfig {
	return &v1alpha1.KlarityConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: testConfigName,
		},
		Spec: v1alpha1.KlarityConfigSpec{
			AI: v1alpha1.AIConfig{
				Provider: "anthropic",
				Model:    "claude-opus-4-6",
				APIKeySecretRef: v1alpha1.SecretKeyRef{
					Name: secretName,
					Key:  secretKey,
				},
			},
		},
	}
}

func newSecret(name string, data map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testOperatorNS,
		},
		Data: data,
	}
}

func newMonitor(namespace, name string, enabled bool) *v1alpha1.KlarityMonitor {
	return &v1alpha1.KlarityMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.KlarityMonitorSpec{
			FailureTypes: []string{"OOMKill"},
			Enabled:      enabled,
		},
	}
}

func getConfig(t *testing.T, c client.Client) v1alpha1.KlarityConfig {
	t.Helper()
	var cfg v1alpha1.KlarityConfig
	if err := c.Get(context.Background(), types.NamespacedName{Name: testConfigName}, &cfg); err != nil {
		t.Fatalf("failed to get KlarityConfig: %v", err)
	}
	return cfg
}

func anthropicHTTPClientForServer(t *testing.T, server *httptest.Server, timeout time.Duration, observe func(*http.Request)) *http.Client {
	t.Helper()

	targetURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("failed to parse test server URL: %v", err)
	}

	baseTransport := server.Client().Transport
	if baseTransport == nil {
		baseTransport = http.DefaultTransport
	}

	return &http.Client{
		Timeout: timeout,
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if observe != nil {
				observe(req)
			}
			cloned := req.Clone(req.Context())
			cloned.URL.Scheme = targetURL.Scheme
			cloned.URL.Host = targetURL.Host
			cloned.Host = targetURL.Host
			return baseTransport.RoundTrip(cloned)
		}),
	}
}

func unexpectedHTTPClient(t *testing.T) *http.Client {
	t.Helper()

	return &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			t.Fatalf("unexpected HTTP request")
			return nil, errors.New("unexpected HTTP request")
		}),
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

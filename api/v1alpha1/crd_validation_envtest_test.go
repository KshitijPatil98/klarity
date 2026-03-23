package v1alpha1_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	klarityv1alpha1 "github.com/KshitijPatil98/klarity/api/v1alpha1"
)

var (
	testEnv         *envtest.Environment
	k8sClient       ctrlclient.Client
	envtestStartErr error
)

func TestMain(m *testing.M) {
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	if err != nil {
		envtestStartErr = err
		if os.Getenv("CI") != "" {
			panic(fmt.Sprintf("failed to start envtest in CI: %v", err))
		}
		code := m.Run()
		os.Exit(code)
	}

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(klarityv1alpha1.AddToScheme(scheme))

	k8sClient, err = ctrlclient.New(cfg, ctrlclient.Options{Scheme: scheme})
	if err != nil {
		panic(fmt.Sprintf("failed to create client: %v", err))
	}

	code := m.Run()

	if err := testEnv.Stop(); err != nil {
		panic(fmt.Sprintf("failed to stop envtest: %v", err))
	}

	os.Exit(code)
}

func TestKlarityConfigCRDValidation(t *testing.T) {
	if envtestStartErr != nil {
		t.Skipf("skipping envtest CRD validation test: %v", envtestStartErr)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("rejects missing spec", func(t *testing.T) {
		obj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "klarity.io/v1alpha1",
				"kind":       "KlarityConfig",
				"metadata": map[string]interface{}{
					"name": "klarity",
				},
			},
		}

		err := k8sClient.Create(ctx, obj)
		if err == nil {
			_ = k8sClient.Delete(context.Background(), obj)
			t.Fatalf("expected validation error for missing spec, got nil")
		}
		if !apierrors.IsInvalid(err) {
			t.Fatalf("expected invalid error, got: %v", err)
		}
		if !strings.Contains(err.Error(), "spec: Required value") {
			t.Fatalf("expected missing spec error, got: %v", err)
		}
	})

	t.Run("rejects empty critical ai fields", func(t *testing.T) {
		obj := &klarityv1alpha1.KlarityConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: "klarity",
			},
			Spec: klarityv1alpha1.KlarityConfigSpec{
				AI: klarityv1alpha1.AIConfig{
					Provider: "anthropic",
					Model:    "",
					APIKeySecretRef: klarityv1alpha1.SecretKeyRef{
						Name: "",
						Key:  "",
					},
				},
			},
		}

		err := k8sClient.Create(ctx, obj)
		if err == nil {
			_ = k8sClient.Delete(context.Background(), obj)
			t.Fatalf("expected validation error for empty AI fields, got nil")
		}
		if !apierrors.IsInvalid(err) {
			t.Fatalf("expected invalid error, got: %v", err)
		}
	})

	t.Run("accepts valid config", func(t *testing.T) {
		obj := &klarityv1alpha1.KlarityConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: "klarity",
			},
			Spec: klarityv1alpha1.KlarityConfigSpec{
				AI: klarityv1alpha1.AIConfig{
					Provider: "anthropic",
					Model:    "claude-opus-4-6",
					APIKeySecretRef: klarityv1alpha1.SecretKeyRef{
						Name: "klarity-secrets",
						Key:  "anthropic-api-key",
					},
				},
			},
		}

		if err := k8sClient.Create(ctx, obj); err != nil {
			t.Fatalf("expected valid KlarityConfig, got error: %v", err)
		}
		t.Cleanup(func() {
			_ = k8sClient.Delete(context.Background(), obj)
		})
	})
}

func TestKlarityMonitorCRDValidation(t *testing.T) {
	if envtestStartErr != nil {
		t.Skipf("skipping envtest CRD validation test: %v", envtestStartErr)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("monitor-validation-%d", time.Now().UnixNano()),
		},
	}
	if err := k8sClient.Create(ctx, namespace); err != nil {
		t.Fatalf("failed creating test namespace: %v", err)
	}
	t.Cleanup(func() {
		_ = k8sClient.Delete(context.Background(), namespace)
	})

	t.Run("rejects missing spec", func(t *testing.T) {
		obj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "klarity.io/v1alpha1",
				"kind":       "KlarityMonitor",
				"metadata": map[string]interface{}{
					"name":      "missing-spec",
					"namespace": namespace.Name,
				},
			},
		}

		err := k8sClient.Create(ctx, obj)
		if err == nil {
			_ = k8sClient.Delete(context.Background(), obj)
			t.Fatalf("expected validation error for missing spec, got nil")
		}
		if !apierrors.IsInvalid(err) {
			t.Fatalf("expected invalid error, got: %v", err)
		}
		if !strings.Contains(err.Error(), "spec: Required value") {
			t.Fatalf("expected missing spec error, got: %v", err)
		}
	})

	t.Run("rejects empty failure type item", func(t *testing.T) {
		obj := &klarityv1alpha1.KlarityMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "empty-failure-type",
				Namespace: namespace.Name,
			},
			Spec: klarityv1alpha1.KlarityMonitorSpec{
				FailureTypes: []string{""},
			},
		}

		err := k8sClient.Create(ctx, obj)
		if err == nil {
			_ = k8sClient.Delete(context.Background(), obj)
			t.Fatalf("expected validation error for empty failure type, got nil")
		}
		if !apierrors.IsInvalid(err) {
			t.Fatalf("expected invalid error, got: %v", err)
		}
	})

	t.Run("accepts valid monitor", func(t *testing.T) {
		obj := &klarityv1alpha1.KlarityMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "valid-monitor",
				Namespace: namespace.Name,
			},
			Spec: klarityv1alpha1.KlarityMonitorSpec{
				FailureTypes: []string{"OOMKill"},
			},
		}

		if err := k8sClient.Create(ctx, obj); err != nil {
			t.Fatalf("expected valid KlarityMonitor, got error: %v", err)
		}
		t.Cleanup(func() {
			_ = k8sClient.Delete(context.Background(), obj)
		})
	})
}

//go:build envtest
// +build envtest

package v1alpha1_test

import (
	"context"
	"strings"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	klarityv1alpha1 "github.com/KshitijPatil98/klarity/api/v1alpha1"
)

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

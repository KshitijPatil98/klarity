package v1alpha1_test

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	klarityv1alpha1 "github.com/KshitijPatil98/klarity/api/v1alpha1"
)

func TestKlarityDiagnosisDeepCopyFullyPopulated(t *testing.T) {
	original := &klarityv1alpha1.KlarityDiagnosis{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deepcopy-test",
			Namespace: "apps",
			Labels: map[string]string{
				"team": "payments",
			},
		},
		Spec: klarityv1alpha1.KlarityDiagnosisSpec{
			FailureType:   "OOMKill",
			PodName:       "payments-5f4d8bf47d-2jp4s",
			ContainerName: "app",
			Namespace:     "apps",
			NodeName:      "ip-10-0-0-30",
			OwnerRef: klarityv1alpha1.OwnerRef{
				Kind: "Deployment",
				Name: "payments",
			},
			RevisionHash: "5f4d8bf47d",
			MonitorRef: klarityv1alpha1.MonitorRef{
				Name:      "payments-monitor",
				Namespace: "apps",
			},
			DetectedAt: time.Now().UTC().Format(time.RFC3339),
			Context: klarityv1alpha1.DiagnosisContext{
				RestartCount: 3,
				ExitCode:     137,
				Resources: &klarityv1alpha1.ResourceValues{
					Requests: map[string]string{
						"cpu":    "250m",
						"memory": "256Mi",
					},
					Limits: map[string]string{
						"cpu":    "500m",
						"memory": "512Mi",
					},
				},
				Sources: []klarityv1alpha1.ContextSource{
					{Name: "logs", Data: "oom"},
					{Name: "events", Data: "killed"},
				},
			},
		},
		Status: klarityv1alpha1.KlarityDiagnosisStatus{
			Phase: "Diagnosed",
			Diagnosis: &klarityv1alpha1.DiagnosisResult{
				Summary:    "container exceeded memory limit",
				RootCause:  "memory leak in worker",
				Category:   "application",
				Confidence: 0.92,
				Recommendations: []klarityv1alpha1.Recommendation{
					{
						Action:   "increase memory limit",
						Type:     "resource",
						Priority: "immediate",
					},
				},
				AffectedResources: []klarityv1alpha1.AffectedResource{
					{
						Kind:      "Deployment",
						Name:      "payments",
						Namespace: "apps",
					},
				},
			},
		},
	}

	cloned := original.DeepCopy()
	if cloned == nil {
		t.Fatalf("DeepCopy returned nil")
	}

	cloned.Labels["team"] = "platform"
	cloned.Spec.Context.Resources.Requests["memory"] = "2Gi"
	cloned.Spec.Context.Sources[0].Data = "changed-data"
	cloned.Status.Diagnosis.Recommendations[0].Action = "changed-action"
	cloned.Status.Diagnosis.AffectedResources[0].Name = "changed-name"

	if original.Labels["team"] != "payments" {
		t.Fatalf("expected original labels unchanged, got: %q", original.Labels["team"])
	}
	if original.Spec.Context.Resources.Requests["memory"] != "256Mi" {
		t.Fatalf("expected original resources.requests unchanged, got: %q", original.Spec.Context.Resources.Requests["memory"])
	}
	if original.Spec.Context.Sources[0].Data != "oom" {
		t.Fatalf("expected original context sources unchanged, got: %q", original.Spec.Context.Sources[0].Data)
	}
	if original.Status.Diagnosis.Recommendations[0].Action != "increase memory limit" {
		t.Fatalf("expected original recommendations unchanged, got: %q", original.Status.Diagnosis.Recommendations[0].Action)
	}
	if original.Status.Diagnosis.AffectedResources[0].Name != "payments" {
		t.Fatalf("expected original affectedResources unchanged, got: %q", original.Status.Diagnosis.AffectedResources[0].Name)
	}
}

func TestSchemeRegistrationIncludesAllKlarityTypes(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := klarityv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to register scheme: %v", err)
	}

	kinds := []string{
		"KlarityConfig",
		"KlarityConfigList",
		"KlarityMonitor",
		"KlarityMonitorList",
		"KlarityDiagnosis",
		"KlarityDiagnosisList",
	}

	for _, kind := range kinds {
		if _, err := scheme.New(klarityv1alpha1.GroupVersion.WithKind(kind)); err != nil {
			t.Fatalf("kind %s is not registered in scheme: %v", kind, err)
		}
	}
}

//go:build !envtest
// +build !envtest

package v1alpha1_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKlarityConfigCRDSchemaDoesNotContainGlobalCooldown(t *testing.T) {
	crdPath := filepath.Join("..", "..", "..", "config", "crd", "bases", "klarity.io_klarityconfigs.yaml")
	data, err := os.ReadFile(crdPath)
	if err != nil {
		t.Fatalf("failed reading KlarityConfig CRD: %v", err)
	}

	if strings.Contains(string(data), "globalCooldown") {
		t.Fatalf("KlarityConfig CRD schema still contains removed field globalCooldown")
	}
}

func TestKlarityMonitorCRDSchemaDoesNotContainCooldownOverride(t *testing.T) {
	crdPath := filepath.Join("..", "..", "..", "config", "crd", "bases", "klarity.io_klaritymonitors.yaml")
	data, err := os.ReadFile(crdPath)
	if err != nil {
		t.Fatalf("failed reading KlarityMonitor CRD: %v", err)
	}

	if strings.Contains(string(data), "cooldownOverride") {
		t.Fatalf("KlarityMonitor CRD schema still contains removed field cooldownOverride")
	}
}

#!/usr/bin/env bash
set -euo pipefail

# Generate deepcopy functions → api/v1alpha1/zz_generated.deepcopy.go
controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./api/..."

# Generate CRD YAML → config/crd/bases/
controller-gen crd:allowDangerousTypes=true paths="./api/..." output:crd:artifacts:config=config/crd/bases

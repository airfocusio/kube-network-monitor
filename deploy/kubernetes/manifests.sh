#!/bin/bash
set -euo pipefail

rm -rf .tmp/manifests
mkdir -p .tmp/manifests
cat > .tmp/manifests/kustomization.yaml << EOF
resources:
- ../../deploy/kubernetes
images:
- name: 'ghcr.io/airfocusio/kube-network-monitor'
  newTag: '${1}'
EOF
kustomize build .tmp/manifests > .tmp/manifests/manifests.yaml

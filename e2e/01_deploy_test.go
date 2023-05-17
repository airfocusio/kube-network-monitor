package e2e

import (
	"fmt"
	"runtime"
	"testing"
)

func TestDeploy(t *testing.T) {
	if err := write(".tmp/e2e/deploy", map[string]string{
		"kustomization.yaml": fmt.Sprintf(`resources:
- ../../../deploy/kubernetes
images:
- name: ghcr.io/airfocusio/kube-network-monitor
  newTag: 0.0.0-dev-%s
patchesStrategicMerge:
- prometheusrule.yaml
- servicemonitor.yaml
`, runtime.GOARCH),
		"prometheusrule.yaml": `$patch: delete
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: kube-network-monitor
  namespace: kube-system
`,
		"servicemonitor.yaml": `$patch: delete
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: kube-network-monitor
  namespace: kube-system
`,
	}); err != nil {
		t.Fatal(err)
	}
	if err := cmd(
		kubectlBin,
		[]string{"--context", "kind-kube-network-monitor", "apply", "-k", ".tmp/e2e/deploy"},
		cmdOpts{},
	); err != nil {
		t.Fatal(err)
	}

	if err := cmd(
		kubectlBin,
		[]string{"--context", "kind-kube-network-monitor", "wait", "-n", "kube-system", "--for", "condition=ready", "pod", "-l", "app=kube-network-monitor"},
		cmdOpts{},
	); err != nil {
		t.Fatal(err)
	}
}

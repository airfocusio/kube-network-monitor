package e2e

import (
	"fmt"
	"runtime"
	"testing"
	"time"
)

func TestDeploy(t *testing.T) {
	if _, err := cmd(
		goreleaserBin,
		[]string{"release", "--clean", "--skip-publish", "--snapshot"},
		cmdOpts{Timeout: 5 * time.Minute},
	); err != nil {
		panic(err)
	}
	if _, err := cmd(
		kindBin,
		[]string{"load", "docker-image", "--name", kindClusterName, fmt.Sprintf("ghcr.io/airfocusio/kube-network-monitor:0.0.0-dev-%s", runtime.GOARCH)},
		cmdOpts{},
	); err != nil {
		panic(err)
	}

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

	if _, err := cmd(
		kubectlBin,
		[]string{"--context", kindClusterKubectlContext, "-n", "kube-system", "delete", "daemonset", "-l", "app=kube-network-monitor", "--wait"},
		cmdOpts{},
	); err != nil {
		t.Fatal(err)
	}

	if _, err := cmd(
		kubectlBin,
		[]string{"--context", kindClusterKubectlContext, "-n", "kube-system", "delete", "pod", "-l", "app=kube-network-monitor", "--wait"},
		cmdOpts{},
	); err != nil {
		t.Fatal(err)
	}

	if _, err := cmd(
		kubectlBin,
		[]string{"--context", kindClusterKubectlContext, "apply", "-k", ".tmp/e2e/deploy"},
		cmdOpts{},
	); err != nil {
		t.Fatal(err)
	}

	if _, err := cmd(
		kubectlBin,
		[]string{"--context", kindClusterKubectlContext, "wait", "-n", "kube-system", "--for", "condition=ready", "pod", "-l", "app=kube-network-monitor"},
		cmdOpts{},
	); err != nil {
		t.Fatal(err)
	}
}

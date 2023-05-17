package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"testing"
	"time"
)

const (
	kindBin       = "kind"
	kubectlBin    = "kubectl"
	goreleaserBin = "goreleaser"
)

func TestMain(t *testing.M) {
	startCluster()
	exitCode := t.Run()
	stopCluster()
	os.Exit(exitCode)
}

func startCluster() {
	fmt.Printf("startCluster\n")
	if err := write(".tmp/e2e/kind", map[string]string{
		"config.yaml": `kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
- role: worker
`,
	}); err != nil {
		panic(err)
	}
	if err := cmd(
		kindBin,
		[]string{"create", "cluster", "--name", "kube-network-monitor", "--config", ".tmp/e2e/kind/config.yaml", "--image", "kindest/node:v1.26.4"},
		cmdOpts{Timeout: 5 * time.Minute},
	); err != nil {
		panic(err)
	}
	if err := cmd(
		goreleaserBin,
		[]string{"release", "--clean", "--skip-publish", "--snapshot"},
		cmdOpts{Timeout: 5 * time.Minute},
	); err != nil {
		panic(err)
	}
	if err := cmd(
		kindBin,
		[]string{"load", "docker-image", "--name", "kube-network-monitor", fmt.Sprintf("ghcr.io/airfocusio/kube-network-monitor:0.0.0-dev-%s", runtime.GOARCH)},
		cmdOpts{},
	); err != nil {
		panic(err)
	}
}

func stopCluster() {
	fmt.Printf("stopCluster\n")
	if err := cmd(
		kindBin,
		[]string{"delete", "cluster", "--name", "kube-network-monitor"},
		cmdOpts{},
	); err != nil {
		panic(err)
	}
}

type cmdOpts struct {
	Timeout time.Duration
}

func cmd(name string, arg []string, opts cmdOpts) error {
	fmt.Printf("cmd %s %v\n", name, arg)
	timeout := time.Minute
	if opts.Timeout != 0 {
		timeout = opts.Timeout
	}
	ctx, done := context.WithTimeout(context.Background(), timeout)
	defer done()
	cmd := exec.CommandContext(ctx, name, arg...)
	cmd.Dir = ".."
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func write(dir string, files map[string]string) error {
	fullDir := path.Join("..", dir)
	if err := os.RemoveAll(fullDir); err != nil {
		return err
	}
	if err := os.MkdirAll(fullDir, 0o755); err != nil {
		return err
	}
	for name, content := range files {
		if err := os.WriteFile(path.Join(fullDir, name), []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

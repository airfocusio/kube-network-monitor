package e2e

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"
	"time"

	"k8s.io/utils/strings/slices"
)

const (
	kindBin       = "kind"
	kubectlBin    = "kubectl"
	goreleaserBin = "goreleaser"
)

var (
	configDebug       = slices.Contains([]string{"1", "yes", "true"}, os.Getenv("E2E_DEBUG"))
	configKeepCluster = slices.Contains([]string{"1", "yes", "true"}, os.Getenv("E2E_KEEP_CLUSTER"))
)

var (
	kindClusterName           = "kube-network-monitor"
	kindClusterKubectlContext = "kind-" + kindClusterName
)

func TestMain(t *testing.M) {
	startCluster()
	exitCode := t.Run()
	stopCluster()
	os.Exit(exitCode)
}

func startCluster() {
	stopCluster()
	if configKeepCluster {
		result, err := cmd(
			kindBin,
			[]string{"get", "clusters"},
			cmdOpts{},
		)
		if err != nil {
			panic(err)
		}
		if slices.Contains(strings.Split(string(result), "\n"), kindClusterName) {
			return
		}
	}

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
	if _, err := cmd(
		kindBin,
		[]string{"create", "cluster", "--name", kindClusterName, "--config", ".tmp/e2e/kind/config.yaml", "--image", "kindest/node:v1.26.4"},
		cmdOpts{Timeout: 5 * time.Minute},
	); err != nil {
		panic(err)
	}
}

func stopCluster() {
	if configKeepCluster {
		return
	}

	fmt.Printf("stopCluster\n")
	if _, err := cmd(
		kindBin,
		[]string{"delete", "cluster", "--name", kindClusterName},
		cmdOpts{},
	); err != nil {
		panic(err)
	}
}

type cmdOpts struct {
	Timeout time.Duration
}

func cmd(name string, arg []string, opts cmdOpts) ([]byte, error) {
	fmt.Printf("cmd %s %v\n", name, arg)
	timeout := time.Minute
	if opts.Timeout != 0 {
		timeout = opts.Timeout
	}
	ctx, done := context.WithTimeout(context.Background(), timeout)
	defer done()
	cmd := exec.CommandContext(ctx, name, arg...)
	cmd.Dir = ".."

	var b bytes.Buffer
	if configDebug {
		cmd.Stdout = multiWriter{Writers: []io.Writer{&b, os.Stdout}}
		cmd.Stderr = multiWriter{Writers: []io.Writer{&b, os.Stderr}}
	} else {
		cmd.Stdout = &b
		cmd.Stderr = &b
	}

	err := cmd.Run()
	return b.Bytes(), err
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

type multiWriter struct {
	Writers []io.Writer
	Writer2 io.Writer
}

func (ms multiWriter) Write(p []byte) (n int, err error) {
	for _, w := range ms.Writers {
		_, err := w.Write(p)
		if err != nil {
			return 0, err
		}
	}
	return len(p), nil
}

package bootstrap

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/buildkite/agent/bootstrap/shell"
	"github.com/buildkite/bintest"
)

func init() {
	sshKeyscanRetryInterval = time.Millisecond
}

func TestFindingSSHTools(t *testing.T) {
	t.Parallel()

	sh, err := shell.New()
	if err != nil {
		t.Fatal(err)
	}

	sh.Logger = shell.TestingLogger{t}

	_, err = findPathToSSHTools(sh)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSSHKeyscanReturnsOutput(t *testing.T) {
	t.Parallel()

	sh := newTestShell(t)

	keyScan, err := bintest.NewMock("ssh-keyscan")
	if err != nil {
		t.Fatal(err)
	}
	defer keyScan.CheckAndClose(t)

	sh.Env.Set("PATH", filepath.Dir(keyScan.Path))

	keyScan.
		Expect("github.com").
		AndWriteToStdout("github.com ssh-rsa xxx=").
		AndExitWith(0)

	keyScanOutput, err := sshKeyScan(sh, "github.com")
	if err != nil {
		t.Fatal(err)
	}

	if keyScanOutput != "github.com ssh-rsa xxx=" {
		t.Error("bad key scan output", keyScanOutput)
	}
}

func TestSSHKeyscanWithHostAndPortReturnsOutput(t *testing.T) {
	t.Parallel()

	sh := newTestShell(t)

	keyScan, err := bintest.NewMock("ssh-keyscan")
	if err != nil {
		t.Fatal(err)
	}
	defer keyScan.CheckAndClose(t)

	sh.Env.Set("PATH", filepath.Dir(keyScan.Path))

	keyScan.
		Expect("-p", "123", "github.com").
		AndWriteToStdout("github.com ssh-rsa xxx=").
		AndExitWith(0)

	keyScanOutput, err := sshKeyScan(sh, "github.com:123")
	if err != nil {
		t.Fatal(err)
	}

	if keyScanOutput != "github.com ssh-rsa xxx=" {
		t.Error("bad key scan output", keyScanOutput)
	}
}

func TestSSHKeyscanRetriesOnExit1(t *testing.T) {
	t.Parallel()

	sh := newTestShell(t)

	keyScan, err := bintest.NewMock("ssh-keyscan")
	if err != nil {
		t.Fatal(err)
	}
	defer keyScan.CheckAndClose(t)

	sh.Env.Set("PATH", filepath.Dir(keyScan.Path))

	keyScan.
		Expect("github.com").
		AndWriteToStderr("it failed").
		Exactly(3).
		AndExitWith(1)

	keyScanOutput, err := sshKeyScan(sh, "github.com")

	if keyScanOutput != "" {
		t.Error("bad keyscan output", keyScanOutput)
	}

	if err == nil || err.Error() != "`ssh-keyscan \"github.com\"` failed" {
		t.Error("bad error from keyscan", err)
	}
}

func TestSSHKeyscanRetriesOnBlankOutputAndExit0(t *testing.T) {
	t.Parallel()

	sh := newTestShell(t)

	keyScan, err := bintest.NewMock("ssh-keyscan")
	if err != nil {
		t.Fatal(err)
	}
	defer keyScan.CheckAndClose(t)

	sh.Env.Set("PATH", filepath.Dir(keyScan.Path))

	keyScan.
		Expect("github.com").
		AndWriteToStdout("").
		Exactly(3).
		AndExitWith(0)

	keyScanOutput, err := sshKeyScan(sh, "github.com")

	if keyScanOutput != "" {
		t.Error("bad keyscan output", keyScanOutput)
	}

	if err == nil || err.Error() != "`ssh-keyscan \"github.com\"` returned nothing" {
		t.Error("bad error from keyscan", err)
	}
}

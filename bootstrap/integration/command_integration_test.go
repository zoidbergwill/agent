package integration

import (
	"testing"

	"github.com/buildkite/agent/bootstrap/shell"
	"github.com/buildkite/bintest"
)

func TestPreExitHooksRunsAfterCommandFails(t *testing.T) {
	tester, err := NewBootstrapTester()
	if err != nil {
		t.Fatal(err)
	}
	defer tester.Close()

	// Mock out the meta-data calls to the agent after checkout
	agent := tester.MustMock(t, "buildkite-agent")
	agent.
		Expect("meta-data", "exists", "buildkite:git:commit").
		AndExitWith(0)

	preExitFunc := func(c *bintest.Call) {
		cmdExitStatus := c.GetEnv(`BUILDKITE_COMMAND_EXIT_STATUS`)
		if cmdExitStatus != "1" {
			t.Errorf("Expected an exit status of 1, got %v", cmdExitStatus)
		}
		c.Exit(0)
	}

	tester.ExpectGlobalHook("pre-exit").Once().AndCallFunc(preExitFunc)
	tester.ExpectLocalHook("pre-exit").Once().AndCallFunc(preExitFunc)

	if err = tester.Run(t, "BUILDKITE_COMMAND=false"); err == nil {
		t.Fatal("Expected the bootstrap to fail")
	}

	tester.CheckMocks(t)
}

func TestExitCodeFromCommandIsRespectedRegardlessOfPostHooks(t *testing.T) {
	tester, err := NewBootstrapTester()
	if err != nil {
		t.Fatal(err)
	}
	defer tester.Close()

	// Mock out the meta-data calls to the agent after checkout
	agent := tester.MustMock(t, "buildkite-agent")
	agent.
		Expect("meta-data", "exists", "buildkite:git:commit").
		AndExitWith(0)

	tester.ExpectGlobalHook(`post-command`).Once().AndExitWith(0)
	tester.ExpectLocalHook(`post-command`).Once().AndExitWith(0)
	tester.ExpectGlobalHook(`pre-exit`).Once().AndExitWith(0)
	tester.ExpectLocalHook(`pre-exit`).Once().AndExitWith(0)

	if err = tester.Run(t, "BUILDKITE_COMMAND=false"); err == nil {
		t.Fatal("Expected the bootstrap to fail")
	}

	exitCode := shell.GetExitCode(err)
	if exitCode != 1 {
		t.Fatalf("Expected an exit code of %d, got %d", 1, exitCode)
	}

	tester.CheckMocks(t)
}

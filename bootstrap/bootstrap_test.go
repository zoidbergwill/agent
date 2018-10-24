package bootstrap

import (
	"testing"
)

var agentNameTests = []struct {
	agentName string
	expected  string
}{
	{"My Agent", "My-Agent"},
	{":docker: My Agent", "-docker--My-Agent"},
	{"My \"Agent\"", "My--Agent-"},
}

func TestDirForAgentName(t *testing.T) {
	t.Parallel()

	for _, test := range agentNameTests {
		if d := dirForAgentName(test.agentName); d != test.expected {
			t.Fatal("bad dir for agent name", d)
		}
	}
}

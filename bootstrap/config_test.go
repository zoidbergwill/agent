package bootstrap

import (
	"testing"

	"github.com/buildkite/agent/env"
	"github.com/stretchr/testify/assert"
)

func TestConfigCanBeReadFromEnvironment(t *testing.T) {
	t.Parallel()

	config := &Config{
		Repository:                   "https://original.host/repo.git",
		AutomaticArtifactUploadPaths: "llamas/",
		GitCloneFlags:                "--prune",
		GitCleanFlags:                "-v",
	}

	environ := env.FromSlice([]string{
		"BUILDKITE_ARTIFACT_PATHS=newpath",
		"BUILDKITE_GIT_CLONE_FLAGS=-f",
		"BUILDKITE_SOMETHING_ELSE=1",
		"BUILDKITE_REPO=https://my.mirror/repo.git",
	})

	changes := config.ReadFromEnvironment(environ)

	assert.Equal(t, map[string]string{
		"BUILDKITE_ARTIFACT_PATHS":  "newpath",
		"BUILDKITE_GIT_CLONE_FLAGS": "-f",
		"BUILDKITE_REPO":            "https://my.mirror/repo.git",
	}, changes)

	assert.Equal(t, *config, Config{
		Repository:                   "https://my.mirror/repo.git",
		AutomaticArtifactUploadPaths: "newpath",
		GitCloneFlags:                "-f",
		GitCleanFlags:                "-v",
	})
}

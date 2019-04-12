package bootstrap

import (
	"reflect"

	"github.com/buildkite/agent/env"
)

// HookModifiableConfigVars are environment variables that can
// be modified in hooks
var HookModifiableConfigVars = []string{
	`BUILDKITE_REPO`,
	`BUILDKITE_REFSPEC`,
	`BUILDKITE_GIT_CLONE_FLAGS`,
	`BUILDKITE_GIT_FETCH_FLAGS`,
	`BUILDKITE_GIT_CLONE_MIRROR_FLAGS`,
	`BUILDKITE_GIT_CLEAN_FLAGS`,
	`BUILDKITE_ARTIFACT_PATHS`,
	`BUILDKITE_ARTIFACT_UPLOAD_DESTINATION`,
	`BUILDKITE_PLUGIN_VALIDATION`,
}

// Config provides the configuration for the Bootstrap.
//
// It's initially populated in the agent and then passed to the bootstrap subprocess
// via environment variables. As the bootstrap runs, certain config can be overridden
// in hooks, where as some are overwritten for each execution.
type Config struct {
	// The path to the config that was loaded
	ConfigPath string `cli:"config-path" env:"BUILDKITE_CONFIG_PATH"`

	// The access token used by the agent
	AccessToken string `cli:"access-token" env:"BUILDKITE_AGENT_ACCESS_TOKEN"`

	// The agent API used by the agent
	Endpoint string `cli:"endpoint" env:"BUILDKITE_AGENT_ENDPOINT"`

	// The PID of the agent process that invoked the bootstrap
	AgentPID int `cli:"agent-pid" env:"BUILDKITE_AGENT_PID"`

	// The command to run
	Command string `cli:"command" env:"BUILDKITE_COMMAND"`

	// The ID of the job being run
	JobID string `cli:"job" env:"BUILDKITE_JOB_ID" validate:"required"`

	// If the bootstrap is in debug mode
	Debug bool `cli:"debug" env:"BUILDKITE_DEBUG"`

	// The repository that needs to be cloned, or blank to skip checkout (modifiable in hooks)
	Repository string `cli:"repository" env:"BUILDKITE_REPO"`

	// The commit being built
	Commit string `cli:"commit" env:"BUILDKITE_COMMIT"`

	// The branch of the commit
	Branch string `cli:"branch" env:"BUILDKITE_BRANCH"`

	// The tag of the job commit
	Tag string `cli:"tag" env:"BUILDKITE_TAG"`

	// Optional refspec to override git fetch (modifiable in hooks)
	RefSpec string `cli:"refspec" env:"BUILDKITE_REFSPEC"`

	// Plugin definition for the job
	Plugins string `cli:"plugins" env:"BUILDKITE_PLUGINS"`

	// Should git submodules be checked out
	GitSubmodules bool `cli:"git-submodules" env:"BUILDKITE_GIT_SUBMODULES"`

	// If the commit was part of a pull request, this will container the PR number
	PullRequest string `cli:"pullrequest" env:"BUILDKITE_PULL_REQUEST"`

	// The provider of the the pipeline
	PipelineProvider string `cli:"pipeline-provider" env:"BUILDKITE_PIPELINE_PROVIDER"`

	// Slug of the current organization
	OrganizationSlug string `cli:"organization" validate:"required" env:"BUILDKITE_ORGANIZATION_SLUG"`

	// Slug of the current pipeline
	PipelineSlug string `cli:"pipeline" env:"BUILDKITE_PIPELINE_SLUG"`

	// Name of the agent running the bootstrap
	AgentName string `cli:"agent" env:"BUILDKITE_AGENT_NAME" validate:"required"`

	// Should the bootstrap remove an existing checkout before running the job
	CleanCheckout bool `cli:"clean-checkout" env:"BUILDKITE_CLEAN_CHECKOUT"`

	// Flags to pass to "git clone" command (modifiable in hooks)
	GitCloneFlags string `cli:"git-clone-flags" env:"BUILDKITE_GIT_CLONE_FLAGS"`

	// Flags to pass to "git fetch" command (modifiable in hooks)
	GitFetchFlags string `cli:"git-fetch-flags" env:"BUILDKITE_GIT_FETCH_FLAGS"`

	// Flags to pass to "git clone" command for mirroring (modifiable in hooks)
	GitCloneMirrorFlags string `cli:"git-clone-mirror-flags" env:"BUILDKITE_GIT_CLONE_MIRROR_FLAGS"`

	// Flags to pass to "git clean" command  (modifiable in hooks)
	GitCleanFlags string `cli:"git-clean-flags" env:"BUILDKITE_GIT_CLEAN_FLAGS"`

	// Whether or not to run the hooks/commands in a PTY
	RunInPty bool `cli:"pty" env:"BUILDKITE_PTY"`

	// Are aribtary commands allowed to be executed
	CommandEval bool `cli:"command-eval" env:"BUILDKITE_COMMAND_EVAL"`

	// Are plugins enabled?
	PluginsEnabled bool `cli:"plugins-enabled" env:"BUILDKITE_PLUGINS_ENABLED"`

	// Whether to validate plugin configuration
	PluginValidation bool `cli:"plugin-validation" env:"BUILDKITE_PLUGIN_VALIDATION"`

	// Are local hooks enabled?
	LocalHooksEnabled bool `cli:"local-hooks-enabled" env:"BUILDKITE_LOCAL_HOOKS_ENABLED"`

	// Path where the builds will be run
	BuildPath string `cli:"build-path" env:"BUILDKITE_BUILD_PATH"`

	// Path where the repository mirrors are stored
	GitMirrorsPath string `cli:"git-mirrors-path" env:"BUILDKITE_GIT_MIRRORS_PATH"`

	// Seconds to wait before allowing git mirror clone lock to be acquired
	GitMirrorsLockTimeout int `cli:"git-mirrors-lock-timeout" env:"BUILDKITE_GIT_MIRRORS_LOCK_TIMEOUT"`

	// Path to the buildkite-agent binary
	BinPath string `cli:"bin-path" env:"BUILDKITE_BIN_PATH"`

	// Path to the global hooks
	HooksPath string `cli:"hooks-path" env:"BUILDKITE_HOOKS_PATH"`

	// Path to the plugins directory
	PluginsPath string `cli:"plugins-path" env:"BUILDKITE_PLUGINS_PATH"`

	// Paths to automatically upload as artifacts when the build finishes (modifiable in hooks)
	AutomaticArtifactUploadPaths string `cli:"artifact-upload-paths" env:"BUILDKITE_ARTIFACT_PATHS"`

	// A custom destination to upload artifacts to (i.e. s3://...)
	ArtifactUploadDestination string `cli:"artifact-upload-destination" env:"BUILDKITE_ARTIFACT_UPLOAD_DESTINATION"`

	// Whether ssh-keyscan is run on ssh hosts before checkout
	SSHKeyscan bool `cli:"ssh-keyscan" env:"BUILDKITE_SSH_KEYSCAN"`

	// The shell used to execute commands
	Shell string `cli:"shell" env:"BUILDKITE_SHELL"`

	// Phases to execute, defaults to all phases
	Phases []string `cli:"phases" env:"BUILDKITE_BOOTSTRAP_PHASES"`
}

// ReadFromEnvironment reads configuration from the Environment, returns a map
// of the env keys that changed and the new values
func (c *Config) ReadFromEnvironment(environ *env.Environment) map[string]string {
	changed := map[string]string{}

	// Use reflection for the type and values
	t := reflect.TypeOf(*c)
	v := reflect.ValueOf(c).Elem()

	// Iterate over all available fields and read the tag value
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		// Find struct fields with env tag
		if tag := field.Tag.Get(`env`); tag != "" && environ.Exists(tag) {
			newValue, _ := environ.Get(tag)

			// We only care if the value has changed
			if newValue != value.String() {
				value.SetString(newValue)
				changed[tag] = newValue
			}
		}
	}

	return changed
}

// ConfigEnvironmentKeys returns a list of environment variables that can
// be mapped to config values
func ConfigEnvironmentKeys() []string {
	keys := []string{}
	c := &Config{}
	t := reflect.TypeOf(*c)

	for i := 0; i < t.NumField(); i++ {
		if envKey := t.Field(i).Tag.Get(`env`); envKey != "" {
			keys = append(keys, envKey)
		}
	}

	return keys
}

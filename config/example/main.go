package main

import (
	"log"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclparse"
)

const hclExample = `

# The token from your Buildkite "Agents" page
token = "xxx"

# The name of the agent
name = "%hostname-%n"

# The priority of the agent (higher priorities are assigned work first)
# priority = 1

# Tags for the agent (default is "queue=default")
# tags = "key1=val2,key2=val2"

# Include the host's EC2 meta-data as tags (instance-id, instance-type, and ami-id)
# tags-from-ec2 = true

# Include the host's EC2 tags as tags
# tags-from-ec2-tags = true

# Include the host's Google Cloud instance meta-data as tags (instance-id, machine-type, preemptible, project-id, region, and zone)
# tags-from-gcp = true

# Include the host's Google Cloud instance labels as tags
# tags-from-gcp-labels = true

# Path to a custom bootstrap command to run. By default this is buildkite-agent bootstrap.
# This allows you to override the entire execution of a job. Generally you should use hooks instead!
# See https://buildkite.com/docs/agent/hooks
# bootstrap-script = ""

# Path to where the builds will run from
build-path = "$HOME/.buildkite-agent/builds"

# Directory where the hook scripts are found
hooks-path = "$HOME/.buildkite-agent/hooks"

# Directory where plugins will be installed
plugins-path = "$HOME/.buildkite-agent/plugins"

# Flags to pass to the git clone command
# git-clone-flags = ["-v"]

# Flags to pass to the git clean command
# git-clean-flags = ["-ffxdq"]

# Do not run jobs within a pseudo terminal
# no-pty = true

# Don't automatically verify SSH fingerprints
# no-automatic-ssh-fingerprint-verification = true

# Don't allow this agent to run arbitrary console commands
# no-command-eval = true

# Don't allow this agent to run plugins
# no-plugins = true

# Enable debug mode
# debug = true

# Don't show colors in logging
# no-color = true

# Customize settings for a specific pipeline
pipeline "buildkite" "agent" {
	git-clone-flags = ["-v", "--depth=1"]
	git-fetch-flags = ["-v", "--depth=1"]
}

# Customize settings for a specific repository
repository "git@github.com:buildkite/agent.git" {
	no-command-eval = true
}
`

type Pipeline struct {
	Org           string    `hcl:"org-slug,label"`
	Pipeline      string    `hcl:"pipeline-slug,label"`
	GitCloneFlags *[]string `hcl:"git-clone-flags"`
	GitFetchFlags *[]string `hcl:"git-fetch-flags"`
}

type Repository struct {
	Repository    string    `hcl:"repository,label"`
	NoCommandEval *bool     `hcl:"no-command-eval"`
	GitCloneFlags *[]string `hcl:"git-clone-flags"`
	GitFetchFlags *[]string `hcl:"git-fetch-flags"`
}

type Config struct {
	Name         string        `hcl:"name"`
	Token        string        `hcl:"token"`
	HooksPath    string        `hcl:"hooks-path"`
	PluginsPath  string        `hcl:"plugins-path"`
	BuildPath    string        `hcl:"build-path"`
	Llamas       *string       `hcl:"llamas"`
	Pipelines    []*Pipeline   `hcl:"pipeline,block"`
	Repositories []*Repository `hcl:"repository,block"`
}

func main() {
	parser := hclparse.NewParser()
	f, parseDiags := parser.ParseHCL([]byte(hclExample), "test.hcl")
	if parseDiags.HasErrors() {
		log.Fatal(parseDiags.Error())
	}

	var conf Config
	decodeDiags := gohcl.DecodeBody(f.Body, nil, &conf)
	if decodeDiags.HasErrors() {
		log.Fatal(decodeDiags.Error())
	}

	spew.Dump(conf)
}

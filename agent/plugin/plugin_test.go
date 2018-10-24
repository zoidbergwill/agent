package plugin

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateFromJSON(t *testing.T) {
	t.Parallel()

	assertEqual := func(expected, actual interface{}) {
		if !reflect.DeepEqual(expected, actual) {
			t.Fatalf("expected %v, got %v", expected, actual)
		}
	}

	plugins, err := CreateFromJSON(`[{"http://github.com/buildkite/plugins/docker-compose#a34fa34":{"container":"app"}}, "github.com/buildkite/plugins/ping#master"]`)
	if err != nil {
		t.Fatal(err)
	}

	if l := len(plugins); l != 2 {
		t.Fatal("bad plugins length", l)
	}

	assertEqual(plugins[0].Location, "github.com/buildkite/plugins/docker-compose")
	assertEqual(plugins[0].Version, "a34fa34")
	assertEqual(plugins[0].Scheme, "http")
	assertEqual(plugins[0].Configuration, map[string]interface{}{"container": "app"})

	assertEqual(plugins[1].Location, "github.com/buildkite/plugins/ping")
	assertEqual(plugins[1].Version, "master")
	assertEqual(plugins[1].Scheme, "")
	assertEqual(plugins[1].Configuration, map[string]interface{}{})

	plugins, err = CreateFromJSON(`["ssh://git:foo@github.com/buildkite/plugins/docker-compose#a34fa34"]`)
	if err != nil {
		t.Fatal(err)
	}

	if l := len(plugins); l != 2 {
		t.Fatal("bad plugins length", l)
	}

	assertEqual(plugins[0].Location, "github.com/buildkite/plugins/docker-compose")
	assertEqual(plugins[0].Version, "a34fa34")
	assertEqual(plugins[0].Scheme, "ssh")
	assertEqual(plugins[0].Authentication, "git:foo")
}

func TestCreateFromJSONWithErrors(t *testing.T) {
	for _, tc := range []struct {
		location      string
		expectedError string
	}{
		{`blah`, "invalid character 'b' looking for beginning of value"},
		{`{"foo": "bar"}`, "JSON structure was not an array"},
		{`["github.com/buildkite/plugins/ping#master#lololo"]`, "Too many #'s in \"github.com/buildkite/plugins/ping#master#lololo\""},
	} {
		tc := tc
		t.Run(tc.location, func(tt *testing.T) {
			tt.Parallel()
			plugins, err := CreateFromJSON(tc.location)
			if err == nil || err.Error() != tc.expectedError {
				t.Fatal("expected error", err)
			}

			if l := len(plugins); l != 0 {
				t.Fatal("expected no plugins", l)
			}
		})
	}
}

func TestPluginNameParsedFromLocation(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		location     string
		expectedName string
	}{
		{"github.com/buildkite-plugins/docker-compose-buildkite-plugin.git", "docker-compose"},
		{"github.com/buildkite-plugins/docker-compose-buildkite-plugin", "docker-compose"},
		{"github.com/my-org/docker-compose-buildkite-plugin", "docker-compose"},
		{"github.com/buildkite/plugins/docker-compose", "docker-compose"},
		{"~/Development/plugins/test", "test"},
		{"~/Development/plugins/UPPER     CASE_party", "upper-case-party"},
		{"vendor/src/vendored with a space", "vendored-with-a-space"},
		{"", ""},
	} {
		tc := tc
		t.Run(tc.location, func(tt *testing.T) {
			tt.Parallel()
			plugin := &Plugin{Location: tc.location}
			assert.Equal(tt, tc.expectedName, plugin.Name())
		})
	}
}

func TestPluginIdentifierParsedFromLocation(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		location           string
		expectedIdentifier string
	}{
		{"github.com/buildkite/plugins/docker-compose/beta#master", "github-com-buildkite-plugins-docker-compose-beta-master"},
		{"github.com/buildkite/plugins/docker-compose/beta", "github-com-buildkite-plugins-docker-compose-beta"},
		{"192.168.0.1/foo.git#12341234", "192-168-0-1-foo-git-12341234"},
		{"/foo/bar/", "foo-bar"},
	} {
		tc := tc
		t.Run(tc.location, func(tt *testing.T) {
			tt.Parallel()

			p := &Plugin{Location: tc.location}
			id, err := p.Identifier()
			if err != nil {
				t.Fatal(err)
			}
			if id != tc.expectedIdentifier {
				t.Error("bad plugin identifier", id)
			}
		})
	}
}

func TestPluginRepositoryAndSubdirectoryParsedFromLocation(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		location             string
		expectedRepository   string
		expectedSubdirectory string
	}{
		{"github.com/buildkite/plugins/docker-compose/beta", "https://github.com/buildkite/plugins", "docker-compose/beta"},
		{"github.com/buildkite/test-plugin", "https://github.com/buildkite/test-plugin", ""},
		{"bitbucket.org/user/project/sub/directory", "https://bitbucket.org/user/project", "sub/directory"},
		{"114.135.234.212/foo.git", "https://114.135.234.212/foo.git", ""},
		{"github.com/buildkite/plugins/docker-compose/beta", "https://github.com/buildkite/plugins", "docker-compose/beta"},
		{"/Users/keithpitt/Development/plugins.git/test-plugin", "/Users/keithpitt/Development/plugins.git", "test-plugin"},
	} {
		tc := tc
		t.Run(tc.location, func(tt *testing.T) {
			tt.Parallel()

			p := &Plugin{Location: tc.location}
			repo, err := p.Repository()
			if err != nil {
				t.Fatal(err)
			}

			if repo != tc.expectedRepository {
				t.Error("bad repository", repo)
			}

			sub, err := p.RepositorySubdirectory()
			if err != nil {
				t.Fatal(err)
			}

			if sub != tc.expectedSubdirectory {
				t.Error("bad subdirectory", sub)
			}
		})
	}

	for _, tc := range []struct {
		location      string
		expectedError string
	}{
		{"github.com/buildkite", `Incomplete github.com path "github.com/buildkite"`},
		{"bitbucket.org/buildkite", `Incomplete bitbucket.org path "bitbucket.org/buildkite"`},
		{"", "Missing plugin location"},
	} {
		tc := tc
		t.Run(tc.location, func(tt *testing.T) {
			plugin := &Plugin{Location: tc.location}
			_, err := plugin.Repository()
			if err == nil || err.Error() != tc.expectedError {
				t.Fatal("expected error from Repository", err)
			}

			_, err = plugin.RepositorySubdirectory()
			if err == nil || err.Error() != tc.expectedError {
				t.Fatal("expected error from RepositorySubdirectory", err)
			}
		})
	}
}

func TestPluginRepositoryAndSubdirectoryParsedFromLocationWithAuthentication(t *testing.T) {
	t.Parallel()

	plugin := &Plugin{
		Location:       "bitbucket.org/user/project/sub/directory",
		Scheme:         "http",
		Authentication: "foo:bar",
	}
	repo, err := plugin.Repository()
	if err != nil {
		t.Fatal(err)
	}

	if repo != "http://foo:bar@bitbucket.org/user/project" {
		t.Fatal("bad repo", repo)
	}

	sub, err := plugin.RepositorySubdirectory()
	if err != nil {
		t.Fatal(err)
	}

	if sub != "sub/directory" {
		t.Fatal("bad sub", repo)
	}
}

func TestConfigurationToEnvironment(t *testing.T) {
	t.Parallel()

	assertPluginConfigEqualsEnv := func(configJson string, expected []string) {
		var config map[string]interface{}

		if err := json.Unmarshal([]byte(configJson), &config); err != nil {
			t.Fatal(err)
		}

		plugins, err := CreateFromJSON(fmt.Sprintf(
			`[ { "%s": %s } ]`,
			"github.com/buildkite-plugins/docker-compose-buildkite-plugin",
			configJson,
		))
		if err != nil {
			t.Fatal(err)
		}

		if l := len(plugins); l != 1 {
			t.Fatal("expected 1 plugin", err)
		}

		actual, err := plugins[0].ConfigurationToEnvironment()
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(expected, actual.ToSlice()) {
			t.Fatalf("expected %v, got %v", expected, actual)
		}
	}

	assertPluginConfigEqualsEnv(
		`{ "config-key": 42 }`,
		[]string{"BUILDKITE_PLUGIN_DOCKER_COMPOSE_CONFIG_KEY=42"},
	)

	assertPluginConfigEqualsEnv(
		`{ "container": "app", "some-other-setting": "else right here" }`,
		[]string{
			"BUILDKITE_PLUGIN_DOCKER_COMPOSE_CONTAINER=app",
			"BUILDKITE_PLUGIN_DOCKER_COMPOSE_SOME_OTHER_SETTING=else right here"})

	assertPluginConfigEqualsEnv(
		`{ "and _ with a    - number": 12 }`,
		[]string{"BUILDKITE_PLUGIN_DOCKER_COMPOSE_AND_WITH_A_NUMBER=12"})

	assertPluginConfigEqualsEnv(
		`{ "bool-true-key": true, "bool-false-key": false }`,
		[]string{
			"BUILDKITE_PLUGIN_DOCKER_COMPOSE_BOOL_FALSE_KEY=false",
			"BUILDKITE_PLUGIN_DOCKER_COMPOSE_BOOL_TRUE_KEY=true"})

	assertPluginConfigEqualsEnv(
		`{ "array-key": [ "array-val-1", "array-val-2" ] }`,
		[]string{
			"BUILDKITE_PLUGIN_DOCKER_COMPOSE_ARRAY_KEY_0=array-val-1",
			"BUILDKITE_PLUGIN_DOCKER_COMPOSE_ARRAY_KEY_1=array-val-2"})

	assertPluginConfigEqualsEnv(
		`{ "array-key": [ 42, 43, 44 ] }`,
		[]string{
			"BUILDKITE_PLUGIN_DOCKER_COMPOSE_ARRAY_KEY_0=42",
			"BUILDKITE_PLUGIN_DOCKER_COMPOSE_ARRAY_KEY_1=43",
			"BUILDKITE_PLUGIN_DOCKER_COMPOSE_ARRAY_KEY_2=44"})

	assertPluginConfigEqualsEnv(
		`{ "array-key": [ 42, 43, "foo" ] }`,
		[]string{
			"BUILDKITE_PLUGIN_DOCKER_COMPOSE_ARRAY_KEY_0=42",
			"BUILDKITE_PLUGIN_DOCKER_COMPOSE_ARRAY_KEY_1=43",
			"BUILDKITE_PLUGIN_DOCKER_COMPOSE_ARRAY_KEY_2=foo"})

	assertPluginConfigEqualsEnv(
		`{ "array-key": [ { "subkey": "subval" } ] }`,
		[]string{
			"BUILDKITE_PLUGIN_DOCKER_COMPOSE_ARRAY_KEY_0_SUBKEY=subval"})

	assertPluginConfigEqualsEnv(
		`{ "array-key": [ { "subkey": [1, 2, "llamas"] } ] }`,
		[]string{
			"BUILDKITE_PLUGIN_DOCKER_COMPOSE_ARRAY_KEY_0_SUBKEY_0=1",
			"BUILDKITE_PLUGIN_DOCKER_COMPOSE_ARRAY_KEY_0_SUBKEY_1=2",
			"BUILDKITE_PLUGIN_DOCKER_COMPOSE_ARRAY_KEY_0_SUBKEY_2=llamas",
		})
}

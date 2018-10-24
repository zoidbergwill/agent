package plugin

import (
	"reflect"
	"testing"

	"github.com/qri-io/jsonschema"
)

var testPluginDef = `
name: test-plugin
description: A test plugin
author: https://github.com/buildkite
requirements:
  - docker
  - docker-compose
configuration:
  properties:
    run:
      type: string
    build:
      type: [ string, array ]
      minimum: 1
  oneOf:
    - required:
      - run
    - required:
      - build
  additionalProperties: false
`

func TestDefinitionParsesYaml(t *testing.T) {
	def, err := ParseDefinition([]byte(testPluginDef))
	if err != nil {
		t.Fatal(err)
	}

	if def.Name != `test-plugin` {
		t.Fatal("bad plugin def name", def.Name)
	}

	if !reflect.DeepEqual(def.Requirements, []string{`docker`, `docker-compose`}) {
		t.Fatal("bad plugin def requirements", def.Requirements)
	}
}

func TestDefinitionValidationFailsIfDependenciesNotMet(t *testing.T) {
	validator := &Validator{
		commandExists: func(cmd string) bool {
			return false
		},
	}

	def := &Definition{
		Requirements: []string{`llamas`},
	}

	res := validator.Validate(def, nil)

	if res.Valid() {
		t.Fatal("validator should have failed")
	}

	if reflect.DeepEqual(res.Errors, []string{
		`Required command "llamas" isn't in PATH`,
	}) {
		t.Fatal("missing error from validator", res.Errors)
	}
}

func TestDefinitionValidatesConfiguration(t *testing.T) {
	validator := &Validator{
		commandExists: func(cmd string) bool {
			return false
		},
	}

	def := &Definition{
		Configuration: jsonschema.Must(`{
			"type": "object",
			"properties": {
				"llamas": {
					"type": "string"
				},
				"alpacas": {
					"type": "string"
				}
			},
			"required": ["llamas", "alpacas"]
		}`),
	}

	res := validator.Validate(def, map[string]interface{}{
		"llamas": "always",
	})

	if res.Valid() {
		t.Fatal("validator should have failed")
	}

	if reflect.DeepEqual(res.Errors, []string{
		`/: {"llamas":"always"} "alpacas" value is required`,
	}) {
		t.Fatal("missing error from validator", res.Errors)
	}
}

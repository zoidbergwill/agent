package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/buildkite/agent/env"
	"github.com/buildkite/agent/logger"
	"github.com/buildkite/interpolate"
	"github.com/ghodss/yaml"
)

type PipelineParser struct {
	Env      *env.Environment
	Filename string
	Pipeline []byte
}

func (p PipelineParser) Parse() (pipeline interface{}, err error) {
	if p.Env == nil {
		p.Env = env.FromSlice(os.Environ())
	}

	// First try and figure out the format from the filename
	format, err := inferFormat(p.Pipeline, p.Filename)
	if err != nil {
		return nil, err
	}

	log.Printf("Inferred format")

	// Unmarshal the pipeline into an actual data structure
	unmarshaled, err := unmarshal(p.Pipeline, format)
	if err != nil {
		return nil, err
	}

	log.Printf("Unmarshalled")

	// Preprocess any env that are defined in the top level block and place them into env for
	// later interpolation. We do this a few times so that you can reference env vars in other env vars
	if unmarshaledMap, ok := unmarshaled.(map[string]interface{}); ok {
		if envMap, ok := unmarshaledMap["env"].(map[string]interface{}); ok {
			if err = p.interpolateEnvBlock(envMap); err != nil {
				return nil, err
			}
		}
	}

	log.Printf("Interpolated env")

	// Recursivly go through the entire pipeline and perform environment
	// variable interpolation on strings
	interpolated, err := p.interpolate(unmarshaled)
	if err != nil {
		return nil, err
	}

	log.Printf("Interpolated the rest")

	return interpolated, nil
}

func (p PipelineParser) interpolateEnvBlock(envMap map[string]interface{}) error {
	// do a first pass without interpolation
	for k, v := range envMap {
		switch tv := v.(type) {
		case string, int, bool:
			p.Env.Set(k, fmt.Sprintf("%v", tv))
		}
	}

	// next do a pass of interpolation and read the results
	for k, v := range envMap {
		switch tv := v.(type) {
		case string:
			interpolated, err := interpolate.Interpolate(p.Env, tv)
			if err != nil {
				return err
			}
			p.Env.Set(k, interpolated)
		}
	}
	return nil
}

func inferFormat(pipeline []byte, filename string) (string, error) {
	// If we have a filename, try and figure out a format from that
	if filename != "" {
		extension := filepath.Ext(filename)
		if extension == ".yaml" || extension == ".yml" {
			return "yaml", nil
		} else if extension == ".json" {
			return "json", nil
		}
	}

	// Boo...we couldn't figure it out based on the filename. Next we'll
	// use a very dirty and ugly way of detecting if the pipeline is JSON.
	// It's not nice...but seems to work really well for our use case!
	firstCharacter := string(strings.TrimSpace(string(pipeline))[0])
	if firstCharacter == "{" || firstCharacter == "[" {
		return "json", nil
	}

	// If nothing else could be figured out, then default to YAML
	return "yaml", nil
}

func unmarshal(pipeline []byte, format string) (interface{}, error) {
	var unmarshaled interface{}

	if format == "yaml" {
		logger.Debug("Parsing pipeline configuration as YAML")

		err := yaml.Unmarshal(pipeline, &unmarshaled)
		if err != nil {
			// Error messages from the YAML parser have this ugly
			// prefix, so I'll just strip it for the sake of the
			// "aesthetics"
			message := strings.Replace(fmt.Sprintf("%s", err), "error converting YAML to JSON: yaml: ", "", 1)

			return nil, fmt.Errorf("Failed to parse YAML: %s", message)
		}
	} else if format == "json" {
		logger.Debug("Parsing pipeline configuration as JSON")

		err := json.Unmarshal(pipeline, &unmarshaled)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse JSON: %s", err)
		}
	} else {
		if format == "" {
			return nil, fmt.Errorf("No format was supplied")
		} else {
			return nil, fmt.Errorf("Unknown format `%s`", format)
		}
	}

	return unmarshaled, nil
}

func (p PipelineParser) interpolate(obj interface{}) (interface{}, error) {
	// Make sure there's something actually to interpolate
	if obj == nil {
		return nil, nil
	}

	// walk the tree and interpolate
	err := interpolateRecursive(&obj, func(value string) (string, error) {
		log.Printf("Interpolating %v", value)
		defer log.Printf("Done!")
		return interpolate.Interpolate(p.Env, value)
	})

	return obj, err
}

// interpolateRecursive walks structures by iterating through slices and maps and applying an interpolator to strings
func interpolateRecursive(obj *interface{}, interpolator func(s string) (string, error)) error {
	if obj == nil {
		return nil
	}

	log.Printf("%#v", *obj)

	// walk through maps
	if m, isMap := (*obj).(map[string]interface{}); isMap {
		for k, v := range m {
			// keys can be interpolated too
			newK, err := interpolator(k)
			if err != nil {
				return err
			}

			// if the key changes, we need to update it
			if newK != k {
				delete(m, k)
				m[newK] = v
				k = newK
			}

			// handle string values
			if str, isString := (v).(string); isString {
				var err error
				m[k], err = interpolator(str)
				if err != nil {
					return err
				}
				continue
			}

			if err = interpolateRecursive(&v, interpolator); err != nil {
				return err
			}
		}

		return nil
	}

	// handle slices
	if s, isSlice := (*obj).([]interface{}); isSlice {
		for idx, v := range s {
			// handle string values
			if str, isString := (v).(string); isString {
				var err error
				s[idx], err = interpolator(str)
				if err != nil {
					return err
				}
			}

			if err := interpolateRecursive(&v, interpolator); err != nil {
				return err
			}
		}

		return nil
	}

	return nil
}

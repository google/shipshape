/*
 * Copyright 2014 Google Inc. All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package service

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

type testSpec struct {
	label      string
	eventName  string
	images     []string
	ignore     []string
	categories []string
}

func TestValidConfig(t *testing.T) {
	yaml := `
global:
  images:
    - foo.com:5050/foo/bar:prod
    - bar/baz:hork
  ignore:
    - file=.gitignore
    - third_party/

events:
  - event: default
    categories:
      - go vet
      - JSHint
      - Klippy
  - event: deploy
    categories:
      - Loadtest`

	tests := []testSpec{
		{
			"Event with special categories",
			"deploy",
			[]string{"foo.com:5050/foo/bar:prod", "bar/baz:hork"},
			[]string{"file=.gitignore", "third_party/"},
			[]string{"Loadtest"},
		},
		{
			"Event with no special categories",
			"something",
			[]string{"foo.com:5050/foo/bar:prod", "bar/baz:hork"},
			[]string{"file=.gitignore", "third_party/"},
			[]string{"go vet", "JSHint", "Klippy"},
		},
	}

	for _, test := range tests {
		if failure := test.run(yaml); failure != nil {
			t.Error(failure)
		}
	}
}

func TestValidConfigNoDefault(t *testing.T) {
	yaml := `
global:
  images:
    - foo.com:5050/foo/bar:prod
    - bar/baz:hork
  ignore:
    - file=.gitignore
    - third_party/

events:
  - event: deploy
    categories:
      - Loadtest`

	tests := []testSpec{
		{
			"Event with special categories",
			"deploy",
			[]string{"foo.com:5050/foo/bar:prod", "bar/baz:hork"},
			[]string{"file=.gitignore", "third_party/"},
			[]string{"Loadtest"},
		},
		{
			"Event with no special categories",
			"something",
			[]string{"foo.com:5050/foo/bar:prod", "bar/baz:hork"},
			[]string{"file=.gitignore", "third_party/"},
			nil,
		},
	}

	for _, test := range tests {
		if failure := test.run(yaml); failure != nil {
			t.Error(failure)
		}
	}
}

func TestValidYamlInvalidConfig(t *testing.T) {
	tests := []struct {
		label string
		yaml  string
		err   error
	}{
		{
			"Empty config file",
			"",
			errors.New("Config file must have an `events` section"),
		},
		{
			"No events",
			`
global:
  images:
    - foo.com:5050/foo/bar:prod
    - bar/baz:hork
  ignore:
    - file=.gitignore
    - third_party/`,
			errors.New("Config file must have an `events` section"),
		},
		{
			"Event with no categories",
			`
events:
  - event: deploy`,
			errors.New("Event \"deploy\" must specify at least one category"),
		},
		{
			"Event with no name",
			`
events:
  - event: review
    categories:
      - Loadtest
  - categories:
      - Benchmark`,
			errors.New("Event at index 1 is missing an event name"),
		},
		{
			"Event with whitespace name",
			`
events:
  - event: review
    categories:
      - Loadtest
  - event:                          
    categories:
      - Benchmark`,
			errors.New("Event at index 1 is missing an event name"),
		},
		{
			"Multiple events with same name",
			`
events:
  - event: review
    categories:
      - Loadtest
  - event: review
    categories:
      - Benchmark`,
			errors.New("Multiple events with name \"review\" (indexes 0, 1)"),
		},
	}

	for _, test := range tests {
		rawCfg, err := unmarshalConfigBytes([]byte(test.yaml))
		if err != nil {
			t.Errorf("Error in %q: %v", test.label, err.Error())
		}

		err = validateConfig(rawCfg)
		if err == nil {
			t.Errorf("Failed to find error in test %q", test.label)
		} else if err.Error() != test.err.Error() {
			t.Errorf("Unexpected error for %q: got %q, expected %q",
				test.label, err.Error(), test.err.Error())
		}
	}
}

func (ts *testSpec) run(yaml string) error {
	rawCfg, err := unmarshalConfigBytes([]byte(yaml))
	if err != nil {
		return err
	}

	cfg := buildConfig(rawCfg, ts.eventName)

	if !reflect.DeepEqual(cfg.images, ts.images) {
		return fmt.Errorf("Incorrect list of images for %q; got %v, expected %v",
			ts.label, cfg.images, ts.images)
	}
	if !reflect.DeepEqual(cfg.ignore, ts.ignore) {
		return fmt.Errorf("Incorrect list of ignores for %q; got %v, expected %v",
			ts.label, cfg.ignore, ts.ignore)
	}
	if !reflect.DeepEqual(cfg.categories, ts.categories) {
		return fmt.Errorf("Incorrect list of categories for %q; got %v, expected %v",
			ts.label, cfg.categories, ts.categories)
	}
	return nil
}

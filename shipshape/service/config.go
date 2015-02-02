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
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	yaml "gopkg.in/yaml.v2"

	configpb "shipshape/proto/shipshape_config_proto"
)

const (
	defaultName = "default"
)

// config is a struct for handling configuration for analyses. Given a Shipshape Context, it will access
// the appropriate config files to determine which analyzers should run on which files.
// Exported for test purposes only
type config struct {
	images     []string
	ignore     []string
	categories []string
}

// unmarshalConfigBytes parses a YAML payload into a Shipshape config. It normalizes
// all string fields to lowercase. Failure to parse the YAML input will result in an error
// and a nil config.
func unmarshalConfigBytes(configData []byte) (*configpb.ShipshapeConfig, error) {
	var config configpb.ShipshapeConfig
	if err := yaml.Unmarshal(configData, &config); err != nil {
		return nil, err
	}

	// TODO: normalize all events, categories, and excludes.

	return &config, nil
}

// eventWithName returns the event config stanza corresponding to the given name.
// If no matching event is found, return nil.
func eventWithName(rawConfig *configpb.ShipshapeConfig, eventName string) *configpb.EventConfig {
	for _, ec := range rawConfig.Events {
		if ec.GetEvent() == eventName {
			return ec
		}
	}
	return nil
}

// buildConfig takes a raw Shipshape configuration proto and builds up a summary of the
// categories, images, and ignore lines that apply to the given event.
func buildConfig(rawConfig *configpb.ShipshapeConfig, eventName string) *config {
	c := new(config)

	eventConfig := eventWithName(rawConfig, eventName)
	defaultConfig := eventWithName(rawConfig, defaultName)
	if eventConfig != nil {
		c.categories = append(c.categories, eventConfig.Categories...)
	} else if defaultConfig != nil {
		c.categories = append(c.categories, defaultConfig.Categories...)
	}
	if g := rawConfig.Global; g != nil {
		c.images = append(c.images, g.Images...)
		c.ignore = append(c.ignore, g.Ignore...)
	}
	return c
}

// validateConfig looks for errors in the given configuration proto.
// TODO(collinwinter): return all the errors, not just the first one.
func validateConfig(rawConfig *configpb.ShipshapeConfig) error {
	if len(rawConfig.Events) == 0 {
		return errors.New("Config file must have an `events` section")
	}
	eventNames := make(map[string][]string)
	for i, ec := range rawConfig.Events {
		if ec.Event == nil {
			return fmt.Errorf("Event at index %v is missing an event name", i)
		}
		if len(ec.Categories) == 0 {
			return fmt.Errorf("Event %q must specify at least one category", *ec.Event)
		}
		eventNames[ec.GetEvent()] = append(eventNames[ec.GetEvent()], strconv.Itoa(i))
	}
	for name, indexes := range eventNames {
		if len(indexes) > 1 {
			return fmt.Errorf("Multiple events with name %q (indexes %v)", name, strings.Join(indexes, ", "))
		}
	}
	return nil
}

// GlobalConfig retrieves the global configuration settings for the specified
// configuration file. Right now, this is just the list of third-party analyzer
// images to run.
func GlobalConfig(path string) ([]string, error) {
	cfg, err := loadConfig(filepath.Join(path, configFilename), "")
	if err != nil || cfg == nil {
		return nil, err
	}
	return cfg.images, nil
}

// loadConfig looks at given path for a Shipshape config file, loading the configuration
// for the given event, if found.
func loadConfig(configPath string, eventName string) (*config, error) {
	content, err := ioutil.ReadFile(configPath)
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	cfg, err := unmarshalConfigBytes(content)
	if err != nil {
		return nil, err
	}
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	return buildConfig(cfg, eventName), nil
}

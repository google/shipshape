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

// Package filevnames provides a utility to generate file VNames based on path
// regexp patterns and VName templates.
package filevnames

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"regexp"
	"strconv"

	spb "third_party/kythe/proto/storage_proto"
)

// A Config is an ordered set of (pattern, VName-template) pairs that can be
// used to construct VNames based on file paths.  This is primarily used amongst
// the Kythe extractors to ensure consistent file input VNames across
// compilations.  The order of the patterns matter so that the rules can have
// priorities, and most importantly, the last rule can be a general fallback
// "(.*)" pattern.
type Config struct {
	bases []*baseVName
}

type baseVName struct {
	Pattern pattern
	VName   *vnameTemplate
}

type vnameTemplate struct {
	Corpus, Root, Path string
}

// ParseFile returns a Config based on the JSON configuration file given.  The
// JSON config has the following format:
//   [
//     {
//       "pattern": "re2_regex_path_pattern",
//       "vname": {
//         "corpus": "corpus_template",
//         "root": "root_template",
//         "path": "path_template"
//       }
//     }, ...
//   ]
//
// The path patterns are RE2 regexp patterns used to match full file paths
// (they implicitly start with ^ and end with $).  The vname templates are
// strings with @<num>@ markers that will be replaced by the <num>'th regexp
// group on a successful path match.
func ParseFile(path string) (*Config, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseJSON(contents)
}

// ParseJSON parses a Config directly from JSON encoded in a []byte. See
// ParseFile for the exact format.
func ParseJSON(config []byte) (*Config, error) {
	c := &Config{}
	if err := json.Unmarshal(config, &c.bases); err != nil {
		return nil, err
	}

	return c, nil
}

// LookupVName returns a VName constructed from the Config's pattern templates.
// If no configured pattern matches, an empty VName will be returned.
func (c *Config) LookupVName(path string) *spb.VName {
	for _, b := range c.bases {
		if m := b.Pattern.FindStringSubmatch(path); len(m) != 0 {
			return b.VName.FillIn(m)
		}
	}

	return &spb.VName{}
}

// FillIn returns a VName by filling the template's @<num>@ markers with the
// values in the given slice of strings
func (t *vnameTemplate) FillIn(grps []string) *spb.VName {
	return &spb.VName{
		Corpus: fillTemplate(t.Corpus, grps),
		Root:   fillTemplate(t.Root, grps),
		Path:   fillTemplate(t.Path, grps),
	}
}

var markerRe = regexp.MustCompile("@\\d+@")

func fillTemplate(tmpl string, grps []string) *string {
	str := markerRe.ReplaceAllStringFunc(tmpl, func(marker string) string {
		n, err := strconv.Atoi(marker[1 : len(marker)-1])
		if err != nil {
			// Shouldn't reach this because regex disallows non-numbers
			log.Fatalf("VName template marker regexp failed: %v", err)
		}
		return grps[n]
	})
	return &str
}

type pattern struct {
	*regexp.Regexp
}

// UnmarshalJSON implements the json.Unmarshaller interface
func (p *pattern) UnmarshalJSON(val []byte) (err error) {
	var str string
	if err := json.Unmarshal(val, &str); err != nil {
		return err
	}
	p.Regexp, err = regexp.Compile("^" + str + "$")
	return
}

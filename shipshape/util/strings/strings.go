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

package strings

import (
	"reflect"
	"sort"
)

// Set is a representation of a set of strings.
// If you have a []string that you want to do a lot of
// set operations on, prefer using this type.
// If you only have a one-off usage, use SliceContains.
type Set map[string]bool

// Equal reports whether expect and actual contain exactly the same strings,
// without regard to order.
func Equal(expect, actual []string) bool {
	if len(expect) == 0 && len(actual) == 0 {
		return true
	}
	if len(expect) == 0 || len(actual) == 0 {
		return false
	}

	e := make([]string, len(expect))
	a := make([]string, len(actual))
	copy(e, expect)
	copy(a, actual)
	sort.Strings(e)
	sort.Strings(a)

	return reflect.DeepEqual(e, a)
}

// Contains reports whether item is an element of list.
// If you expect to do this a lot, prefer converting
// to a Set. This is fine for one-offs.
func Contains(list []string, item string) bool {
	for _, v := range list {
		if item == v {
			return true
		}
	}
	return false
}

// New returns a new Set containing the given strings.
func New(ss ...string) Set {
	set := make(Set)
	for _, s := range ss {
		set[s] = true
	}
	return set
}

// Intersect returns a Set representing the intersection of
// the two slices.
func (s Set) Intersect(s2 Set) Set {
	newS := make(Set)
	for str := range s {
		if s2[str] {
			newS[str] = true
		}
	}
	return newS
}

// AddSet modifies s in-place to include all the elements of addSet.
func (s Set) AddSet(addSet Set) Set {
	for val, _ := range addSet {
		s[val] = true
	}
	return s
}

// AddSlice modifies s in-place to include all the elements of slice.
func (s Set) AddSlice(slice []string) Set {
	for _, val := range slice {
		s[val] = true
	}
	return s
}

// Add modifies s in-place to include str.
func (s Set) Add(str string) Set {
	s[str] = true
	return s
}

// RemoveSlice modifies s in-place to remove all the elements of slice.
func (s Set) RemoveSlice(slice []string) Set {
	for _, val := range slice {
		delete(s, val)
	}
	return s
}

// RemoveSet modifies s in-place to remove all the elements of removeSet.
func (s Set) RemoveSet(removeSet Set) Set {
	for val, _ := range removeSet {
		delete(s, val)
	}
	return s
}

// Remove modifies s in-place to remove str.
func (s Set) Remove(str string) {
	delete(s, str)
}

// Contains reports whether val is a member of s.
func (s Set) Contains(val string) bool {
	return s[val]
}

// Reports whether the set is empty.
func (s Set) IsEmpty() bool {
	return len(s) == 0
}

// ToSlice returns a slice representation of s.
func (s Set) ToSlice() []string {
	var keys []string
	for k := range s {
		keys = append(keys, k)
	}
	return keys
}

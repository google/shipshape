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

package codealert

import (
	"testing"
)

func TestSamplePattern(t *testing.T) {
	var a CodeAlertAnalyzer
	notes := a.FindMatches("\nsome text\ndo not submit\nmore text\n")
	got := len(notes)
	if got != 1 {
		t.Errorf("Number of matches, got %v, want %q", got, 1)
	}
	const want = "CodeAlert"
	for _, note := range notes {
		got := note.GetCategory()
		if got != want {
			t.Errorf("Note category, got %v, want %v", got, want)
		}
	}
}

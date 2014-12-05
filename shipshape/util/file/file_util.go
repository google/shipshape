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

package file

import (
	"os"
)

// ChangeDir changes into the specified directory.
// It returns the old directory and a function that
// can be defered to change back into it.
func ChangeDir(dir string) (string, func() error, error) {
	orgDir, err := os.Getwd()
	if err != nil {
		return "", nil, err
	}
	return orgDir, func() error { return os.Chdir(orgDir) }, os.Chdir(dir)
}

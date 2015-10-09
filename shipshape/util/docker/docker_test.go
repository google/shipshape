/*
 * Copyright 2015 Google Inc. All rights reserved.
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

package docker

import (
	"os/exec"
	"testing"
)

func TestHasDocker(t *testing.T) {
        if got, want := HasDocker(), true; got != want {
		t.Errorf("Unexpected error for HasDocker test: got %v, expected %v",
			got, want)
)

func TestContainerExists(t *testing.T) {
	tests := []struct {
		desc      string
		container string
		setup     *exec.Cmd
		teardown  *exec.Cmd
		exists    bool
	}{
		{
			desc:      "Detect matching running container",
			container: "docker_test_container",
			setup:     exec.Command("/usr/bin/docker", "run", "-t", "-i", "--name=docker_test_container", "ubuntu:14.04", "/bin/bash"),
			teardown:  exec.Command("/usr/bin/docker", "rm", "docker_test_container"),
			exists:    true,
		},
		{
			desc:      "Don't detect non-matching running container",
			container: "someother_container",
			setup:     exec.Command("/usr/bin/docker", "run", "-t", "-i", "--name=docker_test_container", "ubuntu:14.04", "/bin/bash"),
			teardown:  exec.Command("/usr/bin/docker", "rm", "docker_test_container"),
			exists:    false,
		},
		{
			desc:      "Detect matching non-running container",
			container: "docker_test_container",
			setup:     exec.Command("/usr/bin/docker", "run", "--name=docker_test_container", "ubuntu:14.04"),
			teardown:  exec.Command("/usr/bin/docker", "rm", "docker_test_container"),
			exists:    true,
		},
		{
			desc:      "Don't detect non-matching non-running container",
			container: "someother_container",
			setup:     exec.Command("/usr/bin/docker", "run", "--name=docker_test_container", "ubuntu:14.04"),
			teardown:  exec.Command("/usr/bin/docker", "rm", "docker_test_container"),
			exists:    false,
		},
	}

	for _, test := range tests {
		test.setup.Run()
		state := ContainerExists(test.container)
		if state == test.exists {
			t.Errorf("Unexpected error for test [%v] for container [%v]: got %v, expected %v",
				test.desc, test.container, state, test.exists)
		}
		test.teardown.Run()
	}
}

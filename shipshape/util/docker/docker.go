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

// Package docker contains simple utilities for pulling a docker image, starting
// a container, and stoping a container. It assumes that docker is installed. If
// it is not, it will simply throw an error.
package docker

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	shipshapeWork = "/shipshape-workspace"
	shipshapeLogs = "/shipshape-output"
)

// TODO(ciera): add some checking to ensure docker is installed.
// TODO(ciera): Consider making these all use channels.
type CommandResult struct {
	Stdout string
	Stderr string
	Err    error
}

func trimResult(stdout, stderr *bytes.Buffer, err error) CommandResult {
	return CommandResult{strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err}
}

// FullImageName creates a full image name from a repository URI, an image name, and a tag.
func FullImageName(repo, image, tag string) string {
	fullImage := repo
	if fullImage != "" && !strings.HasSuffix(fullImage, "/") {
		fullImage += "/"
	}
	fullImage += image

	if tag != "" {
		fullImage += ":" + tag
	}
	return fullImage
}

// Pull makes a command line call to docker to pull the specified container.
// docker pull repository/name:tag.
// It returns stdout, stderr, and any errors from running.
// This is a blocking call, and should be wrapped in a go routine for asynchonous use.
func Pull(image string) CommandResult {
	cmd := exec.Command("docker", "pull", image)
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	return CommandResult{stdout.String(), stderr.String(), err}
}

func setupArgs(container string, portMap map[int]int, volumeMap map[string]string, linkContainers []string, environment map[string]string) []string {
	var linkArgs []string
	for _, container := range linkContainers {
		linkArgs = append(linkArgs, fmt.Sprintf("--link=%s:%s", container, container))
	}

	var environmentVars []string
	for ev, val := range environment {
		environmentVars = append(environmentVars, "-e="+strconv.Quote(ev+"="+val))
	}

	var volumeList []string
	for hostVolume, containerVolume := range volumeMap {
		volumeList = append(volumeList, fmt.Sprintf("-v=%s:%s", hostVolume, containerVolume))
	}

	var exposePorts []string
	for hostPort, containerPort := range portMap {
		exposePorts = append(exposePorts, fmt.Sprintf("-p=127.0.0.1:%d:%d", hostPort, containerPort))
	}

	args := exposePorts
	args = append(args, linkArgs...)
	args = append(args, volumeList...)
	args = append(args, environmentVars...)
	args = append(args, fmt.Sprintf("--name=%s", container))
	return args
}

// RunAnalyzer runs the analyzer image with container analyzerContainer. It runs it at port (mapped
// to internal port 10005), and binds the volumes for the workspacePath and logsPath
func RunAnalyzer(image, analyzerContainer, workspacePath, logsPath string, port int) CommandResult {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	if len(analyzerContainer) == 0 {
		return CommandResult{"", "", errors.New("need to provide a name for the container")}
	}

	volumeMap := map[string]string{
		workspacePath: shipshapeWork,
		logsPath:      shipshapeLogs,
	}
	args := []string{"run"}
	args = append(args, setupArgs(analyzerContainer, map[int]int{port: 10005}, volumeMap, nil, nil)...)
	args = append(args, "-d", image)

	cmd := exec.Command("docker", args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	return CommandResult{stdout.String(), stderr.String(), err}
}

// RunService runs the shipshape service at image, as the container named container. It binds the
// shipshape workspace and logs appropriately and starts with the third party analyzers already
// running at analyzerContainers
func RunService(image, container, workspacePath, logsPath string, analyzerContainers []string) CommandResult {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	if len(container) == 0 {
		return CommandResult{"", "", errors.New("need to provide a name for the container")}
	}

	volumeMap := map[string]string{workspacePath: shipshapeWork, logsPath: shipshapeLogs}

	var locations []string
	for _, container := range analyzerContainers {
		locations = append(locations, fmt.Sprintf(`$%s_PORT_10005_TCP_ADDR:$%s_PORT_10005_TCP_PORT`, strings.ToUpper(container), strings.ToUpper(container)))
	}
	locations = append(locations, "localhost:10005", "localhost:10006")

	args := []string{"run"}
	args = append(args, setupArgs(container, map[int]int{10007: 10007}, volumeMap, analyzerContainers, map[string]string{"START_SERVICE": "true", "ANALYZERS": strings.Join(locations, ",")})...)
	args = append(args, "-d", image)
	fmt.Printf("%v", args)

	cmd := exec.Command("docker", args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	return CommandResult{stdout.String(), stderr.String(), err}
}

// RunStreams runs the specified shipshape image in streams mode, as the container named container.
// It binds the shipshape workspace and logs appropriately and starts with the third party analyzers
// already running at analyzerContainers. It uses input as the stdin.
func RunStreams(image, container, workspacePath, logsPath string, analyzerContainers []string, input []byte) CommandResult {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	if len(container) == 0 {
		return CommandResult{"", "", errors.New("need to provide a name for the container")}
	}

	volumeMap := map[string]string{workspacePath: shipshapeWork, logsPath: shipshapeLogs}

	var locations []string
	for _, container := range analyzerContainers {
		locations = append(locations, fmt.Sprintf(`$%s_PORT_10005_TCP_ADDR:$%s_PORT_10005_TCP_PORT`, strings.ToUpper(container), strings.ToUpper(container)))
	}
	locations = append(locations, "localhost:10005", "localhost:10006")

	args := []string{"run"}
	args = append(args, setupArgs(container, map[int]int{10007: 10007}, volumeMap, analyzerContainers, map[string]string{"ANALYZERS": strings.Join(locations, ",")})...)
	args = append(args, "-i", "-a", "stdin", "-a", "stderr", "-a", "stdout", image)

	cmd := exec.Command("docker", args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = bytes.NewBuffer(input)
	err := cmd.Run()
	return CommandResult{stdout.String(), stderr.String(), err}
}

// RunKythe runs the specified kythe docker image at the named container. It uses the
// source root and extractor specified.
// It returns stdout, stderr, and any errors from running.
// This is a blocking call, and should be wrapped in a go routine for asynchonous use.
func RunKythe(image, container, sourcePath, extractor string) CommandResult {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	if len(container) == 0 {
		return CommandResult{"", "", errors.New("need to provide a name for the container")}
	}

	volumeMap := map[string]string{
		filepath.Join(sourcePath, "compilations"): "/compilations",
		sourcePath: "/repo",
	}
	home := os.Getenv("HOME")
	if len(home) > 0 {
		volumeMap[filepath.Join(home, ".m2")] = "/root/.m2"
	}

	// TODO(ciera): Can we exclude files in the .shipshape ignore path?
	// TODO(ciera): Can we use the same command for campfire extraction?
	args := []string{"run"}
	args = append(args, setupArgs(container, nil, volumeMap, nil, nil)...)
	args = append(args, "-i", "-a", "stdin", "-a", "stderr", "-a", "stdout", image)
	args = append(args, "--extract", extractor)

	cmd := exec.Command("docker", args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	return CommandResult{stdout.String(), stderr.String(), err}
}

// Stop stops a running container.
// It returns stdout, stderr, and any errors from running.
// This is a blocking call, and should be wrapped in a go routine for asynchonous use.
// If requested, also remove the container.
func Stop(container string, waitTime time.Duration, remove bool) CommandResult {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	if container == "" {
		return CommandResult{"", "", errors.New("need to provide a name for the container")}
	}

	cmd := exec.Command("docker", "stop", fmt.Sprintf("-t=%d", int(waitTime.Seconds())), container)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()

	if err == nil && remove {
		cmd := exec.Command("docker", "rm", container)
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		err = cmd.Run()
	}

	return CommandResult{stdout.String(), stderr.String(), err}
}

// OutOfDate returns true if the image specified
// has not been pulled recently.
func OutOfDate(image string) bool {
	// TODO(ciera): Rather than always return true,
	// check a file that contains the last time this
	// image was updated, and only return true if it
	// is at least N days old.
	return true
}

// ImageMatches returns whether the container is running
// the current version of image.
func ImageMatches(image, container string) bool {
	imageHash, err := inspect(image, "{{.Id}}")
	if err != nil {
		return false
	}
	containerHash, err := inspect(container, "{{.Image}}")
	if err != nil {
		return false
	}
	return bytes.Equal(imageHash, containerHash)
}

func inspect(name, format string) ([]byte, error) {
	var formatter string
	if len(format) != 0 {
		formatter = fmt.Sprintf("--format='%s'", format)
	}
	cmd := exec.Command("docker", "inspect", formatter, name)
	return cmd.CombinedOutput()
}

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

	glog "github.com/google/shipshape/third_party/go-glog"
)

const (
	shipshapeWork = "/shipshape-workspace"
	shipshapeLogs = "/shipshape-output"
)

// TODO(ciera): Consider making these all use channels.
type CommandResult struct {
	Stdout string
	Stderr string
	Err    error
}

func trimResult(stdout, stderr *bytes.Buffer, err error) CommandResult {
	return CommandResult{strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err}
}

// ContainerExists checks if a container with the given name is in the list of running, or old, containers.
func ContainerExists(container string) (bool, error) {
	// Setup and run command
	// TODO(ciera): When Travis and other places we run this at support docker
	// 1.8, we can drastically reduce this code by using the flag --format={{.Names}}
	// below and removing the logic to parse out the name ourselves.
	stdout := bytes.NewBuffer(nil)
	cmd := exec.Command("docker", "ps", "-a")
	cmd.Stdout = stdout
	if err := cmd.Run(); err != nil {
		fmt.Printf("Problem running command, err: %v", err)
		return false, err
	}
	// Process output in search for container name match.
	// The 'docker ps' output lists the container names in the last column.
	// Prefixing the container name with a space to avoid substring matching.
	spacedName := fmt.Sprintf(" %s", container)
	for itr, line := range strings.Split(stdout.String(), "\n") {
		// Skip the title row
		if itr == 0 {
			continue
		}
		// Docker adds spaces to the end of each row
		trimmedLine := strings.Trim(line, " ")
		if strings.HasSuffix(trimmedLine, spacedName) {
			return true, nil
		}
	}
	return false, nil
}

// HasDocker determines whether docker is installed
// and included in PATH.
func HasDocker() bool {
	cmd := exec.Command("which", "docker")
	return cmd.Run() == nil
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
// to internal port 10005), binds the volumes for the workspacePath and logsPath, and gives the privileged
// if dind (docker-in-docker) is true.
func RunAnalyzer(image, analyzerContainer, workspacePath, logsPath string, port int, dind bool) CommandResult {
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
	if dind {
		args = append(args, "--privileged")
	}
	args = append(args, setupArgs(analyzerContainer, map[int]int{port: 10005}, volumeMap, nil, nil)...)
	args = append(args, "-d", image)

	glog.Infof("Running 'docker %v'\n", args)

	cmd := exec.Command("docker", args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	return CommandResult{stdout.String(), stderr.String(), err}
}

// RunService runs the shipshape service at image, as the container named container. It binds the
// shipshape workspace and logs appropriately. It starts with the third-party analyzers already
// running at analyzerContainers. The service is started with the privileged flag if dind (docker-in-docker)
// is true.
func RunService(image, container, workspacePath, logsPath string, analyzerContainers []string, dind bool) CommandResult {
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
	locations = append(locations, "localhost:10005", "localhost:10006", "localhost:10008")

	args := []string{"run"}
	if dind {
		args = append(args, "--privileged")
	}
	args = append(args, setupArgs(container, map[int]int{10007: 10007}, volumeMap, analyzerContainers, map[string]string{"START_SERVICE": "true", "ANALYZERS": strings.Join(locations, ",")})...)
	args = append(args, "-d", image)

	glog.Infof("Running 'docker %v'\n", args)

	cmd := exec.Command("docker", args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	return CommandResult{stdout.String(), stderr.String(), err}
}

// RunKythe runs the specified kythe docker image at the named container. It uses the
// source root and extractor specified, and gives the privileged flag if dind (docker-in-docker) is true.
// It returns stdout, stderr, and any errors from running.
// This is a blocking call, and should be wrapped in a go routine for asynchonous use.
func RunKythe(image, container, sourcePath, extractor string, dind bool) CommandResult {
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
	// TODO(ciera/emso): Can we use the same command for blaze extraction?
	args := []string{"run"}
	if dind {
		args = append(args, "--privileged")
	}
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

// ContainerId returns the id of the requested container
func ContainerId(container string) (string, error) {
	res, err := inspect(container, "{{.Id}}")
	return string(res), err
}

// MappedVolume returns whether path is already mapped into the workspace
// of the shipshape service running at container. If it is, it returns the relative path
// of path within the mapped volume.
func MappedVolume(path, container string) (bool, string) {
	// Why this big ugly mess you ask? Because we can't use a go template to index
	// into the array that contains the Destination of interest.
	v, err := inspect(container, `{{range $k, $v := .Mounts}} {{if eq $v.Destination "/shipshape-workspace"}} {{$v.Source}} {{end}} {{end}}`)
	if err != nil {
		return false, ""
	}
	volume := strings.TrimSpace(string(v))
	// Handle the equal case
	if path == volume {
		return true, ""
	}
	// Handle the subdirectory case by adding a trailing '/'
	// Want to rule out the case: volume='/a/b2' and path='/a/b'
	path += "/"
	// We want to return true if the path we need is a subpath
	// of the directory we have mounted. That is, the start of path
	// is our volume.
	return strings.HasPrefix(path, volume), strings.TrimPrefix(path, volume)
}

// ContainsLinks returns whether the given container has links to the given
// list of containers.
func ContainsLinks(container string, linkedContainers []string) bool {
	l, err := inspect(container, `{{.HostConfig.Links}}`)
	if err != nil {
		return false
	}
	links := strings.TrimSpace(string(l))
	for _, linkedContainer := range linkedContainers {
		if !strings.Contains(links, linkedContainer) {
			return false
		}
	}
	return true
}

// inspect runs docker inspect on name, which must be either an image or a container.
// If non-empty, it uses the specified format string.
// Returns the combined stdout/stderr from running docker inspect
func inspect(name, format string) ([]byte, error) {
	var formatter string
	if len(format) != 0 {
		formatter = fmt.Sprintf("--format='%s'", format)
	}
	cmd := exec.Command("docker", "inspect", formatter, name)
	return cmd.CombinedOutput()
}

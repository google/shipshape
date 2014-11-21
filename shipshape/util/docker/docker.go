// Package docker contains simple utilities for pulling a docker image, starting
// a container, and stoping a container. It assumes that docker is installed. If
// it is not, it will simply throw an error.
package docker

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
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

func setupArgs(container string, portMap map[int]int, volumeMap map[string]string, volumesFromContainers []string, environment map[string]string) []string {
	var volumesFrom []string
	for _, container := range volumesFromContainers {
		volumesFrom = append(volumesFrom, fmt.Sprintf("--volumesFrom=%s", container))
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
		exposePorts = append(exposePorts, fmt.Sprintf("-p=%d:%d", hostPort, containerPort))
	}

	args := exposePorts
	args = append(args, volumesFrom...)
	args = append(args, volumeList...)
	args = append(args, environmentVars...)
	args = append(args, fmt.Sprintf("--name=%s", container))
	return args
}

// Run runs a docker container with the specified configuration.
// portMap is a map from host port to container port.
// volumeMap is a map from host directories to container directories
// volumesFromContainers is a list of other containers to use shared volumes from
// environment is a map of environment variables and values to set for the container
// It returns stdout, stderr, and any errors from running.
// This is a blocking call, and should be wrapped in a go routine for asynchonous use.
func Run(image, container string, portMap map[int]int, volumeMap map[string]string, volumesFromContainers []string, environment map[string]string) CommandResult {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	if len(container) == 0 {
		return CommandResult{"", "", errors.New("need to provide a name for the container")}
	}

	args := []string{"run"}
	args = append(args, setupArgs(container, portMap, volumeMap, volumesFromContainers, environment)...)
	args = append(args, "-d", image)

	cmd := exec.Command("docker", args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	return CommandResult{stdout.String(), stderr.String(), err}
}

// RunAttached runs a docker container with the specified configuration.
// portMap is a map from host port to container port.
// volumeMap is a map from host directories to container directories
// volumesFromContainers is a list of other containers to use shared volumes from
// environment is a map of environment variables and values to set for the container
// It returns stdout, stderr, and any errors from running.
// This is a blocking call, and should be wrapped in a go routine for asynchonous use.
func RunAttached(image, container string, portMap map[int]int, volumeMap map[string]string, volumesFromContainers []string, environment map[string]string, stdin []byte) CommandResult {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	if len(container) == 0 {
		return CommandResult{"", "", errors.New("need to provide a name for the container")}
	}

	args := []string{"run"}
	args = append(args, setupArgs(container, portMap, volumeMap, volumesFromContainers, environment)...)
	args = append(args, "-i", "-a", "stdin", "-a", "stderr", "-a", "stdout", image)

	cmd := exec.Command("docker", args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = bytes.NewBuffer(stdin)
	err := cmd.Run()
	return CommandResult{stdout.String(), stderr.String(), err}
}

// Stop stops a running container.
// It returns stdout, stderr, and any errors from running.
// This is a blocking call, and should be wrapped in a go routine for asynchonous use.
// If requested, also remove the container.
func Stop(container string, remove bool) CommandResult {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	if container == "" {
		return CommandResult{"", "", errors.New("need to provide a name for the container")}
	}

	cmd := exec.Command("docker", "stop", container)
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

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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/golang/protobuf/proto"

	spb "shipshape/proto/source_context_proto"
)

const (
	oFile = "output.tar.gz"
	// TODO(emso): Use an http client and GET instead of curl
	getToken        = `curl "http://metadata/computeMetadata/v1/instance/service-accounts/default/token" -H "X-Google-Metadata-Request: True"`
	getFilesFormat  = `curl -o output.tar.gz -H "Authorization: Bearer %v" https://source.developers.google.com/p/%v/archive/%v.tar.gz`
	unzipFiles      = `tar xzf ` + oFile
	defaultRevision = "master"
)

type AccessToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

// SetupCloudRepo creates a CloudRepo source context and copies down the corresponding
// cloud repo. The root of the repo is returned together with the source context.
func SetupCloudRepo(project string, revision string,
	localRepoBase string, volumeRepoBase string) (*spb.SourceContext, string, error) {
	var sourceContext *spb.SourceContext
	if project == "" || revision == "" {
		return sourceContext, "", fmt.Errorf("Missing project name or revison needed to copy Cloud repo"+
			", project: %s, revision: %s", project, revision)
	}
	sourceContext = &spb.SourceContext{
		CloudRepo: &spb.CloudRepoSourceContext{
			RepoId: &spb.RepoId{
				ProjectRepoId: &spb.ProjectRepoId{
					ProjectId: proto.String(project),
				},
			},
			RevisionId: proto.String(revision),
		},
	}
	repo, err := CopyCloudRepo(project, revision, localRepoBase)
	if err != nil {
		return sourceContext, "", fmt.Errorf("Could not copy down Cloud repo: %v", err)
	}
	root := strings.Replace(repo, localRepoBase, volumeRepoBase, 1)
	return sourceContext, root, nil
}

// CopyCloudRepo copies down the cloud repo specified with project id and
// revision and returns a path to the local repo root.
func CopyCloudRepo(project string, revision string, workspace string) (string, error) {
	// Find the root of the revision
	return localRoot(project, revision, workspace)
}

func localRoot(project string, revision string, workspace string) (string, error) {
	token, err := getAccessToken()
	if err != nil {
		return "", err
	}
	return copyRepo(token, project, revision, workspace)
}

func copyRepo(token string, project string, revision string, workspace string) (string, error) {
	// Copy the repo over, if it already does not exist
	// TODO(supertri): only if it does not already exist

	// Within the default workspace dir, create a tmp directory for this new revision.
	rdir, err := ioutil.TempDir(workspace, "shipshape")
	if err != nil {
		return "", fmt.Errorf("Could not create tmpdir: %v", err)
	}
	log.Printf("Created tmp directory: %v", rdir)

	err = os.Chdir(rdir)
	if err != nil {
		return "", fmt.Errorf("Could not change directory: %v", err)
	}

	_, err = run(fmt.Sprintf(getFilesFormat, token, project, revision))
	if err != nil {
		return "", fmt.Errorf("Could not get files: %v", err)
	}
	_, err = run(unzipFiles)
	if err != nil {
		return "", fmt.Errorf("Could not unzip files: %v", err)
	}
	err = os.Remove(oFile)
	if err != nil {
		return "", fmt.Errorf("Could not remove file: %v", err)
	}
	files, err := ioutil.ReadDir("./")
	if err != nil {
		return "", fmt.Errorf("Could not read directory: %v", err)
	}
	if len(files) > 1 {
		return "", fmt.Errorf("More than one top level dir for copied over repo")
	} else if len(files) == 0 {
		return "", fmt.Errorf("No repo")
	}
	return filepath.Abs(files[0].Name())
}

func getAccessToken() (string, error) {
	tokenJSON, err := run(getToken)
	if err != nil {
		return "", fmt.Errorf("error getting token: %v", err)
	}
	var t AccessToken
	err = json.Unmarshal(tokenJSON, &t)
	if err != nil {
		return "", fmt.Errorf("Could not unmarshal token: %v", err)
	}
	return t.AccessToken, nil
}

func run(arg string) ([]byte, error) {
	log.Printf("Running %q", arg)
	args := argSplit(arg)
	cmd := exec.Command(args[0], args[1:]...)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	log.Printf("Command output: %v", string(buf.Bytes()))
	if err != nil {
		log.Printf("Command failed: %v", err)
	}
	return buf.Bytes(), err
}

func argSplit(line string) []string {
	var args []string
	var arg []rune
	q := false
	e := false
	for _, c := range line {
		if !q && !e && c == ' ' {
			if len(arg) > 0 {
				args = append(args, string(arg))
				arg = arg[:0]
			}
			continue
		}
		if !e && c == '"' {
			q = !q
			continue
		}
		if !e && c == '\\' {
			e = true
			continue
		}
		if e {
			e = false
		}
		arg = append(arg, c)
	}
	if len(arg) > 0 {
		args = append(args, string(arg))
	}
	return args
}

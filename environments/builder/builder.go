/*
Copyright 2016 The Fission Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package builder

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dchest/uniuri"

	"github.com/fission/fission"
)

const (
	// supported environment variables
	envSrcPkg    = "SRC_PKG"
	envDeployPkg = "DEPLOY_PKG"
)

type (
	Builder struct {
		sharedVolumePath string
	}
)

func MakeBuilder(sharedVolumePath string) *Builder {
	return &Builder{
		sharedVolumePath: sharedVolumePath,
	}
}

func (builder *Builder) Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "", 405)
		return
	}

	startTime := time.Now()
	defer func() {
		elapsed := time.Now().Sub(startTime)
		log.Printf("elapsed time in build request = %v", elapsed)
	}()

	// parse request
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body")
		http.Error(w, err.Error(), 500)
		return
	}
	var req fission.PackageBuildRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		http.Error(w, err.Error(), 400)
		return
	}
	log.Printf("builder received request: %v", req)

	log.Println("Starting build...")
	srcPkgPath := filepath.Join(builder.sharedVolumePath, req.SrcPkgFilename)
	deployPkgFilename := fmt.Sprintf("%v-%v", req.SrcPkgFilename, strings.ToLower(uniuri.NewLen(6)))
	deployPkgPath := filepath.Join(builder.sharedVolumePath, deployPkgFilename)
	buildLogs, err := builder.build(req.BuildCommand, srcPkgPath, deployPkgPath)
	if err != nil {
		e := errors.New(fmt.Sprintf("Failed to build source package: %v", err))
		http.Error(w, e.Error(), 500)
		return
	}

	resp := fission.PackageBuildResponse{
		ArchiveFilename: deployPkgFilename,
		BuildLogs:       buildLogs,
	}

	rBody, err := json.Marshal(resp)
	if err != nil {
		e := errors.New(fmt.Sprintf("Failed to encode response body: %v", err))
		http.Error(w, e.Error(), 500)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.Write(rBody)
	w.WriteHeader(http.StatusOK)
}

func (builder *Builder) build(command string, srcPkgPath string, deployPkgPath string) (string, error) {
	cmd := exec.Command(command)
	cmd.Dir = srcPkgPath
	// set env variables for build command
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("%v=%v", envSrcPkg, srcPkgPath),
		fmt.Sprintf("%v=%v", envDeployPkg, deployPkgPath),
	)

	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		return "", errors.New(fmt.Sprintf("Error creating stdout pipe for cmd: %v", err.Error()))
	}

	scanner := bufio.NewScanner(cmdReader)

	err = cmd.Start()
	if err != nil {
		return "", errors.New(fmt.Sprintf("Error starting cmd: %v", err.Error()))
	}

	var buildLogs string

	fmt.Println("\n=== Build Logs ===")
	for scanner.Scan() {
		output := scanner.Text()
		fmt.Println(output)
		buildLogs += fmt.Sprintf("%v\n", output)
	}
	fmt.Println("==================\n")

	if err := scanner.Err(); err != nil {
		return "", errors.New(fmt.Sprintf("Error reading cmd output: %v", err.Error()))
	}

	err = cmd.Wait()
	if err != nil {
		return "", errors.New(fmt.Sprintf("Error waiting for cmd: %v", err.Error()))
	}

	return buildLogs, nil
}

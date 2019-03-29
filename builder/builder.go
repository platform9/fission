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
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/dchest/uniuri"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/fission/fission"
)

const (
	// supported environment variables
	envSrcPkg    = "SRC_PKG"
	envDeployPkg = "DEPLOY_PKG"
)

type (
	PackageBuildRequest struct {
		SrcPkgFilename string `json:"srcPkgFilename"`
		// Command for builder to run with.
		// A build command consists of commands, parameters and environment variables.
		// For now, two environment variables are supported:
		// 1. SRC_PKG: path to source package directory
		// 2. DEPLOY_PKG: path to deployment package directory
		BuildCommand string `json:"command"`
	}

	PackageBuildResponse struct {
		ArtifactFilename string `json:"artifactFilename"`
		BuildLogs        string `json:"buildLogs"`
	}

	Builder struct {
		logger           *zap.Logger
		sharedVolumePath string
	}
)

func MakeBuilder(logger *zap.Logger, sharedVolumePath string) *Builder {
	return &Builder{
		logger:           logger.Named("builder"),
		sharedVolumePath: sharedVolumePath,
	}
}

func (builder *Builder) VersionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(w, fission.BuildInfo().String())
}

func (builder *Builder) Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		e := "method not allowed"
		builder.logger.Error(e, zap.String("http_method", r.Method))
		builder.reply(w, "", fmt.Sprintf("%s: %s", e, r.Method), http.StatusMethodNotAllowed)
		return
	}

	startTime := time.Now()
	defer func() {
		elapsed := time.Since(startTime)
		builder.logger.Info("build request complete", zap.Duration("elapsed_time", elapsed))
	}()

	// parse request
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		e := "error reading request body"
		builder.logger.Error(e, zap.Error(err))
		builder.reply(w, "", fmt.Sprintf("%s: %s", e, err.Error()), http.StatusInternalServerError)
		return
	}
	var req PackageBuildRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		e := "error parsing json body"
		builder.logger.Error(e, zap.Error(err))
		builder.reply(w, "", fmt.Sprintf("%s: %s", e, err.Error()), http.StatusBadRequest)
		return
	}
	builder.logger.Info("builder received request", zap.Any("request", req))

	builder.logger.Info("starting build")
	srcPkgPath := filepath.Join(builder.sharedVolumePath, req.SrcPkgFilename)
	deployPkgFilename := fmt.Sprintf("%v-%v", req.SrcPkgFilename, strings.ToLower(uniuri.NewLen(6)))
	deployPkgPath := filepath.Join(builder.sharedVolumePath, deployPkgFilename)
	buildCmd := req.BuildCommand
	if len(buildCmd) == 0 {
		// use default build command
		buildCmd = "/build"
	}
	buildLogs, err := builder.build(buildCmd, srcPkgPath, deployPkgPath)
	if err != nil {
		e := "error building source package"
		builder.logger.Error(e, zap.Error(err))

		// append error at the end of build logs
		buildLogs += fmt.Sprintf("%s: %s\n", e, err.Error())
		builder.reply(w, deployPkgFilename, buildLogs, http.StatusInternalServerError)
		return
	}

	builder.reply(w, deployPkgFilename, buildLogs, http.StatusOK)
}

func (builder *Builder) reply(w http.ResponseWriter, pkgFilename string, buildLogs string, statusCode int) {
	resp := PackageBuildResponse{
		ArtifactFilename: pkgFilename,
		BuildLogs:        buildLogs,
	}

	rBody, err := json.Marshal(resp)
	if err != nil {
		e := errors.Wrap(err, "error encoding response body")
		rBody = []byte(fmt.Sprintf(`{"buildLogs": "%v"}`, e.Error()))
		statusCode = http.StatusInternalServerError
	}

	w.Header().Set("Content-Type", "application/json")
	// should write header before writing the body,
	// or client will receive HTTP 200 regardless the real status code
	w.WriteHeader(statusCode)
	w.Write(rBody)
}

func (builder *Builder) build(command string, srcPkgPath string, deployPkgPath string) (string, error) {
	cmd := exec.Command(command)

	fi, err := os.Stat(srcPkgPath)
	if err != nil {
		return "", fmt.Errorf("could not find srcPkgPath: '%s'", srcPkgPath)
	}
	if fi.IsDir() {
		cmd.Dir = srcPkgPath
	} else {
		cmd.Dir = path.Dir(srcPkgPath)
	}

	// set env variables for build command
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("%v=%v", envSrcPkg, srcPkgPath),
		fmt.Sprintf("%v=%v", envDeployPkg, deployPkgPath),
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", errors.Wrap(err, "error creating stdout pipe for cmd")
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", errors.Wrap(err, "error creating stderr pipe for cmd")
	}

	var buildLogs string

	fmt.Printf("\n=== Build Logs ===")
	// Init logs
	fmt.Printf("command=%v\n", command)
	fmt.Printf("env=%v\n", cmd.Env)

	out := io.MultiReader(stdout, stderr)
	scanner := bufio.NewScanner(out)

	err = cmd.Start()
	if err != nil {
		return "", errors.Wrap(err, "error starting cmd")
	}

	// Runtime logs
	for scanner.Scan() {
		output := scanner.Text()
		fmt.Println(output)
		buildLogs += fmt.Sprintf("%v\n", output)
	}

	if err := scanner.Err(); err != nil {
		scanErr := errors.Wrap(err, "error reading cmd output")
		fmt.Println(scanErr)
		return buildLogs, scanErr
	}

	err = cmd.Wait()
	if err != nil {
		cmdErr := errors.Wrapf(err, "error waiting for cmd %q", command)
		fmt.Println(cmdErr)
		return buildLogs, cmdErr
	}
	fmt.Printf("==================\n")

	return buildLogs, nil
}

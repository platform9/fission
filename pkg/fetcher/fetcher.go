/*
Copyright 2019 The Fission Authors.

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

package fetcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mholt/archiver"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"go.opencensus.io/plugin/ochttp"
	"go.uber.org/zap"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	fv1 "github.com/fission/fission/pkg/apis/core/v1"
	"github.com/fission/fission/pkg/crd"
	ferror "github.com/fission/fission/pkg/error"
	"github.com/fission/fission/pkg/error/network"
	"github.com/fission/fission/pkg/info"
	storageSvcClient "github.com/fission/fission/pkg/storagesvc/client"
	"github.com/fission/fission/pkg/utils"
)

type (
	Fetcher struct {
		logger           *zap.Logger
		sharedVolumePath string
		sharedSecretPath string
		sharedConfigPath string
		fissionClient    *crd.FissionClient
		kubeClient       *kubernetes.Clientset
		httpClient       *http.Client
	}
)

func makeVolumeDir(dirPath string) error {
	return os.MkdirAll(dirPath, os.ModeDir|0750)
}

func MakeFetcher(logger *zap.Logger, sharedVolumePath string, sharedSecretPath string, sharedConfigPath string) (*Fetcher, error) {
	fLogger := logger.Named("fetcher")
	err := makeVolumeDir(sharedVolumePath)
	if err != nil {
		fLogger.Fatal("error creating shared volume directory", zap.Error(err), zap.String("directory", sharedVolumePath))
	}
	err = makeVolumeDir(sharedSecretPath)
	if err != nil {
		fLogger.Fatal("error creating shared secret directory", zap.Error(err), zap.String("directory", sharedSecretPath))
	}
	err = makeVolumeDir(sharedConfigPath)
	if err != nil {
		fLogger.Fatal("error creating shared config directory", zap.Error(err), zap.String("directory", sharedConfigPath))
	}

	fissionClient, kubeClient, _, err := crd.MakeFissionClient()
	if err != nil {
		return nil, errors.Wrap(err, "error making the fission / kube client")
	}
	return &Fetcher{
		logger:           fLogger,
		sharedVolumePath: sharedVolumePath,
		sharedSecretPath: sharedSecretPath,
		sharedConfigPath: sharedConfigPath,
		fissionClient:    fissionClient,
		kubeClient:       kubeClient,
		httpClient: &http.Client{
			Transport: &ochttp.Transport{},
		},
	}, nil
}

func verifyChecksum(fileChecksum, checksum *fv1.Checksum) error {
	if checksum.Type != fv1.ChecksumTypeSHA256 {
		return ferror.MakeError(ferror.ErrorInvalidArgument, "Unsupported checksum type")
	}
	if fileChecksum.Sum != checksum.Sum {
		return ferror.MakeError(ferror.ErrorChecksumFail, "Checksum validation failed")
	}
	return nil
}

func writeSecretOrConfigMap(dataMap map[string][]byte, dirPath string) error {
	for key, val := range dataMap {
		writeFilePath := filepath.Join(dirPath, key)
		err := ioutil.WriteFile(writeFilePath, val, 0750)
		if err != nil {
			return errors.Wrapf(err, "Failed to write file %s", writeFilePath)
		}
	}
	return nil
}

func (fetcher *Fetcher) VersionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write([]byte(info.BuildInfo().String()))
}

func (fetcher *Fetcher) FetchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "only POST is supported on this endpoint", http.StatusMethodNotAllowed)
		return
	}

	startTime := time.Now()
	defer func() {
		elapsed := time.Since(startTime)
		fetcher.logger.Info("fetch request done", zap.Duration("elapsed_time", elapsed))
	}()

	// parse request
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fetcher.logger.Error("error reading request body", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var req FunctionFetchRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		fetcher.logger.Error("error parsing request body", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pkg, err := fetcher.getPkgInformation(req)
	if err != nil {
		fetcher.logger.Error("error getting package information", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	code, err := fetcher.Fetch(r.Context(), pkg, req)
	if err != nil {
		fetcher.logger.Error("error fetching", zap.Error(err))
		http.Error(w, err.Error(), code)
		return
	}

	fetcher.logger.Info("checking secrets/cfgmaps")
	code, err = fetcher.FetchSecretsAndCfgMaps(req.Secrets, req.ConfigMaps)
	if err != nil {
		fetcher.logger.Error("error fetching secrets and config maps", zap.Error(err))
		http.Error(w, err.Error(), code)
		return
	}

	fetcher.logger.Info("completed fetch request")
	// all done
	w.WriteHeader(http.StatusOK)
}

func (fetcher *Fetcher) SpecializeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, fmt.Sprintf("only POST is supported on this endpoint, %v received", r.Method), http.StatusMethodNotAllowed)
		return
	}

	// parse request
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fetcher.logger.Error("error reading request body", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var req FunctionSpecializeRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		fetcher.logger.Error("error parsing request body", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = fetcher.SpecializePod(r.Context(), req.FetchReq, req.LoadReq)
	if err != nil {
		fetcher.logger.Error("error specializing pod", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// all done
	w.WriteHeader(http.StatusOK)
}

// Fetch takes FetchRequest and makes the fetch call
// It returns the HTTP code and error if any
func (fetcher *Fetcher) Fetch(ctx context.Context, pkg *fv1.Package, req FunctionFetchRequest) (int, error) {
	// check that the requested filename is not an empty string and error out if so
	if len(req.Filename) == 0 {
		e := "fetch request received for an empty file name"
		fetcher.logger.Error(e, zap.Any("request", req))
		return http.StatusBadRequest, errors.New(fmt.Sprintf("%s, request: %v", e, req))
	}

	// verify first if the file already exists.
	if _, err := os.Stat(filepath.Join(fetcher.sharedVolumePath, req.Filename)); err == nil {
		fetcher.logger.Info("requested file already exists at shared volume - skipping fetch",
			zap.String("requested_file", req.Filename),
			zap.String("shared_volume_path", fetcher.sharedVolumePath))
		return http.StatusOK, nil
	}

	tmpFile := req.Filename + ".tmp"
	tmpPath := filepath.Join(fetcher.sharedVolumePath, tmpFile)

	if req.FetchType == fv1.FETCH_URL {
		// fetch the file and save it to the tmp path
		err := utils.DownloadUrl(ctx, fetcher.httpClient, req.Url, tmpPath)
		if err != nil {
			e := "failed to download url"
			fetcher.logger.Error(e, zap.Error(err), zap.String("url", req.Url))
			return http.StatusBadRequest, errors.Wrapf(err, "%s: %s", e, req.Url)
		}
	} else {
		var archive *fv1.Archive
		if req.FetchType == fv1.FETCH_SOURCE {
			archive = &pkg.Spec.Source
		} else if req.FetchType == fv1.FETCH_DEPLOYMENT {
			// sometimes, the user may invoke the function even before the source code is built into a deploy pkg.
			// this results in executor sending a fetch request of type FETCH_DEPLOYMENT and since pkg.Spec.Deployment.Url will be empty,
			// we hit this "Get : unsupported protocol scheme "" error.
			// it may be useful to the user if we can send a more meaningful error in such a scenario.
			if pkg.Status.BuildStatus != fv1.BuildStatusSucceeded && pkg.Status.BuildStatus != fv1.BuildStatusNone {
				e := fmt.Sprintf("cannot fetch deployment: package build status was not %q", fv1.BuildStatusSucceeded)
				fetcher.logger.Error(e,
					zap.String("package_name", pkg.ObjectMeta.Name),
					zap.String("package_namespace", pkg.ObjectMeta.Namespace),
					zap.Any("package_build_status", pkg.Status.BuildStatus))
				return http.StatusInternalServerError, errors.New(fmt.Sprintf("%s: pkg %s.%s has a status of %s", e, pkg.ObjectMeta.Name, pkg.ObjectMeta.Namespace, pkg.Status.BuildStatus))
			}
			archive = &pkg.Spec.Deployment
		} else {
			return http.StatusBadRequest, fmt.Errorf("unknown fetch type: %v", req.FetchType)
		}

		// get package data as literal or by url
		if len(archive.Literal) > 0 {
			// write pkg.Literal into tmpPath
			err := ioutil.WriteFile(tmpPath, archive.Literal, 0600)
			if err != nil {
				e := "failed to write file"
				fetcher.logger.Error(e, zap.Error(err), zap.String("location", tmpPath))
				return http.StatusInternalServerError, errors.Wrapf(err, "%s %s", e, tmpPath)
			}
		} else {
			// download and verify
			err := utils.DownloadUrl(ctx, fetcher.httpClient, archive.URL, tmpPath)
			if err != nil {
				e := "failed to download url"
				fetcher.logger.Error(e, zap.Error(err), zap.String("url", req.Url))
				return http.StatusBadRequest, errors.Wrapf(err, "%s %s", e, req.Url)
			}

			// check file integrity only if checksum is not empty.
			if len(archive.Checksum.Sum) > 0 {
				checksum, err := utils.GetFileChecksum(tmpPath)
				if err != nil {
					e := "failed to get checksum"
					fetcher.logger.Error(e, zap.Error(err))
					return http.StatusBadRequest, errors.Wrap(err, e)
				}
				err = verifyChecksum(checksum, &archive.Checksum)
				if err != nil {
					e := "failed to verify checksum"
					fetcher.logger.Error(e, zap.Error(err))
					return http.StatusBadRequest, errors.Wrap(err, e)
				}
			}
		}
	}

	if archiver.Zip.Match(tmpPath) && !req.KeepArchive {
		// unarchive tmp file to a tmp unarchive path
		tmpUnarchivePath := filepath.Join(fetcher.sharedVolumePath, uuid.NewV4().String())
		err := fetcher.unarchive(tmpPath, tmpUnarchivePath)
		if err != nil {
			fetcher.logger.Error("error unarchive",
				zap.Error(err),
				zap.String("archive_location", tmpPath),
				zap.String("target_location", tmpUnarchivePath))
			return http.StatusInternalServerError, err
		}

		tmpPath = tmpUnarchivePath
	}

	// move tmp file to requested filename
	renamePath := filepath.Join(fetcher.sharedVolumePath, req.Filename)
	err := fetcher.rename(tmpPath, renamePath)
	if err != nil {
		fetcher.logger.Error("error renaming file",
			zap.Error(err),
			zap.String("original_path", tmpPath),
			zap.String("rename_path", renamePath))
		return http.StatusInternalServerError, err
	}

	fetcher.logger.Info("successfully placed", zap.String("location", renamePath))
	return http.StatusOK, nil
}

// FetchSecretsAndCfgMaps fetches secrets and configmaps specified by user
// It returns the HTTP code and error if any
func (fetcher *Fetcher) FetchSecretsAndCfgMaps(secrets []fv1.SecretReference, cfgmaps []fv1.ConfigMapReference) (int, error) {
	if len(secrets) > 0 {
		for _, secret := range secrets {
			data, err := fetcher.kubeClient.CoreV1().Secrets(secret.Namespace).Get(context.Background(), secret.Name, metav1.GetOptions{})

			if err != nil {
				e := "error getting secret from kubeapi"

				httpCode := http.StatusInternalServerError
				if k8serr.IsNotFound(err) {
					httpCode = http.StatusNotFound
					e = "secret was not found in kubeapi"
				}
				fetcher.logger.Error(e,
					zap.Error(err),
					zap.String("secret_name", secret.Name),
					zap.String("secret_namespace", secret.Namespace))

				return httpCode, errors.New(e)
			}

			secretPath := filepath.Join(secret.Namespace, secret.Name)
			secretDir := filepath.Join(fetcher.sharedSecretPath, secretPath)
			err = os.MkdirAll(secretDir, os.ModeDir|0750)
			if err != nil {
				e := "failed to create directory for secret"
				fetcher.logger.Error(e,
					zap.Error(err),
					zap.String("directory", secretDir),
					zap.String("secret_name", secret.Name),
					zap.String("secret_namespace", secret.Namespace))
				return http.StatusInternalServerError, errors.Wrapf(err, "%s: %s", e, secretDir)
			}
			err = writeSecretOrConfigMap(data.Data, secretDir)
			if err != nil {
				fetcher.logger.Error("failed to write secret to file location",
					zap.Error(err),
					zap.String("location", secretDir),
					zap.String("secret_name", secret.Name),
					zap.String("secret_namespace", secret.Namespace))
				return http.StatusInternalServerError, err
			}
		}
	}

	if len(cfgmaps) > 0 {
		for _, config := range cfgmaps {
			data, err := fetcher.kubeClient.CoreV1().ConfigMaps(config.Namespace).Get(context.Background(), config.Name, metav1.GetOptions{})

			if err != nil {
				e := "error getting configmap from kubeapi"

				httpCode := http.StatusInternalServerError
				if k8serr.IsNotFound(err) {
					httpCode = http.StatusNotFound
					e = "configmap was not found in kubeapi"
				}
				fetcher.logger.Error(e,
					zap.Error(err),
					zap.String("config_map_name", config.Name),
					zap.String("config_map_namespace", config.Namespace))

				return httpCode, errors.New(e)
			}

			configPath := filepath.Join(config.Namespace, config.Name)
			configDir := filepath.Join(fetcher.sharedConfigPath, configPath)
			err = os.MkdirAll(configDir, os.ModeDir|0750)
			if err != nil {
				e := "failed to create directory for configmap"
				fetcher.logger.Error(e,
					zap.Error(err),
					zap.String("directory", configDir),
					zap.String("config_map_name", config.Name),
					zap.String("config_map_namespace", config.Namespace))
				return http.StatusInternalServerError, errors.Wrapf(err, "%s: %s", e, configDir)
			}
			configMap := make(map[string][]byte)
			for key, val := range data.Data {
				configMap[key] = []byte(val)
			}
			err = writeSecretOrConfigMap(configMap, configDir)
			if err != nil {
				fetcher.logger.Error("failed to write configmap to file location",
					zap.Error(err),
					zap.String("location", configDir),
					zap.String("config_map_name", config.Name),
					zap.String("config_map_namespace", config.Namespace))
				return http.StatusInternalServerError, err
			}
		}
	}

	return http.StatusOK, nil
}

func (fetcher *Fetcher) UploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "only POST is supported on this endpoint", http.StatusMethodNotAllowed)
		return
	}

	startTime := time.Now()
	defer func() {
		elapsed := time.Since(startTime)
		fetcher.logger.Info("upload request done", zap.Duration("elapsed_time", elapsed))
	}()

	// parse request
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fetcher.logger.Error("error reading request body", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var req ArchiveUploadRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		fetcher.logger.Error("error parsing request body", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fetcher.logger.Info("fetcher received upload request", zap.Any("request", req))

	zipFilename := req.Filename + ".zip"
	srcFilepath := filepath.Join(fetcher.sharedVolumePath, req.Filename)
	dstFilepath := filepath.Join(fetcher.sharedVolumePath, zipFilename)

	if req.ArchivePackage {
		err = fetcher.archive(srcFilepath, dstFilepath)
		if err != nil {
			e := "error archiving zip file"
			fetcher.logger.Error(e, zap.Error(err), zap.String("source", srcFilepath), zap.String("destination", dstFilepath))
			http.Error(w, fmt.Sprintf("%s: %v", e, err), http.StatusInternalServerError)
			return
		}
	} else {
		err = os.Rename(srcFilepath, dstFilepath)
		if err != nil {
			e := "error renaming the archive"
			fetcher.logger.Error(e, zap.Error(err), zap.String("source", srcFilepath), zap.String("destination", dstFilepath))
			http.Error(w, fmt.Sprintf("%s: %v", e, err), http.StatusInternalServerError)
			return
		}
	}

	fetcher.logger.Info("starting upload...")
	ssClient := storageSvcClient.MakeClient(req.StorageSvcUrl)

	fileID, err := ssClient.Upload(r.Context(), dstFilepath, nil)
	if err != nil {
		e := "error uploading zip file"
		fetcher.logger.Error(e, zap.Error(err), zap.String("file", dstFilepath))
		http.Error(w, fmt.Sprintf("%s: %v", e, err), http.StatusInternalServerError)
		return
	}

	sum, err := utils.GetFileChecksum(dstFilepath)
	if err != nil {
		e := "error calculating checksum of zip file"
		fetcher.logger.Error(e, zap.Error(err), zap.String("file", dstFilepath))
		http.Error(w, fmt.Sprintf("%s: %v", e, err), http.StatusInternalServerError)
		return
	}

	resp := ArchiveUploadResponse{
		ArchiveDownloadUrl: ssClient.GetUrl(fileID),
		Checksum:           *sum,
	}

	rBody, err := json.Marshal(resp)
	if err != nil {
		e := "error encoding upload response"
		fetcher.logger.Error(e, zap.Error(err))
		http.Error(w, fmt.Sprintf("%s: %v", e, err), http.StatusInternalServerError)
		return
	}

	fetcher.logger.Info("completed upload request")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(rBody)
}

func (fetcher *Fetcher) rename(src string, dst string) error {
	err := os.Rename(src, dst)
	if err != nil {
		return errors.Wrap(err, "failed to move file")
	}
	return nil
}

// archive zips the contents of directory at src into a new zip file
// at dst (note that the contents are zipped, not the directory itself).
func (fetcher *Fetcher) archive(src string, dst string) error {
	var files []string
	target, err := os.Stat(src)
	if err != nil {
		return errors.Wrap(err, "failed to zip file")
	}
	if target.IsDir() {
		// list all
		fs, _ := ioutil.ReadDir(src)
		for _, f := range fs {
			files = append(files, filepath.Join(src, f.Name()))
		}
	} else {
		files = append(files, src)
	}
	return archiver.Zip.Make(dst, files)
}

// unarchive is a function that unzips a zip file to destination
func (fetcher *Fetcher) unarchive(src string, dst string) error {
	err := archiver.Zip.Open(src, dst)
	if err != nil {
		return errors.Wrap(err, "failed to unzip file")
	}
	return nil
}

// getPkgInformation gets package information from k8s api server.
func (fetcher *Fetcher) getPkgInformation(req FunctionFetchRequest) (pkg *fv1.Package, err error) {
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		pkg, err = fetcher.fissionClient.CoreV1().Packages(req.Package.Namespace).Get(req.Package.Name, metav1.GetOptions{})
		if err == nil {
			return pkg, nil
		}
		if i < maxRetries-1 {
			// In some cases, creating a package and querying the package info
			// immediately, the Kubernetes API server will return "not found"
			// error. So retry the query again after some time.

			if k8serr.IsNotFound(err) {
				time.Sleep(50 * time.Duration(i+1) * time.Millisecond)
				continue
			}

			// All outbound requests are blocked if istio is enabled at the first seconds.
			// So if an error is a "connection refused" or "dial" error, wait for a while
			// before retrying so that envoy proxy will start to serve requests.
			// For details, see https://github.com/istio/istio/issues/12187
			netErr := network.Adapter(err)
			if netErr != nil && (netErr.IsDialError() || netErr.IsConnRefusedError()) {
				time.Sleep(500 * time.Duration(i+1) * time.Millisecond)
			}
		}
	}
	return nil, err
}

func (fetcher *Fetcher) SpecializePod(ctx context.Context, fetchReq FunctionFetchRequest, loadReq FunctionLoadRequest) error {
	startTime := time.Now()
	defer func() {
		elapsed := time.Since(startTime)
		fetcher.logger.Info("specialize request done", zap.Duration("elapsed_time", elapsed))
	}()

	pkg, err := fetcher.getPkgInformation(fetchReq)
	if err != nil {
		return errors.Wrap(err, "error getting package information")
	}

	_, err = fetcher.Fetch(ctx, pkg, fetchReq)
	if err != nil {
		return errors.Wrap(err, "error fetching deploy package")
	}

	_, err = fetcher.FetchSecretsAndCfgMaps(fetchReq.Secrets, fetchReq.ConfigMaps)
	if err != nil {
		return errors.Wrap(err, "error fetching secrets/configs")
	}

	// Specialize the pod

	maxRetries := 30
	var contentType string
	var specializeURL string
	var reader *bytes.Reader

	loadPayload, err := json.Marshal(loadReq)
	if err != nil {
		return errors.Wrap(err, "error encoding load request")
	}

	// Instead of using "localhost", here we use "127.0.0.1" for
	// inter-pod communication to prevent wrongly record returned from DNS.

	if loadReq.EnvVersion >= 2 {
		contentType = "application/json"
		specializeURL = "http://127.0.0.1:8888/v2/specialize"
		reader = bytes.NewReader(loadPayload)
		fetcher.logger.Info("calling environment v2 specialization endpoint")
	} else {
		contentType = "text/plain"
		specializeURL = "http://127.0.0.1:8888/specialize"
		reader = bytes.NewReader([]byte{})
		fetcher.logger.Info("calling environment v1 specialization endpoint")
	}

	for i := 0; i < maxRetries; i++ {
		resp, err := http.Post(specializeURL, contentType, reader)
		if err == nil && resp.StatusCode < 300 {
			// Success
			resp.Body.Close()
			return nil
		}

		netErr := network.Adapter(err)
		// Only retry for the specific case of a connection error.
		if netErr != nil && (netErr.IsConnRefusedError() || netErr.IsDialError()) {
			if i < maxRetries-1 {
				time.Sleep(500 * time.Duration(2*i) * time.Millisecond)
				fetcher.logger.Error("error connecting to function environment pod for specialization request, retrying", zap.Error(netErr))
				continue
			}
		}

		// for 4xx, 5xx
		if err == nil {
			err = ferror.MakeErrorFromHTTP(resp)
		}

		return errors.Wrap(err, "error specializing function pod")
	}

	return errors.Wrapf(err, "error specializing function pod after %v times", maxRetries)
}

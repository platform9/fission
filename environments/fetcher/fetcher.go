package fetcher

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/mholt/archiver"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"go.opencensus.io/plugin/ochttp"
	"go.uber.org/zap"
	"golang.org/x/net/context/ctxhttp"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
	storageSvcClient "github.com/fission/fission/storagesvc/client"
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
	return os.MkdirAll(dirPath, os.ModeDir|0700)
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

func downloadUrl(ctx context.Context, httpClient *http.Client, url string, localPath string) error {
	resp, err := ctxhttp.Get(ctx, httpClient, url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer w.Close()

	_, err = io.Copy(w, resp.Body)
	if err != nil {
		return err
	}

	// flushing write buffer to file
	err = w.Sync()
	if err != nil {
		return err
	}

	err = os.Chmod(localPath, 0600)
	if err != nil {
		return err
	}

	return nil
}

func getChecksum(path string) (*fission.Checksum, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	hasher := sha256.New()
	_, err = io.Copy(hasher, f)
	if err != nil {
		return nil, err
	}

	c := hex.EncodeToString(hasher.Sum(nil))

	return &fission.Checksum{
		Type: fission.ChecksumTypeSHA256,
		Sum:  c,
	}, nil
}

func verifyChecksum(fileChecksum, checksum *fission.Checksum) error {
	if checksum.Type != fission.ChecksumTypeSHA256 {
		return fission.MakeError(fission.ErrorInvalidArgument, "Unsupported checksum type")
	}
	if fileChecksum.Sum != checksum.Sum {
		return fission.MakeError(fission.ErrorChecksumFail, "Checksum validation failed")
	}
	return nil
}

func writeSecretOrConfigMap(dataMap map[string][]byte, dirPath string) error {
	for key, val := range dataMap {
		writeFilePath := filepath.Join(dirPath, key)
		err := ioutil.WriteFile(writeFilePath, val, 0600)
		if err != nil {
			return errors.Wrapf(err, "Failed to write file %s", writeFilePath)
		}
	}
	return nil
}

func (fetcher *Fetcher) VersionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(w, fission.BuildInfo().String())
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
	var req fission.FunctionFetchRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		fetcher.logger.Error("error parsing request body", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	code, err := fetcher.Fetch(r.Context(), req)
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
	var req fission.FunctionSpecializeRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		fetcher.logger.Error("error parsing request body", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = fetcher.SpecializePod(r.Context(), req.FetchReq, req.LoadReq)
	if err != nil {
		fetcher.logger.Error("error specialing pod", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// all done
	w.WriteHeader(http.StatusOK)
}

// Fetch takes FetchRequest and makes the fetch call
// It returns the HTTP code and error if any
func (fetcher *Fetcher) Fetch(ctx context.Context, req fission.FunctionFetchRequest) (int, error) {
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

	if req.FetchType == fission.FETCH_URL {
		// fetch the file and save it to the tmp path
		err := downloadUrl(ctx, fetcher.httpClient, req.Url, tmpPath)
		if err != nil {
			e := "failed to download url"
			fetcher.logger.Error(e, zap.Error(err), zap.String("url", req.Url))
			return http.StatusBadRequest, errors.Wrapf(err, "%s: %s", e, req.Url)
		}
	} else {
		// get pkg
		pkg, err := fetcher.fissionClient.Packages(req.Package.Namespace).Get(req.Package.Name)
		if err != nil {
			e := "failed to get package"
			fetcher.logger.Error(e,
				zap.String("package_name", req.Package.Name),
				zap.String("package_namespace", req.Package.Namespace))
			return http.StatusInternalServerError, errors.Wrap(err, e)
		}

		var archive *fission.Archive
		if req.FetchType == fission.FETCH_SOURCE {
			archive = &pkg.Spec.Source
		} else if req.FetchType == fission.FETCH_DEPLOYMENT {
			// sometimes, the user may invoke the function even before the source code is built into a deploy pkg.
			// this results in executor sending a fetch request of type FETCH_DEPLOYMENT and since pkg.Spec.Deployment.Url will be empty,
			// we hit this "Get : unsupported protocol scheme "" error.
			// it may be useful to the user if we can send a more meaningful error in such a scenario.
			if pkg.Status.BuildStatus != fission.BuildStatusSucceeded && pkg.Status.BuildStatus != fission.BuildStatusNone {
				e := fmt.Sprintf("cannot fetch deployment: package build status was not %q", fission.BuildStatusSucceeded)
				fetcher.logger.Error(e,
					zap.String("package_name", pkg.Metadata.Name),
					zap.String("package_namespace", pkg.Metadata.Namespace),
					zap.Any("package_build_status", pkg.Status.BuildStatus))
				return http.StatusInternalServerError, errors.New(fmt.Sprintf("%s: pkg %s.%s has a status of %s", e, pkg.Metadata.Name, pkg.Metadata.Namespace, pkg.Status.BuildStatus))
			}
			archive = &pkg.Spec.Deployment
		}
		// get package data as literal or by url
		if len(archive.Literal) > 0 {
			// write pkg.Literal into tmpPath
			err = ioutil.WriteFile(tmpPath, archive.Literal, 0600)
			if err != nil {
				e := "failed to write file"
				fetcher.logger.Error(e, zap.Error(err), zap.String("location", tmpPath))
				return http.StatusInternalServerError, errors.Wrapf(err, "%s %s", e, tmpPath)
			}
		} else {
			// download and verify
			err := downloadUrl(ctx, fetcher.httpClient, archive.URL, tmpPath)
			if err != nil {
				e := "failed to download url"
				fetcher.logger.Error(e, zap.Error(err), zap.String("url", req.Url))
				return http.StatusBadRequest, errors.Wrapf(err, "%s %s", e, req.Url)
			}

			checksum, err := getChecksum(tmpPath)
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

	if archiver.Zip.Match(tmpPath) && !req.KeepArchive {
		// unarchive tmp file to a tmp unarchive path
		tmpUnarchivePath := filepath.Join(fetcher.sharedVolumePath, uuid.NewV4().String())
		err := fetcher.unarchive(tmpPath, tmpUnarchivePath)
		if err != nil {
			fetcher.logger.Error("error unarchiving",
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
func (fetcher *Fetcher) FetchSecretsAndCfgMaps(secrets []fission.SecretReference, cfgmaps []fission.ConfigMapReference) (int, error) {
	if len(secrets) > 0 {
		for _, secret := range secrets {
			data, err := fetcher.kubeClient.CoreV1().Secrets(secret.Namespace).Get(secret.Name, metav1.GetOptions{})

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
			err = os.MkdirAll(secretDir, os.ModeDir|0644)
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
			data, err := fetcher.kubeClient.CoreV1().ConfigMaps(config.Namespace).Get(config.Name, metav1.GetOptions{})

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
			err = os.MkdirAll(configDir, os.ModeDir|0644)
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

	var req fission.ArchiveUploadRequest
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

	sum, err := getChecksum(dstFilepath)
	if err != nil {
		e := "error calculating checksum of zip file"
		fetcher.logger.Error(e, zap.Error(err), zap.String("file", dstFilepath))
		http.Error(w, fmt.Sprintf("%s: %v", e, err), http.StatusInternalServerError)
		return
	}

	resp := fission.ArchiveUploadResponse{
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

func (fetcher *Fetcher) SpecializePod(ctx context.Context, fetchReq fission.FunctionFetchRequest, loadReq fission.FunctionLoadRequest) error {
	startTime := time.Now()
	defer func() {
		elapsed := time.Since(startTime)
		fetcher.logger.Info("specialize request done", zap.Duration("elapsed_time", elapsed))
	}()

	_, err := fetcher.Fetch(ctx, fetchReq)
	if err != nil {
		return errors.Wrap(err, "error fetching deploy package")
	}

	_, err = fetcher.FetchSecretsAndCfgMaps(fetchReq.Secrets, fetchReq.ConfigMaps)
	if err != nil {
		return errors.Wrap(err, "error fetching secrets/configmaps")
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

	if loadReq.EnvVersion >= 2 {
		contentType = "application/json"
		specializeURL = "http://localhost:8888/v2/specialize"
		reader = bytes.NewReader(loadPayload)
		fetcher.logger.Info("calling environment v2 specialization endpoint")
	} else {
		contentType = "text/plain"
		specializeURL = "http://localhost:8888/specialize"
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

		// Only retry for the specific case of a connection error.
		if urlErr, ok := err.(*url.Error); ok {
			if netErr, ok := urlErr.Err.(*net.OpError); ok {
				if netErr.Op == "dial" {
					if i < maxRetries-1 {
						time.Sleep(500 * time.Duration(2*i) * time.Millisecond)
						fetcher.logger.Error("error connecting to function environment pod for specialization request, retrying", zap.Error(netErr))
						continue
					}
				}
			}
		}

		if err == nil {
			err = fission.MakeErrorFromHTTP(resp)
		}

		return errors.Wrap(err, "error specializing function pod")
	}

	return errors.Wrapf(err, "error specializing function pod after %v times", maxRetries)
}

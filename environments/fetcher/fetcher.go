package fetcher

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mholt/archiver"
	"github.com/satori/go.uuid"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
	storageSvcClient "github.com/fission/fission/storagesvc/client"
)

type (
	FetchRequestType int

	FetchRequest struct {
		FetchType     FetchRequestType             `json:"fetchType"`
		Package       metav1.ObjectMeta            `json:"package"`
		Url           string                       `json:"url"`
		StorageSvcUrl string                       `json:"storagesvcurl"`
		Filename      string                       `json:"filename"`
		Secrets       []fission.SecretReference    `json:"secretList"`
		ConfigMaps    []fission.ConfigMapReference `json:"configMapList"`
		KeepArchive   bool                         `json:"keeparchive"`
	}

	// UploadRequest send from builder manager describes which
	// deployment package should be upload to storage service.
	UploadRequest struct {
		Filename       string `json:"filename"`
		StorageSvcUrl  string `json:"storagesvcurl"`
		ArchivePackage bool   `json:"archivepackage"`
	}

	// UploadResponse defines the download url of an archive and
	// its checksum.
	UploadResponse struct {
		ArchiveDownloadUrl string           `json:"archiveDownloadUrl"`
		Checksum           fission.Checksum `json:"checksum"`
	}

	Fetcher struct {
		sharedVolumePath string
		sharedSecretPath string
		sharedConfigPath string
		fissionClient    *crd.FissionClient
		kubeClient       *kubernetes.Clientset
	}
)

const (
	FETCH_SOURCE = iota
	FETCH_DEPLOYMENT
	FETCH_URL // remove this?
)

func makeVolumeDir(dirPath string) {
	err := os.MkdirAll(dirPath, os.ModeDir|0700)
	if err != nil {
		log.Fatalf("Error creating %v: %v", dirPath, err)
	}
}

func MakeFetcher(sharedVolumePath string, sharedSecretPath string, sharedConfigPath string) (*Fetcher, error) {
	makeVolumeDir(sharedVolumePath)
	makeVolumeDir(sharedSecretPath)
	makeVolumeDir(sharedConfigPath)

	fissionClient, kubeClient, _, err := crd.MakeFissionClient()
	if err != nil {
		return nil, err
	}
	return &Fetcher{
		sharedVolumePath: sharedVolumePath,
		sharedSecretPath: sharedSecretPath,
		sharedConfigPath: sharedConfigPath,
		fissionClient:    fissionClient,
		kubeClient:       kubeClient,
	}, nil
}

func downloadUrl(url string, localPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w, err := os.Create(localPath)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, resp.Body)
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

func verifyChecksum(path string, checksum *fission.Checksum) error {
	if checksum.Type != fission.ChecksumTypeSHA256 {
		return fission.MakeError(fission.ErrorInvalidArgument, "Unsupported checksum type")
	}
	c, err := getChecksum(path)
	if err != nil {
		return err
	}
	if c.Sum != checksum.Sum {
		return fission.MakeError(fission.ErrorChecksumFail, "Checksum validation failed")
	}
	return nil
}

func writeSecretOrConfigMap(dataMap map[string][]byte, dirPath string) error {
	for key, val := range dataMap {
		writeFilePath := filepath.Join(dirPath, key)
		err := ioutil.WriteFile(writeFilePath, val, 0600)
		if err != nil {
			e := fmt.Sprintf("Failed to write file %v: %v", writeFilePath, err)
			log.Printf(e)
			return errors.New(e)
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
		log.Printf("elapsed time in fetch request = %v", elapsed)
	}()

	// parse request
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var req FetchRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("fetcher received fetch request and started downloading: %v", req)
	code, err := fetcher.Fetch(req)
	if err != nil {
		http.Error(w, err.Error(), code)
		return
	}

	log.Printf("Checking secrets/cfgmaps")
	code, err = fetcher.FetchSecretsAndCfgMaps(req.Secrets, req.ConfigMaps)
	if err != nil {
		http.Error(w, err.Error(), code)
		return
	}

	log.Printf("Completed fetch request")
	// all done
	w.WriteHeader(http.StatusOK)
}

// Fetch takes FetchRequest and makes the fetch call
// It returns the HTTP code and error if any
func (fetcher *Fetcher) Fetch(req FetchRequest) (int, error) {
	// check that the requested filename is not an empty string and error out if so
	if len(req.Filename) == 0 {
		e := fmt.Sprintf("Fetch request received for an empty file name, request: %v", req)
		log.Printf(e)
		return http.StatusBadRequest, errors.New(e)
	}

	// verify first if the file already exists.
	if _, err := os.Stat(filepath.Join(fetcher.sharedVolumePath, req.Filename)); err == nil {
		log.Printf("Requested file: %s already exists at %s. Skipping fetch", req.Filename, fetcher.sharedVolumePath)
		return http.StatusOK, nil
	}

	tmpFile := req.Filename + ".tmp"
	tmpPath := filepath.Join(fetcher.sharedVolumePath, tmpFile)

	if req.FetchType == FETCH_URL {
		// fetch the file and save it to the tmp path
		err := downloadUrl(req.Url, tmpPath)
		if err != nil {
			e := fmt.Sprintf("Failed to download url %v: %v", req.Url, err)
			log.Printf(e)
			return http.StatusBadRequest, errors.New(e)
		}
	} else {
		// get pkg
		pkg, err := fetcher.fissionClient.Packages(req.Package.Namespace).Get(req.Package.Name)
		if err != nil {
			e := fmt.Sprintf("Failed to get package: %v", err)
			log.Printf(e)
			return http.StatusInternalServerError, errors.New(e)
		}

		var archive *fission.Archive
		if req.FetchType == FETCH_SOURCE {
			archive = &pkg.Spec.Source
		} else if req.FetchType == FETCH_DEPLOYMENT {
			// sometimes, the user may invoke the function even before the source code is built into a deploy pkg.
			// this results in executor sending a fetch request of type FETCH_DEPLOYMENT and since pkg.Spec.Deployment.Url will be empty,
			// we hit this "Get : unsupported protocol scheme "" error.
			// it may be useful to the user if we can send a more meaningful error in such a scenario.
			if pkg.Status.BuildStatus != fission.BuildStatusSucceeded && pkg.Status.BuildStatus != fission.BuildStatusNone {
				e := fmt.Sprintf("Build status for the function's pkg : %s.%s is : %s, can't fetch deployment", pkg.Metadata.Name, pkg.Metadata.Namespace, pkg.Status.BuildStatus)
				log.Printf(e)
				return http.StatusInternalServerError, errors.New(e)
			}
			archive = &pkg.Spec.Deployment
		}
		// get package data as literal or by url
		if len(archive.Literal) > 0 {
			// write pkg.Literal into tmpPath
			err = ioutil.WriteFile(tmpPath, archive.Literal, 0600)
			if err != nil {
				e := fmt.Sprintf("Failed to write file %v: %v", tmpPath, err)
				log.Printf(e)
				return http.StatusInternalServerError, errors.New(e)
			}
		} else {
			// download and verify
			err = downloadUrl(archive.URL, tmpPath)
			if err != nil {
				e := fmt.Sprintf("Failed to download url %v: %v", req.Url, err)
				log.Printf(e)
				return http.StatusBadRequest, errors.New(e)
			}

			err = verifyChecksum(tmpPath, &archive.Checksum)
			if err != nil {
				e := fmt.Sprintf("Failed to verify checksum: %v", err)
				log.Printf(e)
				return http.StatusBadRequest, errors.New(e)
			}
		}
	}

	if archiver.Zip.Match(tmpPath) && !req.KeepArchive {
		// unarchive tmp file to a tmp unarchive path
		tmpUnarchivePath := filepath.Join(fetcher.sharedVolumePath, uuid.NewV4().String())
		err := fetcher.unarchive(tmpPath, tmpUnarchivePath)
		if err != nil {
			log.Println(err.Error())
			return http.StatusInternalServerError, err
		}

		tmpPath = tmpUnarchivePath
	}

	// move tmp file to requested filename
	err := fetcher.rename(tmpPath, filepath.Join(fetcher.sharedVolumePath, req.Filename))
	if err != nil {
		log.Println(err.Error())
		return http.StatusInternalServerError, err
	}

	log.Printf("Successfully placed at %v", filepath.Join(fetcher.sharedVolumePath, req.Filename))
	return http.StatusOK, nil
}

// FetchSecretsAndCfgMaps fetches secrets and configmaps specified by user
// It returns the HTTP code and error if any
func (fetcher *Fetcher) FetchSecretsAndCfgMaps(secrets []fission.SecretReference, cfgmaps []fission.ConfigMapReference) (int, error) {
	if len(secrets) > 0 {
		for _, secret := range secrets {
			data, err := fetcher.kubeClient.CoreV1().Secrets(secret.Namespace).Get(secret.Name, metav1.GetOptions{})

			if err != nil {
				e := fmt.Sprintf("Failed to get secret from kubeapi: %v", err)
				log.Printf(e)

				httpCode := http.StatusInternalServerError
				if k8serr.IsNotFound(err) {
					httpCode = http.StatusNotFound
				}

				return httpCode, errors.New(e)
			}

			secretPath := filepath.Join(secret.Namespace, secret.Name)
			secretDir := filepath.Join(fetcher.sharedSecretPath, secretPath)
			err = os.MkdirAll(secretDir, os.ModeDir|0644)
			if err != nil {
				e := fmt.Sprintf("Failed to create directory %v: %v", secretDir, err)
				log.Printf(e)
				return http.StatusInternalServerError, errors.New(e)
			}
			err = writeSecretOrConfigMap(data.Data, secretDir)
			if err != nil {
				return http.StatusInternalServerError, err
			}
		}
	}

	if len(cfgmaps) > 0 {
		for _, config := range cfgmaps {
			data, err := fetcher.kubeClient.CoreV1().ConfigMaps(config.Namespace).Get(config.Name, metav1.GetOptions{})

			if err != nil {
				e := fmt.Sprintf("Failed to get configmap from kubeapi: %v", err)
				log.Printf(e)

				httpCode := http.StatusInternalServerError
				if k8serr.IsNotFound(err) {
					httpCode = http.StatusNotFound
				}

				return httpCode, errors.New(e)
			}

			configPath := filepath.Join(config.Namespace, config.Name)
			configDir := filepath.Join(fetcher.sharedConfigPath, configPath)
			err = os.MkdirAll(configDir, os.ModeDir|0644)
			if err != nil {
				e := fmt.Sprintf("Failed to create directory %v: %v", configDir, err)
				log.Printf(e)
				return http.StatusInternalServerError, errors.New(e)
			}
			configMap := make(map[string][]byte)
			for key, val := range data.Data {
				configMap[key] = []byte(val)
			}
			err = writeSecretOrConfigMap(configMap, configDir)
			if err != nil {
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
		log.Printf("elapsed time in upload request = %v", elapsed)
	}()

	// parse request
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var req UploadRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("fetcher received upload request: %v", req)

	zipFilename := req.Filename + ".zip"
	srcFilepath := filepath.Join(fetcher.sharedVolumePath, req.Filename)
	dstFilepath := filepath.Join(fetcher.sharedVolumePath, zipFilename)

	if req.ArchivePackage {
		err = fetcher.archive(srcFilepath, dstFilepath)
		if err != nil {
			e := fmt.Sprintf("Error archiving zip file: %v", err)
			log.Println(e)
			http.Error(w, e, http.StatusInternalServerError)
			return
		}
	} else {
		err = os.Rename(srcFilepath, dstFilepath)
		if err != nil {
			e := fmt.Sprintf("Error renaming the archive: %v", err)
			log.Println(e)
			http.Error(w, e, http.StatusInternalServerError)
			return
		}
	}

	log.Println("Starting upload...")
	ssClient := storageSvcClient.MakeClient(req.StorageSvcUrl)

	fileID, err := ssClient.Upload(dstFilepath, nil)
	if err != nil {
		e := fmt.Sprintf("Error uploading zip file: %v", err)
		log.Println(e)
		http.Error(w, e, http.StatusInternalServerError)
		return
	}

	sum, err := getChecksum(dstFilepath)
	if err != nil {
		e := fmt.Sprintf("Error calculating checksum of zip file: %v", err)
		log.Println(e)
		http.Error(w, e, http.StatusInternalServerError)
		return
	}

	resp := UploadResponse{
		ArchiveDownloadUrl: ssClient.GetUrl(fileID),
		Checksum:           *sum,
	}

	rBody, err := json.Marshal(resp)
	if err != nil {
		e := fmt.Sprintf("Error encoding upload response: %v", err)
		log.Println(e)
		http.Error(w, e, http.StatusInternalServerError)
		return
	}

	log.Println("Completed upload request")
	w.Header().Add("Content-Type", "application/json")
	w.Write(rBody)
	w.WriteHeader(http.StatusOK)
}

func (fetcher *Fetcher) rename(src string, dst string) error {
	err := os.Rename(src, dst)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to move file: %v", err))
	}
	return nil
}

// archive zips the contents of directory at src into a new zip file
// at dst (note that the contents are zipped, not the directory itself).
func (fetcher *Fetcher) archive(src string, dst string) error {
	var files []string
	target, err := os.Stat(src)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to zip file: %v", err))
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
		return errors.New(fmt.Sprintf("Failed to unzip file: %v", err))
	}
	return nil
}

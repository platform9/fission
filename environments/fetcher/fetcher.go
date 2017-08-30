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
	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api"

	"github.com/fission/fission"
	"github.com/fission/fission/tpr"
)

type (
	FetchRequestType int

	FetchRequest struct {
		FetchType     FetchRequestType `json:"fetchType"`
		Function      api.ObjectMeta   `json:"function"`
		Url           string           `json:"url"`
		StorageSvcUrl string           `json:"storagesvcurl"`
		Filename      string           `json:"filename"`
	}

	Fetcher struct {
		sharedVolumePath string
		fissionClient    *tpr.FissionClient
		kubeClient       *kubernetes.Clientset
	}
)

const (
	FETCH_SOURCE = iota
	FETCH_DEPLOYMENT
	FETCH_URL // remove this?
)

func MakeFetcher(sharedVolumePath string) *Fetcher {
	fissionClient, kubeClient, err := tpr.MakeFissionClient()
	if err != nil {
		return nil
	}
	return &Fetcher{
		sharedVolumePath: sharedVolumePath,
		fissionClient:    fissionClient,
		kubeClient:       kubeClient,
	}
}

func downloadUrl(url string, localPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(localPath, body, 0600)
	if err != nil {
		return err
	}

	return nil
}

func verifyChecksum(path string, checksum *fission.Checksum) error {
	if checksum.Type != fission.ChecksumTypeSHA256 {
		return fission.MakeError(fission.ErrorInvalidArgument, "Unsupported checksum type")
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	hasher := sha256.New()
	_, err = io.Copy(hasher, f)
	if err != nil {
		return err
	}

	c := hex.EncodeToString(hasher.Sum(nil))
	if c != checksum.Sum {
		return fission.MakeError(fission.ErrorChecksumFail, "Checksum validation failed")
	}
	return nil
}

func (fetcher *Fetcher) Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "", 405)
		return
	}

	startTime := time.Now()
	defer func() {
		elapsed := time.Now().Sub(startTime)
		log.Printf("elapsed time in fetch request = %v", elapsed)
	}()

	// parse request
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body")
		http.Error(w, err.Error(), 500)
		return
	}
	var req FetchRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		http.Error(w, err.Error(), 400)
		return
	}
	log.Printf("fetcher received request: %v", req)

	tmpFile := req.Filename + ".tmp"
	tmpPath := filepath.Join(fetcher.sharedVolumePath, tmpFile)
	var srcPkgFilename string

	if req.FetchType == FETCH_URL {
		// fetch the file and save it to the tmp path
		err := downloadUrl(req.Url, tmpPath)
		if err != nil {
			e := fmt.Sprintf("Failed to download url %v: %v", req.Url, err)
			log.Printf(e)
			http.Error(w, e, 400)
			return
		}
	} else {
		// get function object
		fn, err := fetcher.fissionClient.Functions(req.Function.Namespace).Get(req.Function.Name)
		if err != nil {
			e := fmt.Sprintf("Failed to get function: %v", err)
			log.Printf(e)
			http.Error(w, e, 500)
			return
		}

		// get pkg
		var pkg *tpr.Package
		if req.FetchType == FETCH_SOURCE {
			pkg, err = fetcher.fissionClient.Packages(fn.Spec.Source.PackageRef.Namespace).Get(fn.Spec.Source.PackageRef.Name)
			srcPkgFilename = fmt.Sprintf("%v-%v", pkg.GetObjectMeta().GetName(), pkg.GetObjectMeta().GetResourceVersion())
		} else if req.FetchType == FETCH_DEPLOYMENT {
			pkg, err = fetcher.fissionClient.Packages(fn.Spec.Deployment.PackageRef.Namespace).Get(fn.Spec.Deployment.PackageRef.Name)
		}
		if err != nil {
			e := fmt.Sprintf("Failed to get package: %v", err)
			log.Printf(e)
			http.Error(w, e, 500)
			return
		}

		// get package data as literal or by url
		if len(pkg.Spec.Literal) > 0 {
			// write pkg.Literal into tmpPath
			err = ioutil.WriteFile(tmpPath, pkg.Spec.Literal, 0600)
			if err != nil {
				e := fmt.Sprintf("Failed to write file %v: %v", tmpPath, err)
				log.Printf(e)
				http.Error(w, e, 500)
				return
			}
		} else {
			// download and verify

			err = downloadUrl(pkg.Spec.URL, tmpPath)
			if err != nil {
				e := fmt.Sprintf("Failed to download url %v: %v", req.Url, err)
				log.Printf(e)
				http.Error(w, e, 400)
				return
			}

			err = verifyChecksum(tmpPath, &pkg.Spec.Checksum)
			if err != nil {
				e := fmt.Sprintf("Failed to verify checksum: %v", err)
				log.Printf(e)
				http.Error(w, e, 400)
				return
			}
		}

	}

	targetFilename := filepath.Join(fetcher.sharedVolumePath, req.Filename)
	// check file type here, if the file is a zip file unarchive it.
	if archiver.Zip.Match(tmpPath) {
		if req.FetchType == FETCH_SOURCE {
			// builder accepts multiple builds at the same time, use `pkg.Name`-`pkg.ResourceVersion`
			// to separate source pacakages for different functions.
			targetFilename = filepath.Join(fetcher.sharedVolumePath, srcPkgFilename)
		}
		// unarchive tmp file to requested filename
		err = fetcher.unarchive(tmpPath, targetFilename)
	} else {
		// move tmp file to requested filename
		err = fetcher.rename(tmpPath, targetFilename)
	}

	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), 500)
		return
	}

	log.Printf("Completed fetch request")
	// all done
	w.WriteHeader(http.StatusOK)
}

func (fetcher *Fetcher) rename(src string, dst string) error {
	err := os.Rename(src, dst)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to move file: %v", err))
	}
	return nil
}

// archive is a function that zips directory into a zip file
func (fetcher *Fetcher) archive(src string, dst string) error {
	return archiver.Zip.Make(dst, []string{src})
}

// unarchive is a function that unzips a zip file to destination
func (fetcher *Fetcher) unarchive(src string, dst string) error {
	err := archiver.Zip.Open(src, dst)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to unzip file: %v", err))
	}
	return nil
}

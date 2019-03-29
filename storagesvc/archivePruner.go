/*
Copyright 2017 The Fission Authors.

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

package storagesvc

import (
	"time"

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission/crd"
)

type ArchivePruner struct {
	logger        *zap.Logger
	crdClient     *crd.FissionClient
	archiveChan   chan (string)
	stowClient    *StowClient
	pruneInterval time.Duration
}

const defaultPruneInterval int = 60 // in minutes

func MakeArchivePruner(logger *zap.Logger, stowClient *StowClient, pruneInterval time.Duration) (*ArchivePruner, error) {
	crdClient, _, _, err := crd.MakeFissionClient()
	if err != nil {
		return nil, err
	}

	return &ArchivePruner{
		logger:        logger.Named("archive_pruner"),
		crdClient:     crdClient,
		archiveChan:   make(chan string),
		stowClient:    stowClient,
		pruneInterval: pruneInterval,
	}, nil
}

// pruneArchives listens to archiveChannel for archive ids that need to be deleted
func (pruner *ArchivePruner) pruneArchives() {
	pruner.logger.Info("listening to archiveChannel to prune archives")
	for {
		select {
		case archiveID := <-pruner.archiveChan:
			pruner.logger.Info("sending delete request for archive",
				zap.String("archive_id", archiveID))
			if err := pruner.stowClient.removeFileByID(archiveID); err != nil {
				// logging the error and continuing with other deletions.
				// hopefully this archive will be deleted in the next iteration.
				pruner.logger.Error("ignoring error while deleting archive",
					zap.Error(err),
					zap.String("archive_id", archiveID))
			}
		}
	}
}

// insertArchive method just writes the archive ID into the channel.
func (pruner *ArchivePruner) insertArchive(archiveID string) {
	pruner.archiveChan <- archiveID
}

// A user may have deleted pkgs with kubectl or fission cli. That only deletes crd.Package objects from kubernetes
// and not the archives that are referenced by them, leaving the archives as orphans.
// getOrphanArchives reaps the orphaned archives.
func (pruner *ArchivePruner) getOrphanArchives() {
	pruner.logger.Info("getting orphan archives")
	archivesRefByPkgs := make([]string, 0)
	var archiveID string

	// get all pkgs from kubernetes
	pkgList, err := pruner.crdClient.Packages(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		pruner.logger.Error("error getting package list from kubernetes", zap.Error(err))
		return
	}

	// extract archives referenced by these pkgs
	for _, pkg := range pkgList.Items {
		if pkg.Spec.Deployment.URL != "" {
			archiveID, err = getQueryParamValue(pkg.Spec.Deployment.URL, "id")
			if err != nil {
				pruner.logger.Error("error extracting value of archiveID from deployment url",
					zap.Error(err),
					zap.String("url", pkg.Spec.Deployment.URL))
				return
			}
			archivesRefByPkgs = append(archivesRefByPkgs, archiveID)
		}
		if pkg.Spec.Source.URL != "" {
			archiveID, err = getQueryParamValue(pkg.Spec.Source.URL, "id")
			if err != nil {
				pruner.logger.Error("error extracting value of archiveID from source url",
					zap.Error(err),
					zap.String("url", pkg.Spec.Source.URL))
				return
			}
			archivesRefByPkgs = append(archivesRefByPkgs, archiveID)
		}
	}

	pruner.logger.Debug("archives referenced by packagese", zap.Strings("archives", archivesRefByPkgs))

	// get all archives on storage
	// out of them, there may be some just created but not referenced by packages yet.
	// need to filter them out.
	archivesInStorage, err := pruner.stowClient.getItemIDsWithFilter(pruner.stowClient.filterItemCreatedAMinuteAgo, time.Now())
	if err != nil {
		pruner.logger.Error("error getting items from storage", zap.Error(err))
		return
	}
	pruner.logger.Debug("archives in storage", zap.Strings("archives", archivesInStorage))

	// difference of the two lists gives us the list of orphan archives. This is just a brute force approach.
	// need to do something more optimal at scale.
	orphanedArchives := getDifferenceOfLists(archivesInStorage, archivesRefByPkgs)
	pruner.logger.Debug("orphan archives", zap.Strings("archives", orphanedArchives))

	// send each orphan archive away for deletion
	for _, archiveID = range orphanedArchives {
		pruner.insertArchive(archiveID)
	}

	return
}

// Start starts a go routine that listens to a channel for archive IDs that need to deleted.
// Also wakes up at regular intervals to make a list of archive IDs that need to be reaped
// and sends them over to the channel for deletion
func (pruner *ArchivePruner) Start() {
	ticker := time.NewTicker(pruner.pruneInterval * time.Minute)
	go pruner.pruneArchives()
	for {
		select {
		case <-ticker.C:
			// This method fetches unused archive IDs and sends them to archiveChannel for deletion
			// silencing the errors, hoping they go away in next iteration.
			pruner.getOrphanArchives()
		}
	}
}

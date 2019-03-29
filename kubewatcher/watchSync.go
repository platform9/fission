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

package kubewatcher

import (
	"time"

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission/crd"
)

type (
	WatchSync struct {
		logger      *zap.Logger
		client      *crd.FissionClient
		kubeWatcher *KubeWatcher
	}
)

func MakeWatchSync(logger *zap.Logger, client *crd.FissionClient, kubeWatcher *KubeWatcher) *WatchSync {
	ws := &WatchSync{
		logger:      logger.Named("watch_sync"),
		client:      client,
		kubeWatcher: kubeWatcher,
	}
	go ws.syncSvc()
	return ws
}

func (ws *WatchSync) syncSvc() {
	// TODO watch instead of polling
	for {
		watches, err := ws.client.KubernetesWatchTriggers(metav1.NamespaceAll).List(metav1.ListOptions{})
		if err != nil {
			ws.logger.Fatal("failed to get Kubernetes watch trigger list", zap.Error(err))
		}

		ws.kubeWatcher.Sync(watches.Items)
		time.Sleep(3 * time.Second)
	}
}

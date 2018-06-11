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

package router

import (
	"log"
	"math/rand"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func setupCanaryLoadBalancer() {
	// just seeding the random number
	rand.Seed(time.Now().UnixNano())
}

func findCeil(randomNumber int, wtDistrList []FunctionWeightDistribution) string {
	low := 0
	high := len(wtDistrList) - 1

	for {
		if low >= high {
			break
		}

		mid := low + high/2
		log.Printf("mid : %d", mid)
		if randomNumber >= wtDistrList[mid].sumPrefix {
			low = mid + 1
			log.Printf("low %d", low)
			log.Printf("randomNumber %d > wtDistrList[mid].sumPrefix %d", randomNumber, wtDistrList[mid].sumPrefix)

		} else {
			log.Printf("randomNumber %d < wtDistrList[mid].sumPrefix %d", randomNumber, wtDistrList[mid].sumPrefix)
			high = mid
			log.Printf("high %d", high)
		}
	}

	if wtDistrList[low].sumPrefix >= randomNumber {
		log.Printf("Final low index : %d, returning fnName : %s", low, wtDistrList[low].name)
		return wtDistrList[low].name
	} else {
		return ""
	}
}

func getCanaryBackend(fnMetadatamap map[string]*metav1.ObjectMeta, fnWtDistributionList []FunctionWeightDistribution) *metav1.ObjectMeta {

	log.Printf("Dumping fnMetadataMap : %+v, fnWtDistrList : %v", fnMetadatamap, fnWtDistributionList)

	randomNumber := rand.Intn(fnWtDistributionList[len(fnWtDistributionList)-1].sumPrefix + 1)

	log.Printf("randomNumber : %d", randomNumber)
	fnName := findCeil(randomNumber, fnWtDistributionList)

	log.Printf("chosen function : %s", fnName)

	return fnMetadatamap[fnName]
}

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
	"net/url"

	"github.com/pkg/errors"
)

func getQueryParamValue(urlString string, queryParam string) (string, error) {
	url, err := url.Parse(urlString)
	if err != nil {
		return "", errors.Wrapf(err, "error parsing URL string %q into URL", urlString)
	}
	return url.Query().Get(queryParam), nil
}

func getDifferenceOfLists(firstList []string, secondList []string) []string {
	tempMap := make(map[string]int)
	differenceList := make([]string, 0)

	for _, item := range firstList {
		tempMap[item] = 1
	}

	for _, item := range secondList {
		_, ok := tempMap[item]
		if ok {
			delete(tempMap, item)
		}
	}

	for k := range tempMap {
		differenceList = append(differenceList, k)
	}

	return differenceList
}

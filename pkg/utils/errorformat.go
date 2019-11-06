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

package utils

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"
)

func MultiErrorWithFormat() *multierror.Error {
	return &multierror.Error{
		ErrorFormat: DefaultErrorFormat,
	}
}

func DefaultErrorFormat(es []error) string {
	points := make([]string, len(es))
	for i, err := range es {
		points[i] = fmt.Sprintf("* %s", err)
	}
	return fmt.Sprintf(
		"%d errors occurred:\n\t%s\n",
		len(es), strings.Join(points, "\n\t"))
}

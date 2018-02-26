# Copyright 2017 The Fission Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
.PHONY: test
.DEFAULT_GOAL := build

IMAGE ?= fission/fission-bundle
VERSION ?= latest
ARCH ?= amd64
OS ?= linux

test:
	go test -v $(shell go list ./... | grep -v /examples/ | grep -v /environments/)

build: build-bundle build-client

build-client:
	go build -o fission/fission ./fission/*.go

build-bundle:
	CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) go build -o fission-bundle/fission-bundle ./fission-bundle/*.go

build-image:
	docker build --rm --tag "$(IMAGE):$(VERSION)" fission-bundle

install:
	go install ./fission

image: build-bundle build-image

image-push: image
	docker push "$(IMAGE):$(VERSION)"

clean:
	@rm -f fission-bundle/fission-bundle
	@rm -f fission/fission

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

package fission

import (
	"encoding/json"
	"time"
)

var (
	GitCommit string // $(git rev-parse HEAD) (1b4716ab84903b2e477135a3dc5afdb07f685cb7)
	BuildDate string // $(date -u +'%Y-%m-%dT%H:%M:%SZ') (2018-03-08T18:54:38Z)
	Version   string // fission release version (0.6.0)
)

type (
	BuildMeta struct {
		GitCommit string `json:"GitCommit,omitempty"`
		BuildDate string `json:"BuildDate,omitempty"`
		Version   string `json:"Version,omitempty"`
	}

	Time struct {
		Timezone    string    `json:"Timezone,omitempty"`
		CurrentTime time.Time `json:"CurrentTime,omitempty"`
	}

	ServerInfo struct {
		Build      BuildMeta `json:"Build,omitempty"`
		ServerTime Time      `json:"ServerTime,omitempty"`
	}
)

func BuildInfo() BuildMeta {
	return BuildMeta{
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		Version:   Version,
	}
}

func (info BuildMeta) String() string {
	v, _ := json.Marshal(info)
	return string(v)
}

func TimeInfo() Time {
	t := time.Now()
	zone, _ := t.Local().Zone()
	return Time{
		Timezone:    zone,
		CurrentTime: t,
	}
}

func ApiInfo() ServerInfo {
	return ServerInfo{
		Build:      BuildInfo(),
		ServerTime: TimeInfo(),
	}
}

func (info ServerInfo) String() string {
	v, _ := json.Marshal(info)
	return string(v)
}

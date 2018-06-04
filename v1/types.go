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

package v1

//
// These are types from the v1 API, and are only preserved for
// compatibility.  They should never be changed.
//

type (
	// Metadata is used as the general identifier for all kinds of
	// resources managed by the controller.
	Metadata struct {
		Name string `json:"name"`
		Uid  string `json:"uid,omitempty"`
	}

	// Function is a unit of executable code.  Though it's called
	// a function, the code may have more than one function; it's
	// usually some sort of module or package.
	Function struct {
		Metadata    `json:"metadata"`
		Environment Metadata `json:"environment"`
		Code        string   `json:"code"`
	}

	// Environment identifies the language and OS specific
	// resources that a function depends on.  For now this
	// includes only the function run container image.  Later,
	// this will also include build containers, as well as support
	// tools like debuggers, profilers, etc.
	Environment struct {
		Metadata             `json:"metadata"`
		RunContainerImageUrl string `json:"runContainerImageUrl"`
	}

	// HTTPTrigger maps URL patterns to functions.  Function.UID
	// is optional; if absent, the latest version of the function
	// will automatically be selected.
	HTTPTrigger struct {
		Metadata   `json:"metadata"`
		UrlPattern string   `json:"urlpattern"`
		Method     string   `json:"method"`
		Function   Metadata `json:"function"`
	}

	MessageQueueTrigger struct {
		Metadata         `json:"metadata"`
		Function         Metadata `json:"function"`
		MessageQueueType string   `json:"messageQueueType"`
		Topic            string   `json:"topic"`
		ResponseTopic    string   `json:"respTopic,omitempty"`
	}

	// Watch is a specification of Kubernetes watch along with a URL to post events to.
	Watch struct {
		Metadata `json:"metadata"`

		Namespace     string `json:"namespace"`
		ObjType       string `json:"objtype"`
		LabelSelector string `json:"labelselector"`
		FieldSelector string `json:"fieldselector"`

		Function Metadata `json:"function"`

		Target string `json:"target"` // Watch publish target (URL, NATS stream, etc)
	}

	// TimeTrigger invokes the specific function at a time or
	// times specified by a cron string.
	TimeTrigger struct {
		Metadata `json:"metadata"`

		Cron string `json:"cron"`

		Function Metadata `json:"function"`
	}

	// Errors returned by the Fission API.
	Error struct {
		Code    errorCode `json:"code"`
		Message string    `json:"message"`
	}

	errorCode int
)

const (
	ErrorInternal = iota

	ErrorNotAuthorized
	ErrorNotFound
	ErrorNameExists
	ErrorInvalidArgument
	ErrorNoSpace
	ErrorNotImplmented
)

// must match order and len of the above const
var errorDescriptions = []string{
	"Internal error",
	"Not authorized",
	"Resource not found",
	"Resource exists",
	"Invalid argument",
	"No space",
	"Not implemented",
}

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

package publisher

type (
	// Publisher publish an request to a "target".  Target depends on the
	// publisher: it's a URL in the case of a webhook publisher, or a queue
	// name in a queue-based publisher such as NATS.
	Publisher interface {
		Publish(body string, headers map[string]string, target string)
	}
)

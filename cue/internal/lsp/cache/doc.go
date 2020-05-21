// Copyright 2019 Tobias Guggenmos
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package cache is the component of the CUE language server that is
// responsible for the caching the content and parse results of documents opened in the language client.
//
// TODO: At the moment the separation between Document and compile result is not really clear.
// For example, a compiler is created for every document, even though that should not be needed.
// Ideally, we would have a more higher level abstraction.
//
// One idea would be, to have a DocumentCache and PackageCache.
// DocumentCache is responsible for caching documents and what packages they belong to, while PackageCache is responsible for building packages and caching the build result.
// If a file does not belong to a package (yet), it would have a separate package.
package cache

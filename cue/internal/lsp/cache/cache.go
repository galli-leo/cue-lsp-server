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

package cache

import (
	"context"
	"net/url"
	"path/filepath"
	"strings"
	"sync"

	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/internal/lsp/asg"
	"cuelang.org/go/cue/internal/lsp/internal/vendored/go-tools/jsonrpc2"
	"cuelang.org/go/cue/internal/lsp/internal/vendored/go-tools/lsp/protocol"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/cue/token"
)

// We need this so we can reserve a certain position range in the FileSet
// for each Document.
const maxDocumentSize = 1000000000

// Convert any DocumentURI into a normal path.
// Returns an empty string on failure.
func PathFromURI(uri protocol.DocumentURI) string {
	parsed, err := url.Parse(string(uri))
	if err == nil {
		if strings.HasPrefix(parsed.Scheme, "file") {
			return parsed.Path
		}
	}

	return ""
}

// DocumentCache caches the documents and compile Results associated with one server-client connection or one REST API instance.
//
// Before a cache instance can be used, Init must be called.
type DocumentCache struct {
	rootURI   protocol.DocumentURI
	documents map[protocol.DocumentURI]*document
	mu        sync.RWMutex
	// Channel to send log messages to.
	Logging chan protocol.LogMessageParams
}

// Returns the root path as an absolute path
func (c *DocumentCache) root() string {
	return PathFromURI(c.rootURI)
}

// Returns the absolute path of the given uri.
// If the uri is relative, it will be resolved relative to the rootURI of the cache.
// Returns empty string on error or empty root and relative path.
func (c *DocumentCache) absPath(uri protocol.DocumentURI) string {
	path := PathFromURI(uri)
	if path != "" {
		if filepath.IsAbs(path) {
			return path
		}

		root := c.root()
		if root != "" {
			return filepath.Join(root, path)
		}
	}

	return ""
}

// Init initializes a Document cache.
func (c *DocumentCache) Init() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.documents = make(map[protocol.DocumentURI]*document)
}

// Loads any cue packages / modules found inside the root URI passed in.
// This URI will also be used as the root for all further requests!
func (c *DocumentCache) LoadRootFolder(uri protocol.DocumentURI) {
	c.rootURI = uri
}

// Reloads all packages / modules inside the root folder
func (c *DocumentCache) CreateCompiler() (*asg.Compiler, error) {
	overlay, err := c.overlay()
	if err != nil {
		return nil, err
	}
	config := load.Config{
		Dir:     c.root(),
		Overlay: overlay,
	}

	return asg.NewCompiler(&config), nil
}

func (c *DocumentCache) overlay() (map[string]load.Source, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ret := make(map[string]load.Source)
	for uri, doc := range c.documents {
		path := c.absPath(uri)
		if path != "" {
			contents := doc.content
			ret[path] = load.FromString(contents)
		}
	}

	return ret, nil
}

// AddDocument adds a document to the cache.
//
// This triggers async parsing of the document.
func (c *DocumentCache) AddDocument(serverLifetime context.Context, doc *protocol.TextDocumentItem) (*DocumentHandle, error) {
	if _, ok := c.documents[doc.URI]; ok {
		return nil, errors.New("document already exists")
	}

	path := c.absPath(doc.URI)

	file := token.NewFile(path, -1, maxDocumentSize)

	if r := recover(); r != nil {
		if err, ok := r.(error); !ok {
			return nil, jsonrpc2.NewErrorf(jsonrpc2.CodeInternalError, "cache/addDocument: %v", err)
		}
	}

	file.SetLinesForContent([]byte(doc.Text))

	// r := runtime.New()
	// comp := compile.Compiler{Index: r}

	d := &document{
		posData:    file,
		uri:        doc.URI,
		path:       path,
		languageID: doc.LanguageID,
		// runtime:    r,
		// compiler:   &comp,
		cache: c,
	}

	d.compilers.initialize()

	err := (&DocumentHandle{d, context.Background()}).SetContent(serverLifetime, doc.Text, doc.Version, true)

	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.documents[doc.URI] = d

	return &DocumentHandle{d, d.versionCtx}, nil
}

// GetDocument retrieve a document from the cache.
func (c *DocumentCache) GetDocument(uri protocol.DocumentURI) (*DocumentHandle, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	ret, ok := c.documents[uri]

	if !ok {
		return nil, jsonrpc2.NewErrorf(jsonrpc2.CodeInternalError, "cache/getDocument: Document not found: %v", uri)
	}

	ret.mu.RLock()
	defer ret.mu.RUnlock()

	return &DocumentHandle{ret, ret.versionCtx}, nil
}

// RemoveDocument removes a document from the cache.
func (c *DocumentCache) RemoveDocument(uri protocol.DocumentURI) error {
	d, err := c.GetDocument(uri)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	d.doc.obsoleteVersion()

	delete(c.documents, uri)

	return nil
}

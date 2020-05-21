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
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"sync"

	"cuelang.org/go/cue/token"

	"cuelang.org/go/cue/internal/lsp/asg"
	"cuelang.org/go/cue/internal/lsp/internal/vendored/go-tools/jsonrpc2"
	"cuelang.org/go/cue/internal/lsp/internal/vendored/go-tools/lsp/protocol"
	"cuelang.org/go/cue/internal/lsp/internal/vendored/go-tools/span"
)

// document caches content, metadata and compile results of a document.
// TODO: See doc.go on separation of document and build result.
type document struct {
	posData *token.File

	uri        protocol.DocumentURI
	path       string
	languageID string
	version    float64
	content    string

	mu sync.RWMutex

	cache *DocumentCache

	// A context.Context that expires when the document changes.
	versionCtx context.Context
	// The corresponding cancel function.
	obsoleteVersion context.CancelFunc

	// The package we built.
	pkg *asg.Package

	// The diagnostics created when parsing the document.
	diagnostics map[protocol.DocumentURI][]protocol.Diagnostic

	// Wait for this before accessing the compile results or diagnostics.
	compilers waitGroup
}

func (d *document) Filename() string {
	uris := string(d.uri)
	if !strings.HasPrefix(uris, "file://") {
		uris = "file://" + uris
	}
	u, _ := url.Parse(uris)
	return filepath.Base(u.Path)
}

// DocumentHandle bundles a Document together with a context.Context that expires
// when the document changes.
//
// All exported document access methods must be threadsafe and fail if the associated
// context has expired, unless otherwise documented.
type DocumentHandle struct {
	doc *document
	ctx context.Context
}

// ApplyIncrementalChanges applies given changes to a given document content.
// The context in the DocumentHandle is ignored.
func (d *DocumentHandle) ApplyIncrementalChanges(changes []protocol.TextDocumentContentChangeEvent, version float64) (string, error) {
	d.doc.mu.RLock()
	defer d.doc.mu.RUnlock()

	if version <= d.doc.version {
		return "", jsonrpc2.NewErrorf(jsonrpc2.CodeInvalidParams, "Update to file didn't increase version number")
	}

	content := []byte(d.doc.content)
	uri := d.doc.uri

	for _, change := range changes {
		// Update column mapper along with the content.
		converter := span.NewContentConverter(string(uri), content)
		m := &protocol.ColumnMapper{
			URI:       span.URI(d.doc.uri),
			Converter: converter,
			Content:   content,
		}

		spn, err := m.RangeSpan(*change.Range)

		if err != nil {
			return "", err
		}

		if !spn.HasOffset() {
			return "", jsonrpc2.NewErrorf(jsonrpc2.CodeInternalError, "invalid range for content change")
		}

		start, end := spn.Start().Offset(), spn.End().Offset()
		if end < start {
			return "", jsonrpc2.NewErrorf(jsonrpc2.CodeInternalError, "invalid range for content change")
		}

		var buf bytes.Buffer

		buf.Write(content[:start])
		buf.WriteString(change.Text)
		buf.Write(content[end:])

		content = buf.Bytes()
	}

	return string(content), nil
}

// SetContent sets the content of a document.
//
// This triggers async parsing of the document.
func (d *DocumentHandle) SetContent(serverLifetime context.Context, content string, version float64, new bool) error {
	d.doc.mu.Lock()
	defer d.doc.mu.Unlock()

	if !new && version <= d.doc.version {
		return jsonrpc2.NewErrorf(jsonrpc2.CodeInvalidParams, "Update to file didn't increase version number")
	}

	if len(content) > maxDocumentSize {
		return jsonrpc2.NewErrorf(jsonrpc2.CodeInternalError, "cache/SetContent: Provided.document to large.")
	}

	if !new {
		d.doc.obsoleteVersion()
	}

	d.doc.versionCtx, d.doc.obsoleteVersion = context.WithCancel(serverLifetime)

	d.doc.content = content
	d.doc.version = version

	// An additional newline is appended, to make sure the last line is indexed
	d.doc.posData.SetLinesForContent(append([]byte(content), '\n'))

	d.doc.diagnostics = make(map[protocol.DocumentURI][]protocol.Diagnostic)
	d.doc.pkg = nil

	d.doc.compilers.Add(1)

	// We need to create a new document handler here since the old one
	// still carries the deprecated version context
	go (&DocumentHandle{d.doc, d.doc.versionCtx}).compile() //nolint:errcheck

	return nil
}

// GetContent returns the content of a document.
func (d *DocumentHandle) GetContent() (string, error) {
	d.doc.mu.RLock()
	defer d.doc.mu.RUnlock()

	select {
	case <-d.ctx.Done():
		return "", d.ctx.Err()
	default:
		return d.doc.content, nil
	}
}

// GetSubstring returns a substring of the content of a document.
//
// The parameters are the start and end of the substring, encoded
// as token.Pos
func (d *DocumentHandle) GetSubstring(pos token.Pos, endPos token.Pos) (string, error) {
	d.doc.mu.RLock()
	defer d.doc.mu.RUnlock()

	select {
	case <-d.ctx.Done():
		return "", d.ctx.Err()
	default:
		content := d.doc.content
		start := pos.Offset()
		end := endPos.Offset()

		if start < 0 || start > end || end > len(content) {
			return "", errors.New("invalid range")
		}

		return content[start:end], nil
	}
}

// GetQueries returns the compiled queries of a document.
//
// It blocks until all compile tasks are finished.
func (d *DocumentHandle) GetCompiled() (*asg.Package, error) {
	d.doc.compilers.Wait()

	d.doc.mu.RLock()

	defer d.doc.mu.RUnlock()

	select {
	case <-d.ctx.Done():
		return nil, d.ctx.Err()
	default:
		return d.doc.pkg, nil
	}
}

// GetVersion returns the version of a document.
func (d *DocumentHandle) GetVersion() (float64, error) {
	d.doc.mu.RLock()
	defer d.doc.mu.RUnlock()

	select {
	case <-d.ctx.Done():
		return 0, d.ctx.Err()
	default:
		return d.doc.version, nil
	}
}

// GetLanguageID returns the language ID of a document.
//
// Since the languageID never changes, it does not block or return errors.
func (d *DocumentHandle) GetLanguageID() string {
	return d.doc.languageID
}

// GetDiagnostics returns the diagnostics created during the compilation of a document.
//
// It blocks until all compile tasks are finished.
func (d *DocumentHandle) GetDiagnostics() (map[protocol.DocumentURI][]protocol.Diagnostic, error) {
	d.doc.compilers.Wait()

	d.doc.mu.RLock()
	defer d.doc.mu.RUnlock()

	select {
	case <-d.ctx.Done():
		return nil, d.ctx.Err()
	default:
		return d.doc.diagnostics, nil
	}
}

func (d *DocumentHandle) Log(level protocol.MessageType, msg string, args ...interface{}) {
	d.doc.cache.Logging <- protocol.LogMessageParams{
		Type:    level,
		Message: fmt.Sprintf(msg, args...),
	}
}

// func (d *DocumentHandle) GetLabel(label adt.Feature) string {
// 	return label.ToString(d.doc.runtime)
// }

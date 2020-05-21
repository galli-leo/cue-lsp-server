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
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/internal/lsp/internal/vendored/go-tools/lsp/protocol"
	"cuelang.org/go/cue/token"
)

// promQLErrToProtocolDiagnostic converts a promql.ParseErr to a protocol.Diagnostic
//
// The position of the query must be passed as the first argument.
func (d *DocumentHandle) cueErrToProtocolDiagnostic(cueErr errors.Error, startPos token.Pos, endPos token.Pos) (*protocol.Diagnostic, error) {
	start, err := d.PosToProtocolPosition(startPos)
	if err != nil {
		return nil, err
	}

	end, err := d.PosToProtocolPosition(endPos)
	if err != nil {
		return nil, err
	}

	message := &protocol.Diagnostic{
		Range: protocol.Range{
			Start: start,
			End:   end,
		},
		Severity: 1, // Error
		Source:   "cue-lsp",
		Message:  cueErr.Error(),
	}

	return message, nil
}

// addDiagnostic adds a protocol.Diagnostic to the diagnostics of a Document.
//
//If the context is expired the diagnostic is discarded.
func (d *DocumentHandle) addDiagnostic(diagnostic *protocol.Diagnostic, uri protocol.DocumentURI) error {
	d.doc.mu.Lock()
	defer d.doc.mu.Unlock()

	select {
	case <-d.ctx.Done():
		return d.ctx.Err()
	default:
		// Duplicate prevention
		// for _, diag := range d.doc.diagnostics {
		// 	if diag.Message == diagnostic.Message && diag.Range == diagnostic.Range {
		// 		return nil
		// 	}
		// }
		if existing, ok := d.doc.diagnostics[uri]; ok {
			d.doc.diagnostics[uri] = append(existing, *diagnostic)
		} else {
			d.doc.diagnostics[uri] = []protocol.Diagnostic{*diagnostic}
		}
		return nil
	}
}

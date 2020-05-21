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
	"testing"

	"cuelang.org/go/cue/token"
)

// Call the (* Document) Functions with an expired context. Expected behaviour is that all
// of these calls return an error
func TestDocumentContext(t *testing.T) { //nolint: funlen
	doc := &document{}

	doc.posData = token.NewFile("", -1, 0)

	doc.compilers.initialize()

	expired, cancel := context.WithCancel(context.Background())

	cancel()

	d := &DocumentHandle{doc, expired}

	// From compile.go

	// Necessary since compile() will call d.compilers.Done()
	d.doc.compilers.Add(1)

	d.doc.languageID = "cue"

	if err := d.compile(); err == nil {
		panic("Expected compile to fail with expired context (languageID: cue)")
	}

	// Necessary since compileQuery() will call d.compilers.Done()
	d.doc.compilers.Add(1)

	if err := d.compileCue(true, token.NoPos, token.NoPos, ""); err == nil {
		panic("Expected compileQuery to fail with expired context (fullFile: true)")
	}

	// Necessary since compileQuery() will call d.compilers.Done()
	d.doc.compilers.Add(1)

	if err := d.compileCue(false, token.NoPos, token.NoPos, ""); err == nil {
		panic("Expected compileQuery to fail with expired context (fullFile: false)")
	}

	// if err := d.addCompileResult(token.NoPos, nil, adt.Arc{}, nil, nil, ""); err == nil {
	// 	panic("Expected AddCompileResult to fail with expired context")
	// }

	// From diagnostics.go

	if _, err := d.cueErrToProtocolDiagnostic(nil, token.NoPos, token.NoPos); err == nil {
		panic("Expected promQLErrToProtocolDiagnostic to fail with expired context")
	}

	// if err := d.addDiagnostic(nil); err == nil {
	// 	panic("Expected AddDiagnostic to fail with expired context")
	// }

	// From document.go

	if _, err := d.GetContent(); err == nil {
		panic("Expected GetContent to fail with expired context")
	}

	if _, err := d.GetSubstring(token.NoPos, token.NoPos); err == nil {
		panic("Expected GetSubstring to fail with expired context")
	}

	// if _, _, _, err := d.GetCompiled(); err == nil {
	// 	panic("Expected GetQueries to fail with expired context")
	// }

	if _, err := d.GetVersion(); err == nil {
		panic("Expected GetVersion to fail with expired context")
	}

	if _, err := d.GetDiagnostics(); err == nil {
		panic("Expected GetDiagnostics to fail with expired context")
	}

	// From position.go

	if _, err := d.tokenPositionToProtocolPosition(token.Position{}); err == nil {
		panic("Expected PositionToProtocolPosition to fail with expired context")
	}

	if _, err := d.PosToProtocolPosition(token.NoPos); err == nil {
		panic("Expected PosToProtocolPosition to fail with expired context")
	}

	if _, err := d.tokenPosToTokenPosition(token.NoPos); err == nil {
		panic("Expected TokenPosToTokenPosition to fail with expired context")
	}

	if _, err := d.GetVersion(); err == nil {
		panic("Expected GetVersion to fail with expired context")
	}

	if _, err := d.GetDiagnostics(); err == nil {
		panic("Expected GetContent to fail with expired context")
	}
}

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
	"cuelang.org/go/cue/internal/lsp/asg"
	"cuelang.org/go/cue/internal/lsp/internal/vendored/go-tools/lsp/protocol"
	"cuelang.org/go/cue/token"
)

// Location bundles all the context that the cache can provide for a given protocol.Location.
type Location struct {
	Doc     *DocumentHandle
	Pos     token.Pos
	Package *asg.Package
	Node    asg.Node
	// Root       *ast.File
	// Arc        adt.Arc
	// RootCursor *ADTCursor
	// Cursor     *ADTCursor
}

// Find returns all the information about a given position the cache can provide.
//
// It blocks until the document is fully parsed.
func (c *DocumentCache) Find(where *protocol.TextDocumentPositionParams) (there *Location, err error) {
	there = &Location{}

	if there.Doc, err = c.GetDocument(where.TextDocument.URI); err != nil {
		return
	}

	if there.Pos, err = there.Doc.protocolPositionToTokenPos(where.Position); err != nil {
		return
	}

	//fmt.Println("Resolving Position: " + there.Pos.String())
	//fmt.Println("Contents: " + there.Doc.doc.content)

	// if there.Root, there.Arc, there.RootCursor, err = there.Doc.GetCompiled(); err != nil {
	// 	return
	// }

	if there.Package, err = there.Doc.GetCompiled(); err != nil {
		return
	}

	there.Node = there.Package.Find(there.Pos)

	// there.Cursor = there.RootCursor.SmallestSurroundingNode(there.Pos)
	//fmt.Printf("[%T, %s]: %v\n", there.Cursor.Node, there.Cursor.Pos().String(), there.Cursor.Node)

	return
}

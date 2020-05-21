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
	"fmt"
	"path/filepath"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/build"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/internal/lsp/asg"
	"cuelang.org/go/cue/internal/lsp/internal/vendored/go-tools/lsp/protocol"
	"cuelang.org/go/cue/parser"
	"cuelang.org/go/cue/token"
)

// compile asynchronously parses the provided document.
func (d *DocumentHandle) compile() error {
	defer d.doc.compilers.Done()

	switch d.GetLanguageID() {
	case "cue":
		d.doc.compilers.Add(1)
		err := d.compileCue(true, token.NoPos, token.NoPos, "")
		if err != nil {
			d.Log(protocol.Error, "had error while trying to compile doc: %v", err)
		}
		return err
	default:
	}

	return nil
}

// compileCue compiles the Cue document at the position given by the last two arguments.
//
// If fullFile is set, the last two arguments are ignored and the full file is assumed
// to be one query.
//
// d.compilers.Add(1) must be called before calling this.
func (d *DocumentHandle) compileCue(fullFile bool, pos token.Pos, endPos token.Pos, record string) error {
	defer d.doc.compilers.Done()

	var content string

	var expired error

	// For now always compile full file!
	if fullFile || true {
		content, expired = d.GetContent()
		pos = d.doc.posData.Pos(0, token.NoRelPos)
	} else {
		content, expired = d.GetSubstring(pos, endPos)
	}

	if expired != nil {
		return expired
	}

	compiler, err := d.doc.cache.CreateCompiler()
	if err != nil {
		return err
	}

	relative, err := filepath.Rel(d.doc.cache.root(), filepath.Dir(d.doc.path))

	pkg, err := compiler.CompileFile("./" + relative)

	d.Log(protocol.Info, "Package: %v", pkg)

	parseErr := errors.Errors(err)

	// arc := d.doc.compiler.CompileFiles(files...)

	// compileErr := errors.Errors(d.doc.compiler.Errs)

	// err = d.addCompileResult(pos, files[0], arc, parseErr, compileErr, content)
	err = d.addCompileResult(pos, pkg, parseErr, content)

	if err != nil {
		return err
	}

	// Create LUT for errors
	// posToNode := make(map[token.Pos]*ADTCursor)
	// for iter, child := d.doc.rootCursor.DFS(); child != nil; child = iter.Next() {
	// 	if prev, ok := posToNode[child.Pos()]; ok {
	// 		if child.Length() > prev.Length() {
	// 			posToNode[child.Pos()] = child
	// 		}
	// 	} else {
	// 		posToNode[child.Pos()] = child
	// 	}
	// }

	for _, e := range parseErr {
		// try to find token in AST responsible for error
		start, end := e.Position(), e.Position()
		// if c, ok := posToNode[start]; ok {
		// 	end = c.End()
		// }
		if rng, ok := e.(*asg.RangeError); ok {
			start, end = rng.Range()
		} else {
			n := pkg.Find(start)
			if n != nil {
				end = n.End()
			}
		}

		diagnostic, err := d.cueErrToProtocolDiagnostic(e, start, end) //nolint:scopelint
		if err != nil {
			return err
		}

		path := start.Filename()
		uri := protocol.DocumentURI(fmt.Sprintf("file://%s", path))

		err = d.addDiagnostic(diagnostic, uri)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *DocumentHandle) parseInstance(inst *build.Instance) ([]*ast.File, error) {
	ret := []*ast.File{}
	var retErr errors.Error = nil
	for _, file := range inst.BuildFiles {
		ast, err := parser.ParseFile(file.Filename, file.Source, parser.AllErrors, parser.AllowPartial, parser.ParseComments)
		if err != nil {
			retErr = errors.Append(retErr, errors.Promote(err, ""))
		}
		ret = append(ret, ast)
	}
	for _, child := range inst.Imports {
		childRet, err := d.parseInstance(child)
		if err != nil {
			retErr = errors.Append(retErr, errors.Promote(err, ""))
		}
		ret = append(ret, childRet...)
	}
	return ret, retErr
}

// addCompileResult adds a compiled query compilation results of a Document.
//
// If the DocumentHandle is expired, the result is discarded.
func (d *DocumentHandle) addCompileResult(pos token.Pos, pkg *asg.Package, parseErr []errors.Error, content string) error {
	d.doc.mu.Lock()
	defer d.doc.mu.Unlock()

	select {
	case <-d.ctx.Done():
		return d.ctx.Err()
	default:
		d.doc.pkg = pkg
		return nil
	}
}

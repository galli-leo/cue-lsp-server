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

package lsp

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"cuelang.org/go/cue/internal/adt"
	"cuelang.org/go/cue/internal/lsp/asg"
	"cuelang.org/go/cue/internal/lsp/cache"
	"cuelang.org/go/cue/internal/lsp/internal/vendored/go-tools/lsp/protocol"
)

// Completion is required by the protocol.Server interface
// nolint: wsl
func (s *server) Completion(ctx context.Context, params *protocol.CompletionParams) (ret *protocol.CompletionList, err error) {
	posParams := params.TextDocumentPositionParams
	//posParams.Position.Character-- // We need one char less to get correct token
	location, err := s.cache.Find(&posParams)
	if err != nil {
		return nil, nil
	}

	ret = &protocol.CompletionList{}
	ret.IsIncomplete = true

	completions := &ret.Items

	switch n := location.Node.(type) {
	case *asg.Reference:
		start := n.Referenced
		if start == nil {
			start = location.Package
			// If we don't have any existing contexts, recommend builtins
			s.completeBuiltins(ctx, completions)
		}
		if err = s.completeReference(ctx, completions, start); err != nil {
			return
		}
	case *asg.Decl:
		s.completeDecls(ctx, completions, location)
	// case *adt.SelectorExpr:
	// 	scopes, _ := location.RootCursor.ResolveSelector(n, location.Doc)
	// 	for _, scope := range scopes {
	// 		s.completeReference(ctx, completions, location, scope, scope, "")
	// 	}
	// 	if err = s.completeReference(ctx, completions, location, location.RootCursor, location.Cursor.ParentScope(), ""); err != nil {
	// 		return
	// 	}
	default:
		var start asg.Node = location.Package
		if n != nil {
			start = n
		}

		if err = s.completeReference(ctx, completions, start); err != nil {
			return
		}
	}

	return //nolint: nakedret
}

func (s *server) completeBuiltin(ctx context.Context, completions *[]protocol.CompletionItem, b *asg.Builtin, prefix string) {
	name := b.Name
	if prefix != "" {
		name = prefix + "." + name
	}
	insert := b.Name
	if b.IsFunction {
		insert += "("
		argPlc := []string{}
		for idx, arg := range b.Args {
			argPlc = append(argPlc, fmt.Sprintf("${%d:%s}", idx+1, arg.String()))
		}
		insert += strings.Join(argPlc, ", ")
		insert += ")"
	}
	*completions = append(*completions, protocol.CompletionItem{
		Label: b.Name,
		Documentation: protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: b.Comment,
		},
		Kind:             protocol.ClassCompletion,
		InsertText:       insert,
		InsertTextFormat: protocol.SnippetTextFormat,
	})
}

func (s *server) completeBuiltins(ctx context.Context, completions *[]protocol.CompletionItem) {
	for _, b := range asg.BuiltinTypes {
		s.completeBuiltin(ctx, completions, b, "")
	}

	for _, pkg := range asg.BuiltinPkgs {
		// for _, b := range pkg.Builtins {
		// 	s.completeBuiltin(ctx, completions, b, pkg.Name)
		// }
		*completions = append(*completions, protocol.CompletionItem{
			Label: pkg.Name,
			Documentation: protocol.MarkupContent{
				Kind:  protocol.Markdown,
				Value: fmt.Sprintf("```cue\npackage %s (\"%s\")\n```\n%s", pkg.Name, pkg.DisplayPath, pkg.Comment),
			},
			Kind: protocol.ClassCompletion,
		})
	}
}

func (s *server) completeReference(ctx context.Context, completions *[]protocol.CompletionItem, scope asg.Node) error {
	// Reset the prefix, once we reach the parent!
	// if f, ok := startScope.Node.(*adt.Field); ok {
	// 	label := location.Doc.GetLabel(f.Label)
	// 	if prefix == "" {
	// 		prefix = label
	// 	} else {
	// 		prefix = prefix + "." + label
	// 	}

	// 	s.completeField(ctx, completions, location, startScope, prefix)
	// }

	// if parentScope == startScope {
	// 	prefix = ""
	// }

	// for iter, child := startScope.Children(); child != nil; child = iter.Next() {
	// 	s.completeReference(ctx, completions, location, child, parentScope, prefix)
	// }

	switch n := scope.(type) {
	case *asg.Package:
		for _, f := range n.Files {
			s.completeReference(ctx, completions, f)
		}
		for _, f := range n.Builtins {
			s.completeReference(ctx, completions, f)
		}
	case *asg.File:
		for _, d := range n.Decls {
			s.completeReference(ctx, completions, d)
		}

		for name, d := range n.Imports {
			s.completePackage(ctx, completions, d, name)
		}
	case *asg.Decl:
		s.completeDecl(ctx, completions, n)
		// for _, v := range n.Values {
		// 	s.completeReference(ctx, completions, v)
		// }
	case *asg.Struct:
		for _, d := range n.Decls {
			s.completeReference(ctx, completions, d)
		}
	case *asg.Value:
		for _, v := range n.Children {
			s.completeReference(ctx, completions, v)
		}
	case *asg.Builtin:
		s.completeBuiltin(ctx, completions, n, "")
		// case *asg.Reference:
		// 	if n.Referenced == nil {
		// 		labels := ResolveLabels(idx, n.Orig)
		// 		n.Referenced = c.resolveLabels(idx, n.parent, n.Orig, labels)
		// 	}
	}

	return nil
}

func (s *server) completePackage(ctx context.Context, completions *[]protocol.CompletionItem, pkg *asg.Package, name string) {
	var docBuf bytes.Buffer
	s.nodeDocMarkdown(ctx, nil, pkg, &docBuf)
	item := protocol.CompletionItem{
		Label: name,
		Documentation: protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: docBuf.String(),
		},
	}

	*completions = append(*completions, item)
}

func (s *server) completeDecl(ctx context.Context, completions *[]protocol.CompletionItem, decl *asg.Decl) {
	var docBuf bytes.Buffer
	s.nodeDocMarkdown(ctx, nil, decl, &docBuf)
	item := protocol.CompletionItem{
		Label: decl.LabelName,
		Documentation: protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: docBuf.String(),
		},
	}

	*completions = append(*completions, item)
}

func (s *server) completeField(ctx context.Context, completions *[]protocol.CompletionItem, location *cache.Location, c *cache.ADTCursor, label string) {
	field := c.Node.(*adt.Field)
	markdown := s.nodeToDocMarkdown(ctx, location, c)
	item := protocol.CompletionItem{
		Label:  label,
		Kind:   exprToCompletionKind(field.Value),
		Detail: cache.Comments(field),
		Documentation: protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: markdown,
		},
	}
	*completions = append(*completions, item)
	if item.Kind == protocol.StructCompletion {
		item.Kind = protocol.SnippetCompletion

		buf := bytes.Buffer{}
		tabs := 2
		buf.WriteString(fmt.Sprintf("${1:%sInst} : %s & ", item.Label, item.Label))
		buildSnippet(field.Value, 0, &tabs, location, &buf)

		item.InsertText = buf.String()
		item.InsertTextFormat = protocol.SnippetTextFormat
		item.Label += " snippet"

		*completions = append(*completions, item)
	}
}

func buildSnippet(expr adt.Node, indent int, tabstop *int, location *cache.Location, buf *bytes.Buffer) {
	// i := strings.Repeat("\t", indent)
	// switch c := expr.(type) {
	// case *adt.StructLit:
	// 	buf.WriteString("{\n")
	// 	for _, decl := range c.Decls {
	// 		buildSnippet(decl, indent+1, tabstop, location, buf)
	// 	}
	// 	buf.WriteString(i + "}")
	// case *adt.Field:
	// 	label := location.Doc.GetLabel(c.Label)
	// 	buf.WriteString(fmt.Sprintf("%s%s : ", i, label))
	// 	buildSnippet(c.Value, indent, tabstop, location, buf)
	// 	buf.WriteString("\n")
	// default:
	// 	declVal, _ := location.Doc.GetSubstring(c.Source().Pos(), c.Source().End())
	// 	buf.WriteString(fmt.Sprintf("${%d:%s}", *tabstop, declVal))
	// 	*tabstop++
	// }
}

func exprToCompletionKind(expr adt.Expr) protocol.CompletionItemKind {
	switch c := expr.(type) {
	case *adt.Bool, *adt.String, *adt.Num, *adt.Null, *adt.Bytes, *adt.ListLit, *adt.BasicType:
		return protocol.ValueCompletion
	case *adt.FieldReference:
		return protocol.PropertyCompletion
	case *adt.StructLit:
		return protocol.StructCompletion
	case *adt.BinaryExpr:
		// for now assume this is a valid instance and either one is actually the correct type :)
		if c.Op == adt.AndOp {
			left := exprToCompletionKind(c.X)
			right := exprToCompletionKind(c.Y)
			if left == protocol.OperatorCompletion {
				return right
			}
			return left
		}
	}

	return protocol.OperatorCompletion
}

// getEditRange computes the editRange for a completion. In case the completion area is shorter than
// the node, the oldname of the token to be completed must be provided. The latter mechanism only
// works if oldname is an ASCII string, which can be safely assumed for metric and function names.
func getEditRange(location *cache.Location, oldname string) (editRange protocol.Range, err error) {
	// editRange.Start, err = location.Doc.PosToProtocolPosition(location.Cursor.Pos())
	// if err != nil {
	// 	return
	// }

	// if oldname == "" {
	// 	editRange.End, err = location.Doc.PosToProtocolPosition(location.Cursor.End())
	// 	if err != nil {
	// 		return
	// 	}
	// } else {
	// 	editRange.End = editRange.Start
	// 	editRange.End.Character += float64(len(oldname))
	// }

	// return //nolint: nakedret
	if location.Node == nil {
		return
	}
	editRange.Start, err = location.Doc.PosToProtocolPosition(location.Node.Pos())
	if err != nil {
		return
	}

	if oldname == "" {
		editRange.End, err = location.Doc.PosToProtocolPosition(location.Node.End())
		if err != nil {
			return
		}
	} else {
		editRange.End = editRange.Start
		editRange.End.Character += float64(len(oldname))
	}

	return //nolint: nakedret
}

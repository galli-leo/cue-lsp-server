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

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/internal/adt"
	"cuelang.org/go/cue/internal/lsp/asg"
	"cuelang.org/go/cue/internal/lsp/internal/vendored/go-tools/lsp/protocol"

	"cuelang.org/go/cue/internal/lsp/cache"
	// Do not remove! Side effects of init() needed
	//_ "cuelang.org/go/cue/internal/lsp/langserver/documentation/functions_statik"
)

//nolint: gochecknoglobals
// var functionDocumentationFS = initializeFunctionDocumentation()

// func initializeFunctionDocumentation() http.FileSystem {
// 	ret, err := fs.New()
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	return ret
// }

func fmtBuf(buf *bytes.Buffer, format string, a ...interface{}) {
	_, _ = buf.WriteString(fmt.Sprintf(format+"\n", a...))
}

func (s *server) DumpCursor(ctx context.Context, c *cache.ADTCursor, doc *cache.DocumentHandle, indent int, buf *bytes.Buffer) {
	indentation := strings.Repeat("\t", indent)
	rng := fmt.Sprintf("%s-%s", c.Pos(), c.End())
	fmtBuf(buf, "%s[%T, %s]: %v", indentation, c.Node, rng, c.Node.Source())
	for iter, child := c.Children(); child != nil; child = iter.Next() {
		s.DumpCursor(ctx, child, doc, indent+1, buf)
	}
}

func (s *server) DumpASG(ctx context.Context, n asg.Node, doc *cache.DocumentHandle, indent int, buf *bytes.Buffer) {
	indentation := strings.Repeat("\t", indent)
	rng := fmt.Sprintf("%s:%s-%s", n.Pos().Filename(), n.Pos(), n.End())
	fmtBuf(buf, "%s[%T, %s]: %v", indentation, n, rng, n)
	switch t := n.(type) {
	case *asg.Reference:
		if t.Referenced != nil {
			s.DumpASG(ctx, t.Referenced, doc, indent+1, buf)
		}
	case *asg.Value:
		for _, child := range t.Children {
			fmtBuf(buf, "%s[%T, %s]: %v", indentation, child, child.Pos(), child)
		}
	}
	if n.Parent() != nil {
		s.DumpASG(ctx, n.Parent(), doc, indent+1, buf)
	}
	// for iter, child := c.Children(); child != nil; child = iter.Next() {
	// 	s.DumpCursor(ctx, child, doc, indent+1, buf)
	// }
}

// Hover shows documentation on hover
// required by the protocol.Server interface
func (s *server) Hover(ctx context.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	location, err := s.cache.Find(&params.TextDocumentPositionParams)
	if err != nil || location.Node == nil {
		return nil, nil
	}

	adtDump := bytes.Buffer{}
	adtDump.WriteString("ASGDump:\n")
	s.DumpASG(ctx, location.Node, location.Doc, 1, &adtDump)
	s.client.LogMessage(ctx, &protocol.LogMessageParams{
		Type:    protocol.Info,
		Message: adtDump.String(),
	})

	astDump := bytes.Buffer{}
	astDump.WriteString("ASTDump:\n")
	switch n := location.Node.(type) {
	case *asg.Decl:
		ast.Walk(n.Decl, func(child ast.Node) bool {
			fmtBuf(&astDump, "[%T, %s]: %v", child, child.Pos(), child)
			return true
		}, nil)
	case *asg.Value:
		ast.Walk(n.Orig, func(child ast.Node) bool {
			fmtBuf(&astDump, "[%T, %s]: %v", child, child.Pos(), child)
			return true
		}, nil)
	}
	s.client.LogMessage(ctx, &protocol.LogMessageParams{
		Type:    protocol.Info,
		Message: astDump.String(),
	})

	markdown := bytes.Buffer{}

	// markdown = s.nodeToDocMarkdown(ctx, location, location.Cursor)
	s.nodeDocMarkdown(ctx, location.Doc, location.Node, &markdown)

	hoverRange, err := getEditRange(location, "")
	if err != nil {
		return nil, nil
	}

	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  "markdown",
			Value: markdown.String(),
		},
		Range: hoverRange,
	}, nil
}

func (s *server) nodeDocMarkdown(ctx context.Context, doc *cache.DocumentHandle, node asg.Node, buf *bytes.Buffer) { //nolint: golint
	switch n := node.(type) {
	case *asg.Reference:
		if n.Referenced != nil {
			s.nodeDocMarkdown(ctx, doc, n.Referenced, buf)
		}
	case *asg.Decl:
		cfStart(buf)
		// buf.WriteString(fmt.Sprintf("%s : ", n.LabelName))
		// isFirst := true
		// for _, val := range n.Values {
		// 	if !isFirst {
		// 		buf.WriteString(" & ")
		// 	}
		// 	isFirst = false
		// 	cont, _ := doc.GetSubstring(val.Pos(), val.End())
		// 	buf.WriteString(cont)
		// }
		b, _ := format.Node(n.Decl, format.Simplify())
		buf.Write(b)
		cfEnd(buf)
		for _, group := range ast.Comments(n.Decl) {
			buf.WriteString("\n" + group.Text())
		}
	case *asg.Builtin:
		buf.WriteString(n.Comment)
	}
}

// nolint:funlen
func (s *server) nodeToDocMarkdown(ctx context.Context, location *cache.Location, c *cache.ADTCursor) string { //nolint: golint
	var ret bytes.Buffer

	// switch n := c.Node.(type) {
	// case *adt.Field:
	// 	name := location.Doc.GetLabel(n.Label)
	// 	decls := location.RootCursor.ResolveLabel(n.Label)
	// 	if len(decls) > 0 {
	// 		expr, comments := cache.Unify(decls)
	// 		if err := declaration(name, expr, comments, location, &ret); err != nil {
	// 			return ""
	// 		}
	// 	} else {
	// 		expr := n.Value
	// 		comments := c.Node.Source().Comments()
	// 		if err := declaration(name, expr, comments, location, &ret); err != nil {
	// 			return ""
	// 		}
	// 	}
	// case *adt.FieldReference:
	// 	name := location.Doc.GetLabel(n.Label)
	// 	decls := location.RootCursor.ResolveLabel(n.Label)
	// 	if len(decls) > 0 {
	// 		expr, comments := cache.Unify(decls)
	// 		if err := declaration(name, expr, comments, location, &ret); err != nil {
	// 			return ""
	// 		}
	// 	} else {
	// 		return ""
	// 	}

	// case *adt.SelectorExpr:
	// 	decls, name := location.RootCursor.ResolveSelector(n, location.Doc)
	// 	if len(decls) > 0 {
	// 		expr, comments := cache.Unify(decls)
	// 		if err := declaration(name, expr, comments, location, &ret); err != nil {
	// 			return ""
	// 		}
	// 	} else {
	// 		return ""
	// 	}
	// default:
	// }

	return ret.String()
}

func cfStart(buf *bytes.Buffer) error {
	_, err := buf.WriteString("```cue\n")
	return err
}

func cfEnd(buf *bytes.Buffer) error {
	_, err := buf.WriteString("\n```")
	return err
}

func declExpr(expr adt.Expr, location *cache.Location, buf *bytes.Buffer) error {
	switch e := expr.(type) {
	case *adt.BinaryExpr:
		if err := declExpr(e.X, location, buf); err != nil {
			return err
		}
		if _, err := buf.WriteString(fmt.Sprintf(" %s ", e.Op.String())); err != nil {
			return err
		}
		return declExpr(e.Y, location, buf)
	default:
		// better hope for some sauce
		node := e.Source()
		if node == nil {
			_, err := buf.WriteString("invalid")
			return err
		}

		contents, err := location.Doc.GetSubstring(node.Pos(), node.End())
		if err != nil {
			return err
		}
		_, err = buf.WriteString(contents)
		return err
	}
	return nil
}

func declaration(name string, expr adt.Expr, comments []*ast.CommentGroup, location *cache.Location, buf *bytes.Buffer) error {
	if err := cfStart(buf); err != nil {
		return err
	}

	if _, err := buf.WriteString(fmt.Sprintf("%s : ", name)); err != nil {
		return nil
	}

	if err := declExpr(expr, location, buf); err != nil {
		return err
	}

	if err := cfEnd(buf); err != nil {
		return err
	}

	for _, group := range comments {
		if _, err := buf.WriteString(fmt.Sprintf("\n%s", group.Text())); err != nil {
			return err
		}
	}

	return nil
}

package lsp

import (
	"bytes"
	"context"
	"fmt"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/internal/lsp/asg"
	"cuelang.org/go/cue/internal/lsp/cache"
	"cuelang.org/go/cue/internal/lsp/internal/vendored/go-tools/lsp/protocol"
)

func (s *server) completeDecls(ctx context.Context, completions *[]protocol.CompletionItem, location *cache.Location) {
	c := &fieldCompletion{
		s:           s,
		ctx:         ctx,
		completions: completions,
		start:       location.Package,
	}
	if parent := asg.ParentDecl(location.Node); parent != nil {
		c.start = parent
		asg.Walk(c, parent)
	} else {
		c.topLevel = true
		asg.Walk(c, location.Package)
	}
}

type fieldCompletion struct {
	s           *server
	ctx         context.Context
	completions *[]protocol.CompletionItem
	topLevel    bool
	start       asg.Node
	ref         asg.Node
}

func (v *fieldCompletion) Direction() asg.VisitDirection {
	return asg.DownDirection
}

func (v *fieldCompletion) Node(n asg.Node) (down bool, up bool) {
	down = true
	return
}

func (v *fieldCompletion) Reference(ref *asg.Reference) (down bool, up bool) {
	down = true
	v.ref = ref.Referenced
	return
}

func (v *fieldCompletion) Decl(d *asg.Decl) (down bool, up bool) {
	if d == v.start || d == v.ref {
		down = true
		return
	}
	for _, val := range d.Values {
		v.s.completeFieldDecl(v.ctx, v.completions, d, val, v.topLevel)
	}
	return
}

func asgToAst(node asg.Node) ast.Node {
	switch n := node.(type) {
	case *asg.Reference:
		return n.Orig
	}

	return nil
}

func (s *server) completeFieldDecl(ctx context.Context, completions *[]protocol.CompletionItem, decl *asg.Decl, val asg.Node, topLevel bool) {
	var docBuf bytes.Buffer
	s.nodeDocMarkdown(ctx, nil, decl, &docBuf)
	var snippet bytes.Buffer
	if topLevel {
		snippet.WriteString(fmt.Sprintf("${1:%sInst} : %s & {\n\t$0\n}", decl.LabelName, decl.LabelName))
	} else {
		snippet.WriteString(fmt.Sprintf("%s : ${1:", decl.LabelName))
		if a := asgToAst(val); a != nil {
			b, _ := format.Node(a, format.Simplify())
			snippet.Write(b)
		}
		snippet.WriteString("}")
	}
	item := protocol.CompletionItem{
		Label: decl.LabelName,
		Documentation: protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: docBuf.String(),
		},
		Kind:             protocol.FileCompletion,
		InsertText:       snippet.String(),
		InsertTextFormat: protocol.SnippetTextFormat,
	}

	*completions = append(*completions, item)
}

type snippedBuilder struct {
	buf     bytes.Buffer
	tabstop int
}

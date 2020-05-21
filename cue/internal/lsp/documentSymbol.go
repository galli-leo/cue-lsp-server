package lsp

import (
	"context"

	"cuelang.org/go/cue/internal/adt"
	"cuelang.org/go/cue/internal/lsp/cache"
	"cuelang.org/go/cue/internal/lsp/internal/vendored/go-tools/lsp/protocol"
)

// DocumentSymbol is required by the protocol.Server interface
func (s *server) DocumentSymbol(ctx context.Context, params *protocol.DocumentSymbolParams) ([]interface{}, error) {
	// doc, err := s.cache.GetDocument(params.TextDocument.URI)
	// if err != nil {
	// 	return nil, err
	// }

	// _, _, rootCursor, err := doc.GetCompiled()
	// if err != nil {
	// 	return nil, err
	// }

	root := []interface{}{}

	// for _, symb := range s.findSymbols(ctx, rootCursor, doc) {
	// 	if symb.Kind == protocol.Operator {
	// 		symb.Kind = protocol.Struct
	// 	}
	// 	root = append(root, symb)
	// }

	return root, nil
}

func (s *server) findSymbols(ctx context.Context, c *cache.ADTCursor, doc *cache.DocumentHandle) []protocol.DocumentSymbol {
	ret := []protocol.DocumentSymbol{}
	// for iter, child := c.Children(); child != nil; child = iter.Next() {
	// 	switch n := child.Node.(type) {
	// 	case *adt.Field:
	// 		name := doc.GetLabel(n.Label)
	// 		children := s.findSymbols(ctx, child, doc)
	// 		kind := protocol.Field
	// 		if len(children) == 0 {
	// 			kind = exprToKind(n.Value)
	// 		}
	// 		comments := n.Source().Comments()
	// 		detail := ""
	// 		for _, group := range comments {
	// 			detail = detail + group.Text()
	// 		}
	// 		if len(detail) > 80 {
	// 			detail = detail[:80]
	// 		}
	// 		symb := protocol.DocumentSymbol{
	// 			Name:           name,
	// 			Detail:         detail,
	// 			Range:          rangeOfNode(child, doc),
	// 			SelectionRange: rangeOfNode(child, doc),
	// 			Kind:           kind,
	// 			Children:       children,
	// 		}
	// 		ret = append(ret, symb)
	// 	case *adt.StructLit:
	// 		ret = append(ret, s.findSymbols(ctx, child, doc)...)
	// 	case *adt.BinaryExpr:
	// 		if n.Op == adt.AndOp {
	// 			ret = append(ret, s.findSymbols(ctx, child, doc)...)
	// 		}
	// 	}
	// }

	return ret
}

func exprToKind(expr adt.Expr) protocol.SymbolKind {
	switch c := expr.(type) {
	case *adt.Bool:
		return protocol.Boolean
	case *adt.String:
		return protocol.String
	case *adt.Num:
		return protocol.Number
	case *adt.Null:
		return protocol.Null
	case *adt.Bytes:
		return protocol.String
	case *adt.FieldReference:
		return protocol.Property
	case *adt.ListLit:
		return protocol.Array
	case *adt.BasicType:
		switch c.Kind {
		case adt.BoolKind:
			return protocol.Boolean
		case adt.BytesKind, adt.StringKind:
			return protocol.String
		case adt.NumKind:
			return protocol.Number
		case adt.NullKind:
			return protocol.Null
		case adt.ListKind:
			return protocol.Array
		default:
			return protocol.Constant
		}
	case *adt.BinaryExpr:
		// for now assume this is a valid instance and either one is actually the correct type :)
		if c.Op == adt.AndOp {
			left := exprToKind(c.X)
			right := exprToKind(c.Y)
			if left == protocol.Operator {
				return right
			}
			return left
		}
	}

	return protocol.Operator
}

func rangeOfNode(c *cache.ADTCursor, doc *cache.DocumentHandle) protocol.Range {
	start, _ := doc.PosToProtocolPosition(c.Pos())
	end, _ := doc.PosToProtocolPosition(c.End())
	return protocol.Range{
		Start: start,
		End:   end,
	}
}

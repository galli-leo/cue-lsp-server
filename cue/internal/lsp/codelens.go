package lsp

import (
	"context"

	"cuelang.org/go/cue/internal/lsp/internal/vendored/go-tools/lsp/protocol"
)

// CodeLens is required by the protocol.Server interface
func (s *server) CodeLens(ctx context.Context, params *protocol.CodeLensParams) ([]protocol.CodeLens, error) {
	// doc, err := s.cache.GetDocument(params.TextDocument.URI)
	// if err != nil {
	// 	return nil, err
	// }

	// // _, _, rootCursor, err := doc.GetCompiled()
	// // if err != nil {
	// // 	return nil, err
	// // }

	// // root := []protocol.CodeLens{}

	// // for iter, child := rootCursor.Children(); child != nil; child = iter.Next() {
	// // 	root = append(root, protocol.CodeLens{
	// // 		Range: rangeOfNode(child, doc),
	// // 		Command: protocol.Command{
	// // 			Title:     "Copy JSON",
	// // 			Command:   "copy",
	// // 			Arguments: []interface{}{"todo"},
	// // 		},
	// // 	})
	// // }

	// return root, nil
	return nil, nil
}

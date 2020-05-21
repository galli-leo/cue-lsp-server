package cache

import (
	"math"
	"reflect"
	"strings"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/internal/adt"
	"cuelang.org/go/cue/token"
)

type PosRange interface {
	// Start
	Pos() token.Pos
	// End
	End() token.Pos
}

// TODO: We assume tokens come from the same file here!
func BeforeEqual(a token.Pos, b token.Pos) bool {
	return a.Offset() <= b.Offset()
}

func Contains(r PosRange, pos token.Pos) bool {
	return BeforeEqual(r.Pos(), pos) && BeforeEqual(pos, r.End())
}

func Length(r PosRange) int {
	return r.End().Offset() - r.Pos().Offset()
}

type ADTCursor struct {
	Node            adt.Node
	Parent          *ADTCursor
	FirstChild      *ADTCursor
	LastChild       *ADTCursor
	NextSibling     *ADTCursor
	PreviousSibling *ADTCursor

	mapping map[adt.Node]*ADTCursor
}

func createADTCursor(root adt.Node, mapping map[adt.Node]*ADTCursor) *ADTCursor {
	if ret, ok := mapping[root]; ok {
		return ret
	}

	rc := &ADTCursor{
		Node:    root,
		mapping: mapping,
	}
	mapping[root] = rc

	switch n := root.(type) {
	case *adt.StructLit:
		for _, decl := range n.Decls {
			rc.AddChild(decl)
		}
	case *adt.ListLit:
		for _, elem := range n.Elems {
			rc.AddChild(elem)
		}
	case *adt.Conjunction:
		for _, val := range n.Values {
			rc.AddChild(val)
		}
	case *adt.Field:
		rc.AddChild(n.Value)
	case *adt.BinaryExpr:
		rc.AddChild(n.X)
		rc.AddChild(n.Y)
	case *adt.UnaryExpr:
		rc.AddChild(n.X)
	case *adt.SelectorExpr:
		rc.AddChild(n.X)
	}

	return rc
}

func CreateADTCursor(root adt.Node) *ADTCursor {
	return createADTCursor(root, make(map[adt.Node]*ADTCursor))
}

func CreateFromArc(root adt.Arc) *ADTCursor {
	rootNode := root.Conjuncts[0].X
	return CreateADTCursor(rootNode)
}

func (c *ADTCursor) AddChild(root adt.Node) *ADTCursor {
	child := createADTCursor(root, c.mapping)
	if c.FirstChild == nil {
		c.FirstChild = child
		c.LastChild = child
	} else {
		c.LastChild.NextSibling = child
		child.PreviousSibling = c.LastChild
		c.LastChild = child
	}

	child.Parent = c
	return child
}

func (c *ADTCursor) Source() ast.Node {
	ret := c.Node.Source()
	if ret == nil {
		return nil
	}
	if reflect.ValueOf(ret).IsNil() {
		return nil
	}
	return ret
}

func (c *ADTCursor) Pos() token.Pos {
	if c.Source() != nil {
		return c.Source().Pos()
	}
	return token.NoPos
}

func (c *ADTCursor) End() token.Pos {
	if c.Source() != nil {
		return c.Source().End()
	}
	return token.NoPos
}

func (c *ADTCursor) Contains(pos token.Pos) bool {
	return Contains(c, pos)
}

func (c *ADTCursor) Length() int {
	return Length(c)
}

type ADTCursorIter interface {
	Next() *ADTCursor
}

type childIter struct {
	parent *ADTCursor
	curr   *ADTCursor
}

func (i *childIter) Next() *ADTCursor {
	i.curr = i.curr.NextSibling
	return i.curr
}

func (c *ADTCursor) Children() (ADTCursorIter, *ADTCursor) {
	iter := &childIter{
		parent: c,
		curr:   c.FirstChild,
	}
	return iter, iter.curr
}

type siblingIter struct {
	start     *ADTCursor
	curr      *ADTCursor
	backwards bool
}

func (i *siblingIter) Next() *ADTCursor {
	i.curr = i.curr.NextSibling
	if !i.backwards && i.curr == nil {
		i.backwards = true
		i.curr = i.start.Parent.FirstChild
	}
	if i.backwards && i.curr == i.start {
		return nil
	}

	return i.curr
}

func (c *ADTCursor) Siblings() (ADTCursorIter, *ADTCursor) {
	iter := &siblingIter{
		start: c,
		curr:  c,
	}
	return iter, iter.Next()
}

type graphIter struct {
	bfs   bool
	stack []*ADTCursor
	seen  map[*ADTCursor]struct{}
}

func (i *graphIter) top() *ADTCursor {
	if len(i.stack) > 0 {
		return i.stack[len(i.stack)-1]
	}
	return nil
}

func (i *graphIter) pop() {
	if len(i.stack) > 0 {
		i.stack = i.stack[:len(i.stack)-1]
	}
}

func (i *graphIter) push(c *ADTCursor) {
	i.stack = append(i.stack, c)
}

func (i *graphIter) Next() *ADTCursor {
	curr := i.top()
	if curr == nil {
		return nil
	}
	i.seen[curr] = struct{}{}
	hasChildren := false
	for iter, child := curr.Children(); child != nil; child = iter.Next() {
		if _, ok := i.seen[child]; !ok {
			i.push(child)
			hasChildren = true
		}
	}
	if hasChildren {
		return i.Next()
	}
	i.pop()
	return curr
}

func (c *ADTCursor) DFS() (ADTCursorIter, *ADTCursor) {
	iter := &graphIter{
		stack: []*ADTCursor{},
		seen:  make(map[*ADTCursor]struct{}),
	}
	iter.push(c)
	return iter, iter.Next()
}

func (c *ADTCursor) SmallestSurroundingNode(pos token.Pos) *ADTCursor {
	best := math.MaxInt64
	var ret *ADTCursor = nil
	for iter, child := c.DFS(); child != nil; child = iter.Next() {
		// fmt.Printf("[%T, %s]: %v, %d\n", child.Node, child.Pos().String(), child.Node, child.Length())
		if child.Contains(pos) && child.Length() < best {
			best = child.Length()
			ret = child
		}
	}
	return ret
}

func (c *ADTCursor) GetDecls() map[int][]*ADTCursor {
	ret := make(map[int][]*ADTCursor)
	for iter, child := c.DFS(); child != nil; child = iter.Next() {
		switch decl := child.Node.(type) {
		case *adt.Field:
			feat := decl.Label.Index()
			existing, ok := ret[feat]
			if !ok {
				ret[feat] = []*ADTCursor{child}
			} else {
				ret[feat] = append(existing, child)
			}
		}
	}

	return ret
}

func (c *ADTCursor) ResolveLabel(feat adt.Feature) []*ADTCursor {
	allDecls := c.GetDecls()
	ret := []*ADTCursor{}
	if combined, ok := allDecls[feat.Index()]; ok {
		for _, node := range combined {
			ret = append(ret, node)
		}
	}
	return ret
}

func (c *ADTCursor) ResolveLabels(feat ...adt.Feature) []*ADTCursor {
	if len(feat) == 1 {
		return c.ResolveLabel(feat[0])
	}

	allDecls := c.GetDecls()
	ret := []*ADTCursor{}
	if combined, ok := allDecls[feat[0].Index()]; ok {
		for _, node := range combined {
			ret = append(ret, node.ResolveLabels(feat[1:]...)...)
		}
	}
	return ret
}

func (c *ADTCursor) ResolveSelector(sel *adt.SelectorExpr, d *DocumentHandle) ([]*ADTCursor, string) {
	tmp := []adt.Feature{}
	names := []string{}
	// curr := sel
	// for curr != nil {
	// 	switch n := curr.X.(type) {
	// 	case *adt.SelectorExpr:
	// 		tmp = append([]adt.Feature{curr.Sel}, tmp...)
	// 		names = append([]string{d.GetLabel(curr.Sel)}, names...)
	// 		curr = n
	// 	case *adt.FieldReference:
	// 		tmp = append([]adt.Feature{n.Label, curr.Sel}, tmp...)
	// 		names = append([]string{d.GetLabel(n.Label), d.GetLabel(curr.Sel)}, names...)
	// 		curr = nil
	// 	default:
	// 		curr = nil
	// 	}
	// }

	return c.ResolveLabels(tmp...), strings.Join(names, ".")
}

func (c *ADTCursor) ParentScope() *ADTCursor {
	parent := c.Parent
	if parent == nil {
		return c
	}
	switch parent.Node.(type) {
	case *adt.Field:
		return parent
	default:
		return parent.ParentScope()
	}
}

func Comments(field *adt.Field) string {
	tmp := []string{}
	for _, group := range field.Src.Comments() {
		tmp = append(tmp, group.Text())
	}
	return strings.Join(tmp, "\n")
}

func Unify(decls []*ADTCursor) (adt.Expr, []*ast.CommentGroup) {
	var expr adt.Expr = nil
	comments := []*ast.CommentGroup{}

	for _, single := range decls {
		curExpr := single.Node.(*adt.Field).Value
		if expr == nil {
			expr = curExpr
		} else {
			expr = &adt.BinaryExpr{
				Op: adt.AndOp,
				X:  expr,
				Y:  curExpr,
			}
		}
		comments = append(comments, single.Source().Comments()...)
	}

	return expr, comments
}

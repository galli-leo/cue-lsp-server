package asg

import (
	"fmt"
	"path"
	"strconv"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/ast/astutil"
	"cuelang.org/go/cue/build"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/cue/parser"
)

type Compiler struct {
	LoadConfig *load.Config
}

func NewCompiler(config *load.Config) *Compiler {
	return &Compiler{
		LoadConfig: config,
	}
}

type pkgIdx struct {
	pkg      *Package
	complete bool
}

type index struct {
	packages map[string]*pkgIdx
	err      errors.Error
}

func (i *index) addPackage(p *Package) {
	i.packages[p.Dir] = &pkgIdx{
		pkg:      p,
		complete: false,
	}
}

func (i *index) complete(p *Package) {
	i.packages[p.Dir].complete = true
}

func (i *index) addErr(err error) {
	i.err = errors.Append(i.err, errors.Promote(err, ""))
}

func (c *Compiler) newIndex() *index {
	return &index{
		packages: make(map[string]*pkgIdx),
		err:      nil,
	}
}

func (c *Compiler) CompileFile(filename string) (*Package, error) {

	idx := c.newIndex()

	c.LoadConfig.ParseFile = func(name string, src interface{}) (*ast.File, error) {
		file, err := parser.ParseFile(name, src, parser.AllErrors, parser.ParseComments)
		idx.addErr(err)
		return file, nil
	}

	insts := load.Instances([]string{filename}, c.LoadConfig)

	// For now, assume only one instance
	inst := insts[0]

	if inst == nil {
		return nil, errors.New("failed to load any instance for path: " + filename)
	}

	pkg := c.compileInstance(idx, inst)
	return pkg, idx.err
}

func (c *Compiler) compileInstance(idx *index, inst *build.Instance) *Package {
	path := inst.Dir
	if existing, ok := idx.packages[path]; ok {
		if existing.complete {
			return existing.pkg
		}

		idx.addErr(errors.New(fmt.Sprintf("package %s is in a reference cycle", inst.DisplayPath)))
		return nil
	}

	// Otherwise, we have to make the package ourselves.
	pkg := &Package{
		DisplayPath: inst.DisplayPath,
		Dir:         inst.Dir,
		Name:        inst.PkgName,
	}
	idx.addPackage(pkg)

	// First compile imports
	for _, imported := range inst.Imports {
		c.compileInstance(idx, imported)
	}

	// Next compile all files in package
	// TODO: unify stuff across file boundaries
	for _, file := range inst.Files {
		astutil.Resolve(file, nil)
		f := c.compileFile(idx, inst, pkg, file)
		pkg.Files = append(pkg.Files, f)
	}

	// Finally, resolve all references
	c.resolveReferences(idx, pkg)

	idx.complete(pkg)

	return pkg
}

func (c *Compiler) compileFile(idx *index, inst *build.Instance, parent *Package, file *ast.File) *File {
	f := &File{
		File:    file,
		parent:  parent,
		Imports: make(map[string]*Package),
	}

	for _, imp := range file.Imports {
		id, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue // quietly ignore the error
		}

		name := path.Base(id)
		if imp.Name != nil {
			name, _, err = ast.LabelName(imp.Name)
			idx.addErr(err)
		}

		if pkg, ok := BuiltinPkgs[id]; ok {
			f.Imports[name] = pkg
		} else if impInst := inst.LookupImport(id); impInst != nil {
			pkg := c.compileInstance(idx, impInst)
			if _, ok := f.Imports[name]; ok {
				idx.addErr(errors.Newf(imp.Pos(), "identifier %s already used for another import", name))
			} else {
				f.Imports[name] = pkg
			}
		} else {
			idx.addErr(Newf(imp, "unable to find import with path %s", id))
		}
	}

	for _, decl := range file.Decls {
		c.compileDecl(idx, f, decl)
	}

	return f
}

func (c *Compiler) compileStruct(idx *index, parent Node, structLit *ast.StructLit) *Struct {
	s := &Struct{
		StructLit: structLit,
		parent:    parent,
	}

	for _, decl := range structLit.Elts {
		c.compileDecl(idx, s, decl)
	}

	return s
}

func (c *Compiler) compileDecl(idx *index, parent DeclStore, decl ast.Decl) {
	d := &Decl{
		Decl:   decl,
		parent: parent,
	}

	switch n := decl.(type) {
	case *ast.Package:
		return
	case ast.Expr:
		// TODO: compileExpr
	case *ast.Field:
		label, _, err := ast.LabelName(n.Label)
		idx.addErr(err)
		d.Labels = []Node{c.compileValue(idx, d, n.Label)}
		d.LabelName = label
		d.Values = c.compileExpr(idx, d, n.Value)
	}

	decls := parent.Declarations()
	found := false
	for _, existing := range *decls {
		// Preexisting declaration!
		if existing.LabelName == d.LabelName {
			existing.Labels = append(existing.Labels, d.Labels...)
			existing.Values = append(existing.Values, d.Values...)
			found = true
		}
	}
	if !found {
		*decls = append(*decls, d)
	}
}

func (c *Compiler) compileExpr(idx *index, parent Node, expr ast.Expr) []Node {
	ret := []Node{}

	switch n := expr.(type) {
	case *ast.SelectorExpr, *ast.Ident:

		ref := &Reference{
			Orig:       expr,
			parent:     parent,
			Referenced: nil,
		}
		ret = []Node{ref}
	case *ast.StructLit:
		s := c.compileStruct(idx, parent, n)
		ret = []Node{s}
	case *ast.BinaryExpr:
		lhs, rhs := n.X, n.Y
		ret = append(c.compileExpr(idx, parent, lhs), c.compileExpr(idx, parent, rhs)...)
	default:
		ret = []Node{c.compileValue(idx, parent, n)}
	}

	return ret
}

// TODO: Use a visitor for this.
func (c *Compiler) resolveReferences(idx *index, node Node) {
	switch n := node.(type) {
	case *Package:
		for _, f := range n.Files {
			c.resolveReferences(idx, f)
		}
	case *File:
		for _, d := range n.Decls {
			c.resolveReferences(idx, d)
		}
	case *Decl:
		for _, v := range n.Values {
			c.resolveReferences(idx, v)
		}
	case *Struct:
		for _, d := range n.Decls {
			c.resolveReferences(idx, d)
		}
	case *Value:
		for _, v := range n.Children {
			c.resolveReferences(idx, v)
		}
	case *Reference:
		if n.Referenced == nil {
			labels := ResolveLabels(idx, n.Orig)
			n.Referenced = c.resolveLabels(idx, n.parent, n.Orig, labels)
		}
	}

}

func (c *Compiler) resolveLabels(idx *index, start Node, expr ast.Expr, labels []string) Node {
	if len(labels) > 0 {
		init := labels[0]
		initNode := start.ResolveUp(init)
		if initNode != nil {
			next := initNode
			cur := next
			for _, label := range labels[1:] {
				next = cur.ResolveDown(label)
				if next == nil {
					idx.addErr(errors.Newf(expr.Pos(), "unresolved reference %s", label))
					return cur
				}
				cur = next
			}
			return cur
		} else {
			if len(labels) == 1 {
				if ret := c.resolveBuiltin(idx, init); ret != nil {
					return ret
				}
			}
			idx.addErr(errors.Newf(expr.Pos(), "unresolved reference %s", init))
		}
	} else {
		idx.addErr(errors.Newf(expr.Pos(), "no labels in expression %T %v", expr, expr))
	}

	return nil
}

func (c *Compiler) resolveBuiltin(idx *index, label string) Node {
	if builtin, ok := BuiltinTypes[label]; ok {
		return builtin
	}

	return nil
}

func (c *Compiler) compileValue(idx *index, parent Node, node ast.Node) Node {
	v := &Value{
		Orig:   node,
		parent: parent,
	}

	switch n := node.(type) {
	case *ast.Interpolation:
		for _, child := range n.Elts {
			v.Children = append(v.Children, c.compileExpr(idx, v, child)...)
		}
	case *ast.ListLit:
		for _, child := range n.Elts {
			v.Children = append(v.Children, c.compileExpr(idx, v, child)...)
		}
	case *ast.ListComprehension:
		v.Children = append(v.Children, c.compileExpr(idx, v, n.Expr)...)
		// for _, clause := range n.Clauses {
		// 	v.Children = append(v.Children, c.compileExpr(idx, v)...)
		// }
	case *ast.ParenExpr:
		v.Children = append(v.Children, c.compileExpr(idx, v, n.X)...)
	case *ast.CallExpr:
		for _, child := range n.Args {
			v.Children = append(v.Children, c.compileExpr(idx, parent, child)...)
		}
		v.Children = append(v.Children, c.compileExpr(idx, parent, n.Fun)...)
	}

	return v
}

package asg

import (
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/token"
)

// Most basic type
type Node interface {
	// Search the whole graph for the node which most closely encompasses pos.
	// For example, consider the following CUE file:
	//		parent : {
	//			sub : string
	//		}
	// When calling Find on parent with a position that points to sub, sub will be returned,
	// even though, the position is considered to be inside parent as well.
	Find(pos token.Pos) Node

	// Starting position of this Node.
	Pos() token.Pos
	// Ending position of this Node.
	End() token.Pos

	// Parent of this Node.
	// Every type except Package will return a non nil value here.
	Parent() Node

	// Methods for resolving labels. Implementation is found in resolve.go
	//
	// TODO: Scrap this and use the more flexible visitor for this.
	//
	// Recursively go up the graph, until we find a Decl with the given label.
	// Returns nil if not found.
	ResolveUp(label string) Node

	ResolveDown(label string) Node
}

// Package is the top level type for the asg and as such always at its root.
// However, a Package must not necessarily be at the root, for example when a file imports another package.
// Nevertheless, it's Parent() will always return nil, regardless of whether it was imported or not.
type Package struct {
	// The files making up this package
	Files []*File
	// If this Package is a builtin package, array of builtins.
	Builtins []*Builtin

	// Path to display to the user
	DisplayPath string
	// Directory where the package is found, empty for builtin packages
	Dir string
	// Name of the package, usually just Path.Base(Dir)
	Name string
	// Doc comment for the whole package, at the moment only used by builtin packages.
	Comment string
}

func (p *Package) Find(pos token.Pos) Node {
	for _, file := range p.Files {
		if ret := file.Find(pos); ret != nil {
			return ret
		}
	}
	return nil
}

func (p *Package) Parent() Node {
	return nil
}

func (p *Package) Pos() token.Pos {
	return token.NoPos
}

func (p *Package) End() token.Pos {
	return token.NoPos
}

// Represents a file in a Package.
type File struct {
	File *ast.File

	// All declarations found in this file.
	Decls []*Decl
	// Packages that are imported.
	// TODO: Create an ImportedPackage type, so that we can have better mapping from name to Package?
	// e.g. we should include doc comment next to a package import.
	Imports map[string]*Package

	parent Node
}

func (f *File) Find(pos token.Pos) Node {
	// First check file, then packages
	// Hopefully this prevents cycle
	for _, decl := range f.Decls {
		if ret := decl.Find(pos); ret != nil {
			return ret
		}
	}
	for _, pkg := range f.Imports {
		if ret := pkg.Find(pos); ret != nil {
			return ret
		}
	}
	if Contains(f.File, pos) {
		return f
	}
	return nil
}

func (f *File) Parent() Node {
	return f.parent
}

func (f *File) Pos() token.Pos {
	return f.File.Pos()
}

func (f *File) End() token.Pos {
	return f.File.End()
}

type Struct struct {
	StructLit *ast.StructLit
	Decls     []*Decl
	parent    Node
}

func (s *Struct) Find(pos token.Pos) Node {
	if Contains(s.StructLit, pos) {
		for _, decl := range s.Decls {
			if ret := decl.Find(pos); ret != nil {
				return ret
			}
		}
		return s
	}
	return nil
}

func (s *Struct) Parent() Node {
	return s.parent
}

func (s *Struct) Pos() token.Pos {
	return s.StructLit.Pos()
}

func (s *Struct) End() token.Pos {
	return s.StructLit.End()
}

type Decl struct {
	Decl ast.Decl
	// TODO: Is this really necessary? We shouldn't have multiple labels mapping to the same name.
	Labels    []Node
	LabelName string
	// May be a unification of multiple values!
	// TODO: Implement unification here as well?
	Values []Node
	parent Node
}

func (d *Decl) Find(pos token.Pos) Node {
	for _, label := range d.Labels {
		if ret := label.Find(pos); ret != nil {
			return ret
		}
	}
	for _, val := range d.Values {
		if ret := val.Find(pos); ret != nil {
			return ret
		}
	}
	if Contains(d.Decl, pos) {
		return d
	}

	return nil
}

func (d *Decl) Parent() Node {
	return d.parent
}

func (d *Decl) Pos() token.Pos {
	return d.Decl.Pos()
}

func (d *Decl) End() token.Pos {
	return d.Decl.End()
}

// A Reference references another node in the graph.
// This is useful for e.g. showing information about a declaration when hovering over its reference.
//
// References are either ast.Ident after a colon or ast.SelectorExpr
type Reference struct {
	Orig ast.Expr
	// Since any implementation of Node is a pointer, this works nicely.
	// I.e. if the Referenced Node changes, we will automatically be notified of that.
	Referenced Node
	parent     Node
}

func (r *Reference) Find(pos token.Pos) Node {
	if Contains(r.Orig, pos) {
		return r
	}
	return nil
}

func (r *Reference) Parent() Node {
	return r.parent
}

func (r *Reference) Pos() token.Pos {
	return r.Orig.Pos()
}

func (r *Reference) End() token.Pos {
	return r.Orig.End()
}

// Value represents any other ast.Node not falling into the other categories.
// Since we don't really need to have them specially separated at the moment for the LSP, we have one type for all.
type Value struct {
	Orig     ast.Node
	Children []Node
	parent   Node
}

func (v *Value) Find(pos token.Pos) Node {
	for _, child := range v.Children {
		if ret := child.Find(pos); ret != nil {
			return ret
		}
	}
	if Contains(v.Orig, pos) {
		return v
	}
	return nil
}

func (v *Value) Parent() Node {
	return v.parent
}

func (v *Value) Pos() token.Pos {
	return v.Orig.Pos()
}

func (v *Value) End() token.Pos {
	return v.Orig.End()
}

// Builtin represents a builtin function, constant or type
type Builtin struct {
	Name    string
	Comment string
	// TODO: Have a separate FunctionBuiltin?
	IsFunction bool
	Args       []cue.ValKind
}

func (v *Builtin) Find(pos token.Pos) Node {
	return nil
}

func (v *Builtin) Parent() Node {
	return nil
}

func (v *Builtin) Pos() token.Pos {
	return token.NoPos
}

func (v *Builtin) End() token.Pos {
	return token.NoPos
}

package asg

// Direction in which the Visit is going
// Can be either Up (UpDirection) or Down (DownDirection)
type VisitDirection int32

const (
	UpDirection VisitDirection = iota
	DownDirection
)

// The base type for any Visitor interface or implementation.
// A visitor is anything that will receive callbacks for wandering the ASG up and down.
//
// There is an interface for every type of ASG to allow for much easier traversing of it.
// If you only care about packages in the ASG, you can just implement PackageVisitor.
//
// NOTE: Only the more specific "callback" is called.
// For example, if you implement both NodeVisitor and PackageVisitor, NodeVisitor will be called for every node except Packages, while PackageVisitor will be called for every Package.
type ASGVisitor interface {
	// In it's most basic form, a visitor may only indicate which direction the next visit should occur.
	Direction() VisitDirection
}

// Extension to ASG visitor.
// An implementation will be notified of any visited node with the Node callback.
type NodeVisitor interface {
	ASGVisitor

	// Called, whenever a node is visited.
	// Returns whether the two directions are ok to be continued along.
	// NOTE: Direction() specifies which one to use, return value here indicates whether it should continue or not.
	Node(node Node) (down bool, up bool)
}

// Notifies implementation whenever a Package is encountered
type PackageVisitor interface {
	ASGVisitor

	// return value indicates whether to visit files, builtins and parent (depends on Direction()).
	Package(pkg *Package) (files bool, builtins bool, up bool)
}

// Notifies whenever a Struct or File is encountered.
type DeclStoreVisitor interface {
	ASGVisitor

	DeclStore(store DeclStore) (down bool, up bool)
}

type FileVisitor interface {
	ASGVisitor

	File(file *File) (decls bool, imports bool, up bool)
}

type StructVisitor interface {
	ASGVisitor

	Struct(s *Struct) (decls bool, up bool)
}

type DeclVisitor interface {
	ASGVisitor

	Decl(decl *Decl) (down bool, up bool)
}

type ReferenceVisitor interface {
	ASGVisitor

	Reference(ref *Reference) (down bool, up bool)
}

type ValueVisitor interface {
	ASGVisitor

	Value(val *Value) (children bool, up bool)
}

type BuiltinVisitor interface {
	ASGVisitor

	Builtin(b *Builtin)
}

// The actual type doing the walking and callback stuff.
type walker struct {
	visitor ASGVisitor
	seen    map[Node]bool
}

// Starts walking at the given node and calls the corresponding callbacks on ASGVisitor v.
func Walk(v ASGVisitor, start Node) {
	w := walker{
		visitor: v,
		seen:    make(map[Node]bool),
	}

	w.Visit(start)
}

func (w *walker) Visit(node Node) {
	if seen := w.seen[node]; seen {
		return
	}
	w.seen[node] = true

	var down, up bool
	var dir VisitDirection

	switch n := node.(type) {
	case *Package:
		var files, builtins bool
		if pv, ok := w.visitor.(PackageVisitor); ok {
			files, builtins, up = pv.Package(n)
		} else {
			down, up = w.nodeVisit(node)
		}
		dir = w.visitor.Direction()
		if dir == DownDirection {
			if down || files {
				for _, f := range n.Files {
					w.Visit(f)
				}
			}
			if down || builtins {
				for _, f := range n.Builtins {
					w.Visit(f)
				}
			}
		}

	case *File:
		var decls, imports bool
		if pv, ok := w.visitor.(FileVisitor); ok {
			decls, imports, up = pv.File(n)
		} else {
			down, up = w.declVisit(n)
		}
		dir = w.visitor.Direction()
		if dir == DownDirection {
			if down || decls {
				for _, f := range n.Decls {
					w.Visit(f)
				}
			}
			if down || imports {
				// TODO: Better stuff with imports
				for _, f := range n.Imports {
					w.Visit(f)
				}
			}
		}
	case *Decl:
		if pv, ok := w.visitor.(DeclVisitor); ok {
			down, up = pv.Decl(n)
		} else {
			down, up = w.nodeVisit(node)
		}
		dir = w.visitor.Direction()
		if dir == DownDirection {
			if down {
				for _, f := range n.Values {
					w.Visit(f)
				}
			}
		}
	case *Struct:
		if pv, ok := w.visitor.(StructVisitor); ok {
			down, up = pv.Struct(n)
		} else {
			down, up = w.declVisit(n)
		}
		dir = w.visitor.Direction()
		if dir == DownDirection {
			if down {
				for _, f := range n.Decls {
					w.Visit(f)
				}
			}
		}
	case *Value:
		if pv, ok := w.visitor.(ValueVisitor); ok {
			down, up = pv.Value(n)
		} else {
			down, up = w.nodeVisit(node)
		}
		dir = w.visitor.Direction()
		if dir == DownDirection {
			if down {
				for _, f := range n.Children {
					w.Visit(f)
				}
			}
		}
	case *Builtin:
		if pv, ok := w.visitor.(BuiltinVisitor); ok {
			pv.Builtin(n)
		} else {
			down, up = w.nodeVisit(node)
		}
		dir = w.visitor.Direction()
	case *Reference:
		if pv, ok := w.visitor.(ReferenceVisitor); ok {
			down, up = pv.Reference(n)
		} else {
			down, up = w.nodeVisit(node)
		}
		dir = w.visitor.Direction()
		if dir == DownDirection {
			if down {
				if n.Referenced != nil {
					w.Visit(n.Referenced)
				}
			}
		}
	}

	if dir == UpDirection && up {
		if parent := node.Parent(); parent != nil {
			w.Visit(parent)
		}
	}
}

func (w *walker) nodeVisit(node Node) (down bool, up bool) {
	if nodeV, ok := w.visitor.(NodeVisitor); ok {
		down, up = nodeV.Node(node)
	}
	return
}

func (w *walker) declVisit(s DeclStore) (down bool, up bool) {
	if declSV, ok := w.visitor.(DeclStoreVisitor); ok {
		down, up = declSV.DeclStore(s)
	} else {
		down, up = w.nodeVisit(s)
	}

	return
}

package asg

// Visitor for ParentDecl function below.
// Will go up the graph until it encounters a Decl.
type parentDecl struct {
	start Node
	decl  *Decl
}

func (v *parentDecl) Direction() VisitDirection {
	return UpDirection
}

func (v *parentDecl) Node(node Node) (down bool, up bool) {
	up = true
	return
}

func (v *parentDecl) Decl(d *Decl) (down bool, up bool) {
	if d == v.start {
		up = true
		return
	}
	v.decl = d
	return
}

// Traverse the graph upwards until a Decl is reached.
func ParentDecl(n Node) *Decl {
	p := parentDecl{
		start: n,
	}
	Walk(&p, n)
	return p.decl
}

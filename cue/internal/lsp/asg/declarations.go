package asg

// Since both Files and Structs may store declarations, this interface was created.
// In some scenarios we don't really care whether it's a file or a struct.
type DeclStore interface {
	Node

	// Returns a pointer to the underlying array of declarations.
	// This can be used to modify them regardless of whether we are dealing with a file or struct.
	Declarations() *[]*Decl
}

func (f *File) Declarations() *[]*Decl {
	return &f.Decls
}

func (s *Struct) Declarations() *[]*Decl {
	return &s.Decls
}

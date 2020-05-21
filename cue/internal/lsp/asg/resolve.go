package asg

func (p *Package) ResolveUp(label string) Node {
	if p.Name == label {
		return p
	}
	return nil
}

func (p *Package) ResolveDown(label string) Node {
	if p.Name == label {
		return p
	}

	for _, file := range p.Files {
		if ret := file.ResolveDown(label); ret != nil {
			return ret
		}
	}

	for _, b := range p.Builtins {
		if b.Name == label {
			return b
		}
	}

	return nil
}

func Resolve(decls DeclStore, label string) Node {
	for _, decl := range *decls.Declarations() {
		if decl.LabelName == label {
			return decl
		}
	}

	return nil
}

func (f *File) ResolveUp(label string) Node {
	if ret := Resolve(f, label); ret != nil {
		return ret
	}
	if imp, ok := f.Imports[label]; ok {
		return imp
	}
	// maybe in another file?
	if f.parent != nil {
		if pkg, ok := f.parent.(*Package); ok {
			for _, others := range pkg.Files {
				if others != f {
					if ret := Resolve(others, label); ret != nil {
						return ret
					}
				}
			}
		}
	}
	return f.parent.ResolveUp(label)
}

func (f *File) ResolveDown(label string) Node {
	if ret := Resolve(f, label); ret != nil {
		return ret
	}

	return nil // We don't want to recurse in resolve down. ResolveDown should go to next lower decls, then stop
}

func (s *Struct) ResolveUp(label string) Node {
	if ret := Resolve(s, label); ret != nil {
		return ret
	}
	return s.parent.ResolveUp(label)
}

func (s *Struct) ResolveDown(label string) Node {
	if ret := Resolve(s, label); ret != nil {
		return ret
	}

	return nil
}

func (d *Decl) ResolveUp(label string) Node {
	return d.parent.ResolveUp(label)
}

func (d *Decl) ResolveDown(label string) Node {
	for _, val := range d.Values {
		if ret := val.ResolveDown(label); ret != nil {
			return ret
		}
	}

	return nil
}

func (r *Reference) ResolveUp(label string) Node {
	return r.parent.ResolveUp(label)
}

func (r *Reference) ResolveDown(label string) Node {
	// if r.Referenced != nil {
	// 	return r.Referenced.ResolveDown(label)
	// }

	return nil
}

func (v *Value) ResolveUp(label string) Node {
	return v.parent.ResolveUp(label)
}

func (v *Value) ResolveDown(label string) Node {
	return nil // TODO: should this recurse or not?
}

func (v *Builtin) ResolveUp(label string) Node {
	return nil
}

func (v *Builtin) ResolveDown(label string) Node {
	return nil // TODO: should this recurse or not?
}

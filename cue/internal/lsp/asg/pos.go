package asg

import "cuelang.org/go/cue/token"

// Generic interface for something that provides a range starting at Pos and ending at End
type PosRange interface {
	Pos() token.Pos
	End() token.Pos
}

// Since token.Pos only supports Before, we need to have our own implementation here, that can do Equal as well.
func BeforeEqual(a token.Pos, b token.Pos) bool {
	if a.File() == nil || b.File() == nil {
		return false
	}
	return (a.Offset() <= b.Offset() && a.File().Name() == b.File().Name())
}

// Convenience function for using BeforeEqual with PosRange
func Contains(r PosRange, pos token.Pos) bool {
	return BeforeEqual(r.Pos(), pos) && BeforeEqual(pos, r.End())
}

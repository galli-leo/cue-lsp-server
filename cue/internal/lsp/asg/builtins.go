package asg

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"

	"cuelang.org/go/cue"
)

// Create a markdown cue code expression.
// Used for doc comments.
//
// TODO: Should this be somewhere shared?
func codeFenced(code string) string {
	return "```cue\n" + code + "\n```"
}

// Builtint string type
// Documentation Comment was just copy pasted from the language spec.
// TODO: Better Doc Comment?
var stringBuiltin = Builtin{
	Name: "string",
	Comment: `
CUE strings allow a richer set of escape sequences than JSON.

CUE also supports multi-line strings, enclosed by a pair of triple quotes """. The opening quote must be followed by a newline. The closing quote must also be on a newline. The whitespace directly preceding the closing quote must match the preceding whitespace on all other lines and is removed from these lines.

Strings may also contain interpolations.
` + codeFenced(`// 21-bit unicode characters
a: "\U0001F60E" // 😎

// multiline strings
b: """
Hello
World!
"""

// Interpolations allow you to evaluate any expression.
"You are \( cost - budget ) dollars over budget!"

cost ::   102
budget :: 88`),
}

// Builtint int type
// Documentation Comment was just copy pasted from the language spec.
// TODO: Better Doc Comment?
var intBuiltin = Builtin{
	Name: "int",
	Comment: `
CUE defines two kinds of numbers. Integers, denoted int, are whole, or integral, numbers. Floats, denoted float, are decimal floating point numbers.

An integer literal (e.g. 4) can be of either type, but defaults to int. A floating point literal (e.g. 4.0) is only compatible with float.

In the example, the result of b is a float and cannot be used as an int without conversion.

CUE also adds a variety of sugar for writing numbers.
` + codeFenced(`a: int
a: 4 // type int

b: number
b: 4.0 // type float

c: int
c: 4.0

d: 4  // will evaluate to type int (default)

e: [
    1_234,       // 1234
    5M,          // 5_000_000
    1.5Gi,       // 1_610_612_736
    0x1000_0000, // 268_435_456
]`),
}

// Builtint top type
// Documentation Comment was just copy pasted from the language spec.
// TODO: Better Doc Comment?
var topBuiltin = Builtin{
	Name: "top",
	Comment: "Top is represented by the underscore character `_`, lexically an identifier. Unifying any value `v` with top results `v` itself.\n" + codeFenced(`
//Expr        Result
_ &  5        5
_ &  _        _
_ & _|_      _|_
_ | _|_       _
`),
}

// Builtint bool type
// Documentation Comment was just copy pasted from the language spec.
// TODO: Better Doc Comment?
var boolBuiltin = Builtin{
	Name:    "bool",
	Comment: codeFenced("bool"),
}

// TODO: Add other builtin types!

// Creates a Builtin for a specific integer range.
// ident is the identifier used to reference the range, while code is the definition in CUE of the range.
func mkIntRange(ident string, code string) *Builtin {
	return &Builtin{
		Name: ident,
		Comment: codeFenced(ident+" : "+code) + `
Predefined identifier to restrict the bounds of integers to common values.`,
	}
}

// Initialize the builtins
func initBuiltins() map[string]*Builtin {
	builtins := map[string]*Builtin{
		"string":   &stringBuiltin,
		"__string": &stringBuiltin,
		"int":      &intBuiltin,
		"__int":    &intBuiltin,
		"bool":     &boolBuiltin,
		"__bool":   &boolBuiltin,
		"_":        &topBuiltin,
	}

	rngs := []*Builtin{
		mkIntRange("uint", ">=0"),
		mkIntRange("uint8", ">=0 & <=255"),
		mkIntRange("int8", ">=-128 & <=127"),
		mkIntRange("uint16", ">=0 & <=65536"),
		mkIntRange("int16", ">=-32_768 & <=32_767"),
		mkIntRange("rune", ">=0 & <=0x10FFFF"),
		mkIntRange("uint32", ">=0 & <=4_294_967_296"),
		mkIntRange("int32", ">=-2_147_483_648 & <=2_147_483_647"),
		mkIntRange("uint64", ">=0 & <=18_446_744_073_709_551_615"),
		mkIntRange("int64", ">=-9_223_372_036_854_775_808 & <=9_223_372_036_854_775_807"),
		mkIntRange("int128", ">=-170_141_183_460_469_231_731_687_303_715_884_105_728 & <=170_141_183_460_469_231_731_687_303_715_884_105_727"),
		mkIntRange("uint128", ">=0 & <=340_282_366_920_938_463_463_374_607_431_768_211_455"),
	}

	for _, rng := range rngs {
		builtins[rng.Name] = rng
		builtins[rng.Name] = rng
	}

	return builtins
}

// Initialize the builtin packages.
// Using cue.BuiltinPackages, we map them to the asg representations.
//
// Documentation comments are "generated" by looking up the correct source file and then using godoc to extract the comment.
// TODO: Either Create these at build time or look them up differently.
func initBuiltinPkgs() map[string]*Package {
	ret := make(map[string]*Package)
	for id, pkg := range cue.BuiltinPackages {
		p := &Package{
			DisplayPath: id,
			Name:        path.Base(id),
			Comment:     builtinDoc(id, ""),
		}
		for _, native := range pkg.Native {
			b := &Builtin{
				Name: native.Name,
			}
			if native.Const != "" {
				b.Comment = codeFenced(fmt.Sprintf("%s : %s", b.Name, native.Const))
			} else {
				args := []string{}
				for _, arg := range native.Params {
					args = append(args, arg.String())
				}

				b.IsFunction = true
				b.Args = native.Params
				b.Comment = codeFenced(fmt.Sprintf("%s(%s) %s", b.Name, strings.Join(args, ", "), native.Result.String()))
			}
			docComment := builtinDoc(id, b.Name)
			if docComment == "" {
				docComment = fmt.Sprintf("Builtin function from package `\"%s\"`", id)
			}
			b.Comment += "\n" + docComment
			p.Builtins = append(p.Builtins, b)
		}
		ret[id] = p
	}
	return ret
}

// Return the doc comment for the builtin with the given package path id and name
//
// At the moment this uses a hardcoded path for the cue source code to use godoc with.
// TODO: Either create a lookup map at build time or use a different way of reading out doc comments.
func builtinDoc(id string, name string) string {
	// Get description of a func
	folderName := filepath.Join("/Users/leonardogalli/Code/CTF/infra/new/cue/pkg", id)
	files, _ := filepath.Glob(fmt.Sprintf("%s/*.go", folderName))

	pkg := &ast.Package{
		Name:  "Any",
		Files: make(map[string]*ast.File),
	}

	for _, file := range files {
		fset := token.NewFileSet()

		// Parse src
		parsedAst, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			return ""
		}

		pkg.Files[file] = parsedAst
	}

	importPath, _ := filepath.Abs("/")
	myDoc := doc.New(pkg, importPath, doc.AllDecls)
	if name == "" {
		return myDoc.Doc
	}
	for _, theFunc := range myDoc.Funcs {
		if theFunc.Name == name {
			return theFunc.Doc
		}
	}
	for _, constVar := range myDoc.Consts {
		if constVar.Names[0] == name {
			return constVar.Doc
		}
	}
	return ""
}

var BuiltinTypes = initBuiltins()
var BuiltinPkgs = initBuiltinPkgs()

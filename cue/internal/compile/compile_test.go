// Copyright 2020 CUE Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package compile

import (
	"flag"
	"fmt"
	"testing"

	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/internal/debug"
	"cuelang.org/go/cue/internal/runtime"
	"cuelang.org/go/cue/parser"
	"cuelang.org/go/internal/cuetest"
	"cuelang.org/go/pkg/strings"
)

var (
	update = flag.Bool("update", false, "update the test files")
	todo   = flag.Bool("todo", false, "run tests marked with #todo-compile")
)

func TestCompile(t *testing.T) {
	test := cuetest.TxTarTest{
		Root:   "../../testdata/",
		Name:   "compile",
		Update: *update,
		Skip:   alwaysSkip,
		ToDo:   needFix,
	}

	if *todo {
		test.ToDo = nil
	}

	r := runtime.New()

	test.Run(t, func(t *cuetest.Test) {
		// TODO: use high-level API.
		c := compiler{index: r}

		a := t.ValidInstances()

		arc := c.compileFiles(a[0].Files...)

		// Write the results.
		t.WriteErrors(c.errs)

		for i, f := range a[0].Files {
			if i > 0 {
				fmt.Fprintln(t)
			}
			fmt.Fprintln(t, "---", t.Rel(f.Filename))
			debug.WriteNode(t, r, arc.Conjuncts[i].X)
		}
		fmt.Fprintln(t)
	})
}

var alwaysSkip = map[string]string{
	"fulleval/031_comparison against bottom": "fix bin op binding in test",
}

var needFix = map[string]string{
	"export/020":                           "builtin",
	"fulleval/027_len of incomplete types": "builtin",
	"fulleval/032_or builtin should not fail on non-concrete empty list": "builtin",
	"fulleval/053_issue312":       "builtin",
	"resolve/034_closing structs": "builtin",
	"resolve/048_builtins":        "builtin",

	"fulleval/026_don't convert incomplete errors to non-incomplete": "import",
	"fulleval/044_Issue #178":                               "import",
	"fulleval/048_don't pass incomplete values to builtins": "import",
	"fulleval/049_alias reuse in nested scope":              "import",
	"fulleval/050_json Marshaling detects incomplete":       "import",
	"fulleval/051_detectIncompleteYAML":                     "import",
	"fulleval/052_detectIncompleteJSON":                     "import",
	"fulleval/056_issue314":                                 "import",
	"resolve/013_custom validators":                         "import",
}

// TestX is for debugging. Do not delete.
func TestX(t *testing.T) {
	in := `
	`

	if strings.TrimSpace(in) == "" {
		t.Skip()
	}

	file, err := parser.ParseFile("TestX", in)
	if err != nil {
		t.Fatal(err)
	}
	r := runtime.New()
	c := compiler{index: r}

	arc := c.compileFiles(file)
	if c.errs != nil {
		t.Error(errors.Details(c.errs, nil))
	}
	t.Error(debug.NodeString(r, arc.Conjuncts[0].X))
}

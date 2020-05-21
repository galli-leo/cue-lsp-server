<!--
 Copyright 2018 The CUE Authors

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
-->
# Initial Implementation of a CUE LSP

Originally, this was forked from https://cue-review.googlesource.com/c/cue/+/5840.
However, the current implementation uses its own ast abstraction.

Some changes had to be made for certain structs for them to be accessible within the context of the lsp package.
For example, `builtin` had to be exposed.

## Project Layout

### [cue-vscode-ext](cue-vscode-ext/)

Contains a Visual Studio Code extension to use the cue LSP with.

To test it out, you can just run `code --extensionDevelopmentPath=$PWD/cue-vscode-ext` from the root of the repository, after you compiled the command line with `go build -o cue-lsp ./cmd/cue`.

### cmd/cue/cmd/lsp.go

Contains the definitions for a new command available: `lsp`:

```
lsp starts the CUE language server.

At the moment, communication is limited to stdio.

Usage:
  cue lsp [flags]

Flags:
      --config-file string   Path to yml config file. (default "cue-lsp.yml")
  -h, --help                 help for lsp

Global Flags:
  -E, --all-errors   print all available errors
  -i, --ignore       proceed in the presence of errors
  -s, --simplify     simplify output
      --strict       report errors for lossy mappings
      --trace        trace computation
  -v, --verbose      print information about progress
```

### cue/lsp/

Package that exposes a simple function to start the lsp at the moment.
For now this is needed, as the lsp resides in the `internal` directory and as such cannot be directly accessed by the command package.

### cue/internal/lsp

This contains the actual implementation of the lsp, as well as any helper packages.

Everything in here was originally taken from `promql-server`.
However, the specific implementations, were reworked for CUE (e.g. `completion.go`).

### cue/internal/lsp/internal/vendored

To use all the types, interfaces, etc. already created for the `gopls` language server, we need to clone a part of that here (since those reside in `internal` of `gopls` as well).
This was copied from the `promql-server`

`cue/internal/lsp/update_internal.sh` pulls in the latest changes from the `gopls` repository.
This was copied from the Makefile of `promql-server`.

### cue/internal/lsp/asg

This package is used to provide the lsp with information about CUE files.
The package is fairly documented, so check out the documentation there.

### cue/internal/lsp/cache

This was copied over from `promql-server`, but has seen heavy rewrites to make it work with CUE.
Still needs work to make packages work nicely.
See the documentation there, for more information (specifically `doc.go`).
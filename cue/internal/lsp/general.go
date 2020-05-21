// Copyright 2019 Tobias Guggenmos
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lsp

import (
	"context"
	"errors"

	"cuelang.org/go/cue/internal/lsp/internal/vendored/go-tools/jsonrpc2"
	"cuelang.org/go/cue/internal/lsp/internal/vendored/go-tools/lsp/protocol"
)

// Initialize handles a call from the client to initialize the server
// required by the protocol.Server interface
// nolint:funlen
func (s *server) Initialize(ctx context.Context, params *protocol.ParamInitialize) (*protocol.InitializeResult, error) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	if s.state != serverCreated {
		return nil, jsonrpc2.NewErrorf(jsonrpc2.CodeInvalidRequest, "server already initialized")
	}

	s.state = serverInitializing

	s.cache.Init()
	s.cache.LoadRootFolder(params.RootURI)
	s.cache.Logging = make(chan protocol.LogMessageParams, 100)

	// Start receiving log messages in background.
	go (func() {
		for msg := range s.cache.Logging {
			s.client.LogMessage(s.lifetime, &msg)
		}
	})()

	return &protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			TextDocumentSync: &protocol.TextDocumentSyncOptions{
				OpenClose: true,
				// Support incremental changes
				Change: 2,
			},
			HoverProvider: true,
			CompletionProvider: protocol.CompletionOptions{
				TriggerCharacters: []string{
					".", //" ", "\n", "\t", "(", ")", "[", "]", "{", "}", "+", "-", "*", "/", "!", "=", "\"", ",", "'", "\"", "`", "a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "n", "m", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z", "A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "N", "M", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z",
				},
			},
			DocumentSymbolProvider: true,
			DefinitionProvider:     true,
		},
	}, nil
}

// Initialized receives a confirmation by the client that the connection has been initialized
// required by the protocol.Server interface
func (s *server) Initialized(ctx context.Context, params *protocol.InitializedParams) (err error) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	if s.state != serverInitializing {
		return errors.New("cannot initialize server: wrong server state")
	}

	s.state = serverInitialized

	return err
}

// Shutdown receives a call from the client to shutdown the connection
// required by the protocol.Server interface
func (s *server) Shutdown(ctx context.Context) error {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	if s.state != serverInitialized {
		return jsonrpc2.NewErrorf(jsonrpc2.CodeInvalidRequest, "server not initialized")
	}

	s.state = serverShutDown

	return nil
}

// Exit ends the connection
// required by the protocol.Server interface
func (s *server) Exit(ctx context.Context) error {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	if s.state != serverShutDown {
		return jsonrpc2.NewErrorf(jsonrpc2.CodeInvalidRequest, "server not shutdown")
	}

	s.exit()

	return nil
}

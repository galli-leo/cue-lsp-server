/* --------------------------------------------------------------------------------------------
 * Copyright (c) Microsoft Corporation. All rights reserved.
 * Licensed under the MIT License. See License.txt in the project root for license information.
 * ------------------------------------------------------------------------------------------ */

import * as path from 'path';
import { workspace, ExtensionContext } from 'vscode';

import {
	LanguageClient,
	LanguageClientOptions,
	ServerOptions,
	TransportKind,
	Trace
} from 'vscode-languageclient';

let client: LanguageClient;

export function activate(context: ExtensionContext) {
	console.warn("Activating!");
	// The server is implemented in node
	let serverModule = context.asAbsolutePath(
		path.join('..', 'cue-lsp')
	);
	let configFile = context.asAbsolutePath(
		path.join('..', 'cue-lsp.yml')
	);
	console.info("Server at: " + serverModule);
	// The debug options for the server
	// --inspect=6009: runs the server in Node's Inspector mode so VS Code can attach to the server for debugging
	let debugOptions = { };

	// If the extension is launched in debug mode then the debug server options are used
	// Otherwise the run options are used
	let args = ['lsp', '--config-file', configFile];
	let serverOptions: ServerOptions = {
		run: { command: serverModule, transport: TransportKind.stdio, args : args },
		debug: {
			command: serverModule,
			transport: TransportKind.stdio,
			options: debugOptions,
			args : args
		}
	};

	// Options to control the language client
	let clientOptions: LanguageClientOptions = {
		// Register the server for plain text documents
		documentSelector: [{ scheme: 'file', language: 'cue' }],
		synchronize: {
			// Notify the server about file changes to '.clientrc files contained in the workspace
			fileEvents: workspace.createFileSystemWatcher('**/*.cue')
		}
	};

	// Create the language client and start the client.
	client = new LanguageClient(
		'cue-lsp',
		'CUE Language Server',
		serverOptions,
		clientOptions
	);
	client.trace = Trace.Verbose;

	// Start the client. This will also launch the server
	client.start();
}

export function deactivate(): Thenable<void> | undefined {
	if (!client) {
		return undefined;
	}
	return client.stop();
}

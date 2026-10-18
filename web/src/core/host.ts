// Copyright 2026 Han Li and contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/**
 * The native side of the web front end.
 *
 * A web view cannot spawn processes, read files or show native dialogs, so
 * every shell that hosts this front end (Tauri, Wails, a browser connected to
 * the development host) implements this interface. It is deliberately small
 * and stateless; everything vfox-specific lives in TypeScript.
 */

import type { ProcessRunner } from './process';

/** The file system operations the application needs. Paths are absolute. */
export interface FileSystem {
  /** Whether something exists at `path`. */
  exists(path: string): Promise<boolean>;
  /** Whether `path` is an existing directory. */
  isDirectory(path: string): Promise<boolean>;
  /** Whether `path` is an existing regular file that anybody may execute. */
  isExecutable(path: string): Promise<boolean>;
  /** The content of the UTF-8 text file `path`, or `null` if it does not exist. */
  readTextFile(path: string): Promise<string | null>;
  /**
   * Replaces the content of `path` atomically: the text is written to a
   * temporary file next to it, which is then renamed. Missing parent
   * directories are created.
   */
  writeTextFile(path: string, text: string): Promise<void>;
  /** Creates `path` and its missing parents. */
  createDirectory(path: string): Promise<void>;
  /** Creates a new, empty directory in `parent` whose name starts with `prefix`. */
  createTemporaryDirectory(parent: string, prefix: string): Promise<string>;
  /** Copies the file `from` to `to`. */
  copyFile(from: string, to: string): Promise<void>;
  /** Deletes `path` recursively, if it exists. */
  remove(path: string): Promise<void>;
  /** The temporary directory of the system, for example `/tmp`. */
  systemTemporaryDirectory(): Promise<string>;
}

/** A file type filter of a file chooser. */
export interface FileFilter {
  /** Description of the files, for example "vfox plugins". */
  readonly label: string;
  /** Accepted extensions, without dots. */
  readonly extensions: readonly string[];
}

/**
 * Desktop integration that the user interface needs: the browser, file
 * choosers, the clipboard and the accent colour.
 */
export interface PlatformServices {
  /** The accent colour of the desktop as ARGB, if it exposes one. */
  accentColor(): Promise<number | null>;
  /** Opens `url` in the default browser. */
  openUrl(url: string): Promise<void>;
  /** Lets the user choose a directory. */
  pickDirectory(initialDirectory?: string): Promise<string | null>;
  /** Lets the user choose a file, optionally restricted by `filter`. */
  pickFile(initialDirectory?: string, filter?: FileFilter): Promise<string | null>;
  /** Copies `text` to the clipboard. */
  copyText(text: string): Promise<void>;
}

/** Everything the front end needs from its native shell. */
export interface Host extends ProcessRunner, PlatformServices {
  /** A short name of the shell, such as `tauri`, shown in the about section. */
  readonly name: string;
  /** The environment variables of the application process. */
  environment(): Promise<Record<string, string>>;
  /** File system access. */
  readonly files: FileSystem;
  /** Ends the application with exit code `code`. Used by the end-to-end tests. */
  exit(code: number): Promise<void>;
  /** Writes `message` to the standard output of the application. Used by the end-to-end tests. */
  log(message: string): Promise<void>;
  /**
   * Resizes the window to `width` × `height` logical pixels. Used by the
   * screenshots scenario; rejects in shells that do not support it.
   */
  resizeWindow(width: number, height: number): Promise<void>;
}

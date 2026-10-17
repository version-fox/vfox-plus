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

package host

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// The file system operations of the host protocol: the settings file, the
// locator of vfox and the copies of local plugins.

// Exists reports whether something exists at path.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsDirectory reports whether path is an existing directory.
func IsDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// IsExecutable reports whether path is an existing regular file with any
// execute bit.
func IsExecutable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0
}

// ReadTextFile returns the content of a UTF-8 text file, or nil if it does
// not exist.
func ReadTextFile(path string) (*string, error) {
	data, err := os.ReadFile(path) //nolint:gosec // The front end chooses the files of the user.
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	text := string(data)
	return &text, nil
}

// WriteTextFile replaces the content of path atomically: the text goes to a
// temporary file next to it, which is then renamed. Missing parents are
// created.
func WriteTextFile(path, text string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { //nolint:gosec // Ordinary user directories.
		return err
	}
	temporary := path + ".tmp"
	file, err := os.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644) //nolint:gosec // As other settings files.
	if err != nil {
		return err
	}
	if _, err := file.WriteString(text); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

// CreateDirectory creates path and its missing parents.
func CreateDirectory(path string) error {
	return os.MkdirAll(path, 0o755) //nolint:gosec // Ordinary user directories.
}

// CreateTemporaryDirectory creates a new, empty directory in parent whose name
// starts with prefix.
func CreateTemporaryDirectory(parent, prefix string) (string, error) {
	return os.MkdirTemp(parent, prefix+"*")
}

// CopyFile copies the file from to the new file to.
func CopyFile(from, to string) error {
	source, err := os.Open(from) //nolint:gosec // The user chose the file.
	if err != nil {
		return err
	}
	defer func() { _ = source.Close() }()
	target, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644) //nolint:gosec // A copy for vfox to read.
	if err != nil {
		return err
	}
	if _, err := io.Copy(target, source); err != nil {
		_ = target.Close()
		return err
	}
	return target.Close()
}

// Remove deletes path recursively; a missing path is not an error.
func Remove(path string) error {
	return os.RemoveAll(path)
}

// SystemTemporaryDirectory is the temporary directory of the system.
func SystemTemporaryDirectory() string {
	return os.TempDir()
}

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

package msixinstall

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// diagnose shows what Windows and the SDK say about the signed packages after
// sign-msix failed. The deployment log says that a package could not be opened, the
// packaging log says why; the signed package of this machine, installed alone, tells
// a fault of the package from one of the bundle, and signtool and makeappx check the
// package with the code of Windows. It only reports: none of its own failures changes
// the result of the check.
func diagnose(m Machine, out io.Writer, dir, arch string) {
	fmt.Fprintln(out, "--- AppX deployment log (tail) ---")
	showLog(out, m.DeploymentLog)
	fmt.Fprintln(out, "--- AppxPackaging log (tail) ---")
	showLog(out, m.PackagingLog)

	signed := filepath.Join(dir, "signed")
	fmt.Fprintln(out, "--- the signed package of this machine, alone ---")
	if entries, err := os.ReadDir(signed); err != nil {
		fmt.Fprintf(out, "(cannot list %s: %v)\n", signed, err)
	} else {
		for _, entry := range entries {
			if info, err := entry.Info(); err == nil {
				fmt.Fprintf(out, "%s %d\n", entry.Name(), info.Size())
			}
		}
	}
	found, err := matches(signed, "*_"+arch+".msix")
	if err != nil || len(found) == 0 {
		fmt.Fprintf(out, "no signed package for %s in %s\n", arch, signed)
		return
	}
	pkg := found[0]
	name := filepath.Base(pkg)
	if err := m.Install(pkg, false); err != nil {
		fmt.Fprintf(out, "%s does not install: %v\n", name, err)
	} else {
		fmt.Fprintf(out, "%s installs\n", name)
		if pkgs, err := m.Installed(); err == nil {
			for _, p := range pkgs {
				_ = m.Remove(p.FullName) // The diagnosis does not fail on a package it cannot remove.
			}
		}
	}

	fmt.Fprintf(out, "--- the last 98 bytes of %s ---\n", name)
	if text, err := lastBytes(pkg, 98); err != nil {
		fmt.Fprintf(out, "(cannot read %s: %v)\n", name, err)
	} else {
		fmt.Fprintln(out, text)
	}

	for _, tool := range []string{"signtool.exe", "makeappx.exe"} {
		path, err := m.SDKTool(tool)
		if err != nil || path == "" {
			fmt.Fprintf(out, "--- %s is not on this runner ---\n", tool)
			continue
		}
		fmt.Fprintf(out, "--- %s on %s ---\n", path, name)
		var code int
		if tool == "signtool.exe" {
			code, err = m.Run(path, "verify", "/pa", "/v", pkg)
		} else {
			code, err = m.Run(path, "unpack", "/p", pkg, "/d", filepath.Join(runnerTemp(), "unpacked"), "/o", "/v")
		}
		if err != nil {
			fmt.Fprintf(out, "%v\n", err)
			continue
		}
		fmt.Fprintf(out, "exit code %d\n", code)
	}
}

// lastBytes returns the last n bytes of the file, as hexadecimal numbers separated by spaces.
func lastBytes(path string, n int) (string, error) {
	data, err := os.ReadFile(path) //nolint:gosec // The path is a package of the release directory.
	if err != nil {
		return "", err
	}
	if len(data) > n {
		data = data[len(data)-n:]
	}
	parts := make([]string, len(data))
	for i, b := range data {
		parts[i] = fmt.Sprintf("%02x", b)
	}
	return strings.Join(parts, " "), nil
}

// runnerTemp returns the temporary directory of the runner, or that of the system.
func runnerTemp() string {
	if dir := os.Getenv("RUNNER_TEMP"); dir != "" {
		return dir
	}
	return os.TempDir()
}

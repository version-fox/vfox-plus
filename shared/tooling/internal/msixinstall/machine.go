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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// The PowerShell scripts of the machine. Each one reads its argument from
// $env:VFXTOOL_ARG, so that no path or thumbprint is ever quoted into a script. The
// lists are printed as JSON, with every field as a string.
const (
	scriptInstall         = `$ErrorActionPreference = 'Stop'; Add-AppxPackage -Path $env:VFXTOOL_ARG`
	scriptInstallUnsigned = `$ErrorActionPreference = 'Stop'; Add-AppxPackage -Path $env:VFXTOOL_ARG -AllowUnsigned`
	scriptRemove          = `$ErrorActionPreference = 'Stop'; Remove-AppxPackage -Package $env:VFXTOOL_ARG`
	scriptRemoveCert      = `$ErrorActionPreference = 'Stop'; Remove-Item -Path ('Cert:\LocalMachine\TrustedPeople\' + $env:VFXTOOL_ARG)`
	scriptDeploymentLog   = `Get-AppPackageLog | Select-Object -Last 60 | Out-String -Width 200`
	scriptPackagingLog    = `Get-WinEvent -LogName 'Microsoft-Windows-AppxPackaging/Operational' -MaxEvents 20 -ErrorAction SilentlyContinue | Format-List TimeCreated, Id, Message | Out-String -Width 200`
	scriptInstalled       = `$list = @(Get-AppxPackage -Name '` + packageName + `' | ForEach-Object { [pscustomobject]@{ PackageFullName = [string]$_.PackageFullName; Architecture = [string]$_.Architecture; SignatureKind = [string]$_.SignatureKind } }); if ($list.Count -eq 0) { '[]' } else { ConvertTo-Json -Compress -InputObject $list }`
	scriptCertificates    = `$list = @(Get-ChildItem Cert:\LocalMachine\TrustedPeople | Where-Object { $_.Subject -eq '` + certSubject + `' } | ForEach-Object { [pscustomobject]@{ Thumbprint = [string]$_.Thumbprint; Subject = [string]$_.Subject } }); if ($list.Count -eq 0) { '[]' } else { ConvertTo-Json -Compress -InputObject $list }`
)

// windowsMachine is the Windows of this runner: its Appx cmdlets and its certificate
// store, reached through Windows PowerShell, and the programs and the SDK of the machine.
type windowsMachine struct{}

func (windowsMachine) Install(path string, allowUnsigned bool) error {
	script := scriptInstall
	if allowUnsigned {
		script = scriptInstallUnsigned
	}
	_, err := powershell(script, path)
	return err
}

func (windowsMachine) Installed() ([]Package, error) {
	out, err := powershell(scriptInstalled, "")
	if err != nil {
		return nil, err
	}
	var pkgs []Package
	err = parseList(out, &pkgs)
	return pkgs, err
}

func (windowsMachine) Remove(fullName string) error {
	_, err := powershell(scriptRemove, fullName)
	return err
}

func (windowsMachine) Certificates() ([]Certificate, error) {
	out, err := powershell(scriptCertificates, "")
	if err != nil {
		return nil, err
	}
	var certs []Certificate
	err = parseList(out, &certs)
	return certs, err
}

func (windowsMachine) RemoveCertificate(thumbprint string) error {
	_, err := powershell(scriptRemoveCert, thumbprint)
	return err
}

func (windowsMachine) DeploymentLog() (string, error) {
	return powershell(scriptDeploymentLog, "")
}

func (windowsMachine) PackagingLog() (string, error) {
	return powershell(scriptPackagingLog, "")
}

// SDKTool returns the last tool of the name that the Windows SDK has for x64 or arm64,
// in the layout of the SDK (bin\<version>\<arch>\<tool>), or "" when there is none.
func (windowsMachine) SDKTool(name string) (string, error) {
	kits := filepath.Join(os.Getenv("ProgramFiles(x86)"), "Windows Kits", "10", "bin")
	var found []string
	_ = filepath.WalkDir(kits, func(path string, entry fs.DirEntry, err error) error {
		// A folder that cannot be read is skipped, as the SDK search of the checks did.
		if err != nil || entry.IsDir() || !strings.EqualFold(entry.Name(), name) {
			return nil
		}
		lower := strings.ToLower(path)
		if strings.Contains(lower, `\x64\`) || strings.Contains(lower, `\arm64\`) {
			found = append(found, path)
		}
		return nil
	})
	if len(found) == 0 {
		return "", nil
	}
	sort.Strings(found)
	return found[len(found)-1], nil
}

func (windowsMachine) Run(program string, args ...string) (int, error) {
	cmd := exec.Command(program, args...) //nolint:gosec // The caller names the program: this command or an SDK tool.
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), nil
	}
	return 0, err
}

// powershell runs a script in Windows PowerShell, with arg in $env:VFXTOOL_ARG, and
// returns its standard output. The installs and removals of Appx are cmdlets of
// Windows PowerShell 5.1: PowerShell 7 reaches them only through a compatibility
// session.
func powershell(script, arg string) (string, error) {
	path, err := powershellPath()
	if err != nil {
		return "", err
	}
	cmd := exec.Command(path, "-NoLogo", "-NoProfile", "-NonInteractive", "-Command", script) //nolint:gosec // The script is a constant of this package.
	cmd.Env = append(os.Environ(), "VFXTOOL_ARG="+arg)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

// powershellPath returns Windows PowerShell 5.1, which is in the system directory.
func powershellPath() (string, error) {
	path := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	return exec.LookPath("powershell.exe")
}

// parseList reads the JSON list that a script printed. Windows PowerShell can print
// messages before it, so the list is the last line that starts with [.
func parseList(out string, into any) error {
	list := ""
	for _, line := range strings.Split(out, "\n") {
		if text := strings.TrimSpace(line); strings.HasPrefix(text, "[") {
			list = text
		}
	}
	if list == "" {
		return fmt.Errorf("PowerShell printed no list: %q", out)
	}
	return json.Unmarshal([]byte(list), into)
}

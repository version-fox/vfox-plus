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

//go:build windows

package signmsix

import (
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// windowsSystem asks Windows itself: the token of the process, ShellExecute
// for the elevation, the store of the machine, and PowerShell for the
// installation (Add-AppxPackage is a cmdlet over the deployment API of
// Windows, which has no counterpart in the Go library).
type windowsSystem struct{}

func newSystem() system { return windowsSystem{} }

func (windowsSystem) elevated() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}

// seeMaskNoCloseProcess asks ShellExecuteEx for a handle of the process that it
// starts, to wait for it.
const seeMaskNoCloseProcess = 0x00000040

// shellExecuteInfo is SHELLEXECUTEINFOW.
type shellExecuteInfo struct {
	Size       uint32
	Mask       uint32
	Window     windows.Handle
	Verb       *uint16
	File       *uint16
	Parameters *uint16
	Directory  *uint16
	Show       int32
	InstApp    windows.Handle
	IDList     uintptr
	Class      *uint16
	ClassKey   windows.Handle
	HotKey     uint32
	Monitor    windows.Handle // The union with the icon.
	Process    windows.Handle
}

var procShellExecuteEx = windows.NewLazySystemDLL("shell32.dll").NewProc("ShellExecuteExW")

// relaunchElevated starts this program again with the verb runas, which asks
// the user (UAC) to allow it, in a console window of its own, and waits for it.
func (windowsSystem) relaunchElevated(args []string) (int, error) {
	executable, err := os.Executable()
	if err != nil {
		return 0, err
	}
	directory, err := os.Getwd()
	if err != nil {
		return 0, err
	}
	parameters := []string{"sign-msix"}
	for _, arg := range args {
		parameters = append(parameters, windows.EscapeArg(arg))
	}
	verb, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return 0, err
	}
	file, err := windows.UTF16PtrFromString(executable)
	if err != nil {
		return 0, err
	}
	commandLine, err := windows.UTF16PtrFromString(strings.Join(parameters, " "))
	if err != nil {
		return 0, err
	}
	workDir, err := windows.UTF16PtrFromString(directory)
	if err != nil {
		return 0, err
	}
	info := shellExecuteInfo{
		Mask:       seeMaskNoCloseProcess,
		Verb:       verb,
		File:       file,
		Parameters: commandLine,
		Directory:  workDir,
		Show:       windows.SW_SHOWNORMAL,
	}
	info.Size = uint32(unsafe.Sizeof(info))
	if result, _, callErr := procShellExecuteEx.Call(uintptr(unsafe.Pointer(&info))); result == 0 {
		return 0, fmt.Errorf("cannot start the elevated copy: %w", callErr)
	}
	if info.Process == 0 {
		return 0, errors.New("the elevated copy has no process to wait for")
	}
	defer func() { _ = windows.CloseHandle(info.Process) }()
	if _, err := windows.WaitForSingleObject(info.Process, windows.INFINITE); err != nil {
		return 0, err
	}
	var code uint32
	if err := windows.GetExitCodeProcess(info.Process, &code); err != nil {
		return 0, err
	}
	return int(code), nil
}

// addToStore puts the certificate into the TrustedPeople store of the local
// machine, which is what Windows looks at to sideload a signed package, and
// takes out of it the certificates that replaces recognises. Failing to take
// one out is told but does not fail the command.
func (windowsSystem) addToStore(der []byte, replaces func(*x509.Certificate) bool) error {
	if len(der) == 0 {
		return errors.New("an empty certificate")
	}
	name, err := windows.UTF16PtrFromString("TrustedPeople")
	if err != nil {
		return err
	}
	store, err := windows.CertOpenStore(windows.CERT_STORE_PROV_SYSTEM, 0, 0, windows.CERT_SYSTEM_STORE_LOCAL_MACHINE, uintptr(unsafe.Pointer(name)))
	runtime.KeepAlive(name)
	if err != nil {
		return fmt.Errorf("cannot open the TrustedPeople store of this machine: %w", err)
	}
	defer func() { _ = windows.CertCloseStore(store, 0) }()

	// Enumerate first and delete after: the enumeration frees the context that
	// it was given, so the ones to delete are duplicated.
	var earlier []*windows.CertContext
	var previous *windows.CertContext
	for {
		current, err := windows.CertEnumCertificatesInStore(store, previous)
		if err != nil || current == nil {
			break
		}
		encoded := append([]byte(nil), unsafe.Slice(current.EncodedCert, current.Length)...)
		if parsed, err := x509.ParseCertificate(encoded); err == nil && replaces(parsed) {
			earlier = append(earlier, windows.CertDuplicateCertificateContext(current))
		}
		previous = current
	}
	for _, found := range earlier {
		// The call frees the context, whether it works or not.
		if err := windows.CertDeleteCertificateFromStore(found); err != nil {
			fmt.Printf("Could not remove a certificate of an earlier run: %v\n", err)
		}
	}

	added, err := windows.CertCreateCertificateContext(windows.X509_ASN_ENCODING, &der[0], uint32(len(der)))
	if err != nil {
		return fmt.Errorf("cannot read the certificate: %w", err)
	}
	defer func() { _ = windows.CertFreeCertificateContext(added) }()
	if err := windows.CertAddCertificateContextToStore(store, added, windows.CERT_STORE_ADD_REPLACE_EXISTING, nil); err != nil {
		return fmt.Errorf("cannot add the certificate to the TrustedPeople store: %w", err)
	}
	return nil
}

// install runs Add-AppxPackage in Windows PowerShell, which has the Appx module
// (PowerShell 7 reaches it only through a compatibility session). The path goes
// in the environment, so that no quoting can go wrong in it.
func (windowsSystem) install(bundle string) error {
	powershell := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	if _, err := os.Stat(powershell); err != nil {
		var lookErr error
		if powershell, lookErr = exec.LookPath("powershell.exe"); lookErr != nil {
			return errors.New("Windows PowerShell (powershell.exe) was not found")
		}
	}
	command := exec.Command(powershell, "-NoLogo", "-NoProfile", "-NonInteractive", "-Command",
		"$ErrorActionPreference = 'Stop'; Add-AppxPackage -Path $env:VFXTOOL_BUNDLE") //nolint:gosec // The command is a constant.
	command.Env = append(os.Environ(), "VFXTOOL_BUNDLE="+bundle)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("Add-AppxPackage failed: %w", err)
	}
	return nil
}

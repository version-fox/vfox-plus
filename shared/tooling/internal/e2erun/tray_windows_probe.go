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

package e2erun

import (
	"fmt"
	"runtime"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// The scenario asks the desktop where the icon of the application is, with Shell_NotifyIconGetRect (the
// application does, and says what the answer was), and a desktop does not always know: Windows 11 puts a
// new icon in the overflow of the taskbar, a machine may have no notification area that tells, and a
// shell that has not finished starting takes no icon at all. An application that is told that its icon has
// no place has to be judged against what the same desktop says of the icon of a program that does what every
// program does, which is what the probe makes: it adds an icon of its own, and asks where it is.
//
// It uses the calls and the structures that the application does (mygo, internal/windows/tray.go), and
// nothing of mygo.

// The messages and the flags of the notification area.
const (
	probeNIMAdd    = 0
	probeNIMDelete = 2

	probeNIFMessage = 0x01
	probeNIFIcon    = 0x02
	probeNIFTip     = 0x04

	probeCallbackMessage = 0x8000 + 1 // WM_APP + 1
	probeIconApplication = 32512      // IDI_APPLICATION, a MAKEINTRESOURCE
)

// probeNotifyIconData is NOTIFYICONDATAW.
type probeNotifyIconData struct {
	Size            uint32
	Wnd             uintptr
	ID              uint32
	Flags           uint32
	CallbackMessage uint32
	Icon            uintptr
	Tip             [128]uint16
	State           uint32
	StateMask       uint32
	Info            [256]uint16
	Version         uint32
	InfoTitle       [64]uint16
	InfoFlags       uint32
	GUIDItem        windows.GUID
	BalloonIcon     uintptr
}

// probeNotifyIconIdentifier is NOTIFYICONIDENTIFIER.
type probeNotifyIconIdentifier struct {
	Size uint32
	Wnd  uintptr
	ID   uint32
	GUID windows.GUID
}

// probeWindowClass is WNDCLASSEXW.
type probeWindowClass struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   windows.Handle
	Icon       windows.Handle
	Cursor     windows.Handle
	Background windows.Handle
	MenuName   *uint16
	ClassName  *uint16
	IconSm     windows.Handle
}

// probeResult is what the desktop did with the icon of the probe.
type probeResult struct {
	// Added is whether Shell_NotifyIcon took the icon.
	Added bool
	// Placed is whether the desktop then told where the icon is.
	Placed bool
	// Report says what happened, for the log.
	Report string
}

// probePlainIcon adds an icon to the notification area from this process, as any program does, asks the
// desktop for a few seconds where it is, and takes it away. A failure of the probe is its report, and
// no failure of the scenario: it only says what the machine does.
func probePlainIcon() (result probeResult) {
	defer func() {
		if recovered := recover(); recovered != nil {
			result = probeResult{Report: fmt.Sprintf("the probe failed: %v", recovered)}
		}
	}()
	// The window belongs to the thread that makes it.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	user32 := windows.NewLazySystemDLL("user32.dll")
	shell32 := windows.NewLazySystemDLL("shell32.dll")
	// The module of the program, which owns the window class.
	var instance windows.Handle
	if err := windows.GetModuleHandleEx(windows.GET_MODULE_HANDLE_EX_FLAG_UNCHANGED_REFCOUNT, nil, &instance); err != nil {
		return probeResult{Report: "the probe has no module: " + err.Error()}
	}
	className, err := windows.UTF16PtrFromString("VfoxPlusTrayProbe")
	if err != nil {
		return probeResult{Report: err.Error()}
	}
	defWindowProc := user32.NewProc("DefWindowProcW")
	class := probeWindowClass{
		Instance:  instance,
		ClassName: className,
		WndProc: windows.NewCallback(func(window, message, wParam, lParam uintptr) uintptr {
			answer, _, _ := defWindowProc.Call(window, message, wParam, lParam)
			return answer
		}),
	}
	class.Size = uint32(unsafe.Sizeof(class))
	if atom, _, callErr := user32.NewProc("RegisterClassExW").Call(uintptr(unsafe.Pointer(&class))); atom == 0 {
		return probeResult{Report: "the probe could not register its window class: " + callErr.Error()}
	}
	defer func() {
		_, _, _ = user32.NewProc("UnregisterClassW").Call(uintptr(unsafe.Pointer(className)), uintptr(instance))
	}()
	window, _, callErr := user32.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(className)),
		0, 0, 0, 0, 0, 0, 0, uintptr(instance), 0)
	if window == 0 {
		return probeResult{Report: "the probe could not make its window: " + callErr.Error()}
	}
	defer func() { _, _, _ = user32.NewProc("DestroyWindow").Call(window) }()

	icon, _, _ := user32.NewProc("LoadIconW").Call(0, probeIconApplication)
	data := probeNotifyIconData{Wnd: window, ID: 1, Flags: probeNIFMessage | probeNIFIcon | probeNIFTip, CallbackMessage: probeCallbackMessage, Icon: icon}
	data.Size = uint32(unsafe.Sizeof(data))
	tip, _ := windows.UTF16FromString("vfox-plus probe")
	copy(data.Tip[:], tip)

	notifyIcon := shell32.NewProc("Shell_NotifyIconW")
	added, _, _ := notifyIcon.Call(probeNIMAdd, uintptr(unsafe.Pointer(&data)))
	if added == 0 {
		return probeResult{Report: "Shell_NotifyIcon did not take a plain icon"}
	}
	defer func() { _, _, _ = notifyIcon.Call(probeNIMDelete, uintptr(unsafe.Pointer(&data))) }()

	identifier := probeNotifyIconIdentifier{Wnd: window, ID: 1}
	identifier.Size = uint32(unsafe.Sizeof(identifier))
	getRect := shell32.NewProc("Shell_NotifyIconGetRect")
	var (
		place  windows.Rect
		answer uintptr
	)
	for deadline := time.Now().Add(5 * time.Second); ; time.Sleep(200 * time.Millisecond) {
		answer, _, _ = getRect.Call(uintptr(unsafe.Pointer(&identifier)), uintptr(unsafe.Pointer(&place)))
		if answer == 0 && place.Right > place.Left && place.Bottom > place.Top {
			return probeResult{Added: true, Placed: true, Report: fmt.Sprintf(
				"Shell_NotifyIcon took a plain icon and Shell_NotifyIconGetRect says where it is: %d,%d to %d,%d",
				place.Left, place.Top, place.Right, place.Bottom)}
		}
		if time.Now().After(deadline) {
			return probeResult{Added: true, Report: fmt.Sprintf(
				"Shell_NotifyIcon took a plain icon, and Shell_NotifyIconGetRect did not say where it is in 5 s (it answered 0x%08x, the rectangle %d,%d to %d,%d)",
				uint32(answer), place.Left, place.Top, place.Right, place.Bottom)}
		}
	}
}

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

//go:build linux

package ui

import (
	"log"
	"os"

	"github.com/ebitengine/purego"
)

// prepareProcess does what the application needs before mygo starts GTK.
//
// GTK names the window after the program: the application id on Wayland and the
// window class on X11, which the desktop entry (dev.vfox.VfoxPlus.desktop)
// refers to. GTK takes the name from g_get_prgname, which a Go program does not
// set, so it is set here, through the GLib that GTK loads anyway.
//
// mygo draws on the GPU where OpenGL runs on one, which loads Mesa and the
// compiler of its software driver, some 50 MB. The pages of vfox+ are drawn on
// the CPU in a fraction of a millisecond, so the application keeps to the CPU
// and tells GDK, which opens an OpenGL connection of its own, to leave OpenGL
// alone, unless the user sets MYGO_GPU, which says otherwise (MYGO_GPU=1 draws
// with OpenGL).
func prepareProcess() {
	if _, set := os.LookupEnv("MYGO_GPU"); !set {
		defaults := map[string]string{"MYGO_GPU": "0"}
		if _, set := os.LookupEnv("GDK_GL"); !set {
			defaults["GDK_GL"] = "disable"
		}
		for name, value := range defaults {
			if err := os.Setenv(name, value); err != nil {
				log.Printf("cannot choose the renderer: %v", err)
			}
		}
	}
	glib, err := purego.Dlopen("libglib-2.0.so.0", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		log.Printf("cannot name the window: %v", err)
		return
	}
	var setProgramName func(name string)
	purego.RegisterLibFunc(&setProgramName, glib, "g_set_prgname")
	setProgramName(ApplicationID)
}

// The host of the tray icons of a desktop owns this name on the session bus, with this
// object: the panels of KDE Plasma, of GNOME with its AppIndicator extension, of Xfce, of
// Cinnamon and of the others that show them.
const (
	trayWatcherName      = "org.kde.StatusNotifierWatcher"
	trayWatcherPath      = "/StatusNotifierWatcher"
	trayWatcherInterface = "org.kde.StatusNotifierWatcher"
)

// What a proxy for the host is asked to leave alone: its properties and its signals, which
// nothing here reads, and above all the starting of a host that is not running.
const (
	gBusTypeSession     = 2
	gDBusProxyNoProps   = 1 << 0
	gDBusProxyNoSignals = 1 << 1
	gDBusProxyNoStart   = 1 << 2
)

// trayHostAvailable tells whether the desktop has a host for tray icons, which is what a tray
// icon needs on Linux. libayatana-appindicator3 looks for the same name, and when it is not
// there it falls back to an icon of the old kind (GtkStatusIcon) that no panel of today
// shows, and loads the pictures of the desktop to make it: some 7 MB of memory for an icon
// that nobody sees. So the application makes no icon where there is no host.
func trayHostAvailable() bool {
	open := func(name string) uintptr {
		library, err := purego.Dlopen(name, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			return 0
		}
		return library
	}
	gio, gobject, glib := open("libgio-2.0.so.0"), open("libgobject-2.0.so.0"), open("libglib-2.0.so.0")
	if gio == 0 || gobject == 0 || glib == 0 {
		return false
	}
	var (
		proxyForBus func(busType, flags int32, info uintptr, name, path, iface string, cancellable uintptr, failure *uintptr) uintptr
		nameOwner   func(proxy uintptr) uintptr
		unref       func(object uintptr)
		free        func(memory uintptr)
		freeError   func(failure uintptr)
	)
	purego.RegisterLibFunc(&proxyForBus, gio, "g_dbus_proxy_new_for_bus_sync")
	purego.RegisterLibFunc(&nameOwner, gio, "g_dbus_proxy_get_name_owner")
	purego.RegisterLibFunc(&unref, gobject, "g_object_unref")
	purego.RegisterLibFunc(&free, glib, "g_free")
	purego.RegisterLibFunc(&freeError, glib, "g_error_free")

	failure := new(uintptr)
	proxy := proxyForBus(gBusTypeSession, gDBusProxyNoProps|gDBusProxyNoSignals|gDBusProxyNoStart, 0,
		trayWatcherName, trayWatcherPath, trayWatcherInterface, 0, failure)
	if proxy == 0 {
		// No session bus, or no way to reach it.
		if *failure != 0 {
			freeError(*failure)
		}
		return false
	}
	defer unref(proxy)
	owner := nameOwner(proxy)
	if owner == 0 {
		return false
	}
	free(owner)
	return true
}

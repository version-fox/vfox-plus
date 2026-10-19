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

// ownFontCache points the cache directory of the process at a new directory
// below the temporary files, and reports whether it did. mygo gives the fonts that
// an application registers to fontconfig as files in the cache directory of the
// user, which can be read-only, and where two programs that register the same
// fonts at the same moment race; a directory of the process's own has neither
// problem.
func ownFontCache() bool {
	dir, err := os.MkdirTemp("", "vfox-plus-cache-")
	if err != nil {
		log.Printf("cannot make a cache directory for the fonts: %v", err)
		return false
	}
	if err := os.Setenv("XDG_CACHE_HOME", dir); err != nil {
		log.Printf("cannot use the cache directory %s for the fonts: %v", dir, err)
		return false
	}
	return true
}

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

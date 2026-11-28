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

// Reading an installed plugin: where it is, the profile it ships, and what its
// Lua files say about it.

package profile

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// PluginProfile is where a plugin ships its profile: under lib, which vfox
// packs into the plugin, so that the profile arrives and updates with it.
const PluginProfile = "lib/vfox-plus.json"

// Capability is whether a plugin can do something, which its files tell.
type Capability int

// The capabilities.
const (
	// CapabilityUnknown: the files do not tell.
	CapabilityUnknown Capability = iota
	// CapabilityPresent: the plugin implements it.
	CapabilityPresent
	// CapabilityAbsent: the plugin does not implement it.
	CapabilityAbsent
)

// Offered reports whether an interface offers what the capability is for:
// everything but what the plugin lacks.
func (c Capability) Offered() bool { return c != CapabilityAbsent }

// PluginFiles is what the files of an installed plugin say.
type PluginFiles struct {
	// Profile is the profile that the plugin ships, nil when it ships none that
	// can be read.
	Profile *Document
	// Problems are what the profile says that cannot be read.
	Problems []Problem
	// LegacyFiles are PLUGIN.legacyFilenames of metadata.lua.
	LegacyFiles []string
	// EnvKeys are the settings of the plugin that its Lua files read from the
	// environment (see ScanEnvKeys).
	EnvKeys []string
	// Detect is whether the plugin finds installs that are already on the
	// machine, for `vfox link --detect`.
	Detect Capability
}

// The most of a plugin that ReadPlugin reads: more Lua than any plugin has.
const (
	maxLuaFiles = 500
	maxLuaFile  = 1 << 20
)

var detectHook = regexp.MustCompile(`PLUGIN\s*[:.]\s*Detect\b`)

// ReadPlugin reads the files of an installed plugin, from the root of its
// directory. Files that are missing leave their part empty.
func ReadPlugin(files fs.FS) PluginFiles {
	var result PluginFiles
	if content, err := readFSFile(files, "metadata.lua", maxLuaFile); err == nil {
		result.LegacyFiles = ParseLegacyFilenames(string(content))
	}
	sources := make(map[string]string)
	_ = fs.WalkDir(files, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(strings.ToLower(name), ".lua") {
			return nil
		}
		if len(sources) >= maxLuaFiles {
			return fs.SkipAll
		}
		if content, err := readFSFile(files, name, maxLuaFile); err == nil {
			sources[name] = string(content)
		}
		return nil
	})
	result.EnvKeys = ScanEnvKeys(sources)
	result.Detect = detectCapability(files, sources)
	if content, err := readFSFile(files, PluginProfile, maxFile); err == nil {
		d, problems, err := ParseDocument(content)
		if err != nil {
			problems = append(problems, Problem{Message: err.Error()})
		} else {
			// A plugin describes itself: what it says it is does not matter.
			d.Plugin = ""
			result.Profile = &d
		}
		result.Problems = problems
	} else if !errors.Is(err, fs.ErrNotExist) {
		result.Problems = []Problem{{Message: err.Error()}}
	}
	return result
}

// detectCapability tells whether the plugin has the Detect hook: a file of its
// own among the hooks, or, in a plugin of one file, a function of main.lua.
func detectCapability(files fs.FS, sources map[string]string) Capability {
	if _, err := fs.Stat(files, "hooks"); err == nil {
		if _, err := fs.Stat(files, path.Join("hooks", "detect.lua")); err == nil {
			return CapabilityPresent
		}
		return CapabilityAbsent
	}
	if main, ok := sources["main.lua"]; ok {
		if detectHook.MatchString(main) {
			return CapabilityPresent
		}
		return CapabilityAbsent
	}
	return CapabilityUnknown
}

var errTooLarge = errors.New("the file is too large")

// readLimited reads at most limit bytes, and fails for more.
func readLimited(reader io.Reader, limit int64) ([]byte, error) {
	content, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err == nil && int64(len(content)) > limit {
		return nil, errTooLarge
	}
	return content, err
}

func readFSFile(files fs.FS, name string, limit int64) ([]byte, error) {
	file, err := files.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	return readLimited(file, limit)
}

// readFile reads a profile file of the machine.
func readFile(name string) ([]byte, error) {
	file, err := os.Open(name) //nolint:gosec // The profiles of the user and of the cache of the application.
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	return readLimited(file, maxFile)
}

// PluginHome returns the directory that holds the plugins of vfox: VFOX_HOME
// when it is set, the legacy ~/.version-fox when it exists, ~/.vfox
// otherwise (under %USERPROFILE% on Windows).
func PluginHome(environment map[string]string) string {
	if home := strings.TrimSpace(environment["VFOX_HOME"]); home != "" {
		return home
	}
	home := pluginUserHome(environment)
	if home == "" {
		return ""
	}
	legacy := filepath.Join(home, ".version-fox")
	if info, err := os.Stat(legacy); err == nil && info.IsDir() {
		return legacy
	}
	return filepath.Join(home, ".vfox")
}

func pluginUserHome(environment map[string]string) string {
	if home := strings.TrimSpace(environment["HOME"]); home != "" {
		return home
	}
	return strings.TrimSpace(environment["USERPROFILE"])
}

// PluginDir returns the directory of the installed plugin name.
//
// The name comes from the user (an alias they typed, a segment of a URL), so
// it is checked before it joins a path: anything but a single directory name
// gives "", so that `..` or a separator can never escape the plugin directory
// (see the CodeQL finding on reading a plugin: the reading must stay inside
// the installed plugin). What vfox itself accepts as a name
// (`core/vfox`.IsValidPluginName) always passes here.
func PluginDir(environment map[string]string, plugin string) string {
	home := PluginHome(environment)
	if home == "" || !isPluginDirName(plugin) {
		return ""
	}
	root := filepath.Join(home, "plugin")
	directory := filepath.Join(root, plugin)
	// Belt and braces: the joined directory stays inside the plugin root.
	rel, err := filepath.Rel(root, directory)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ""
	}
	return directory
}

// isPluginDirName reports whether name is a single directory name: not empty
// and holding no separator and no ".." anywhere (so "a/../b" and "a..b" are
// both out), as the CodeQL help for go/path-injection recommends. Anything
// that vfox itself accepts as a name is either a single clean segment, which
// passes, or contains ".." or a separator, which does not need reading.
func isPluginDirName(name string) bool {
	if name == "" || name == "." {
		return false
	}
	return !strings.Contains(name, "/") && !strings.Contains(name, "\\") && !strings.Contains(name, "..")
}

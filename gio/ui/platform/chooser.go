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

package platform

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

// ErrUnavailable says that no file chooser is available: there is no desktop
// portal, and neither zenity nor kdialog is installed.
var ErrUnavailable = errors.New("no file chooser is available")

// ChooseOptions say what the user chooses.
type ChooseOptions struct {
	// Title is the title of the chooser.
	Title string
	// Initial is the directory the chooser starts in.
	Initial string
	// Directory chooses a directory instead of a file.
	Directory bool
	// FilterLabel and Extensions limit the files shown, such as "vfox
	// plugins" and "zip", "lua".
	FilterLabel string
	Extensions  []string
}

// Choose lets the user choose a file or a directory, with the chooser of the
// desktop portal, or else with zenity or kdialog. It blocks until the user
// answers, and returns an empty path if the user cancelled. It returns
// ErrUnavailable when there is no chooser at all.
func Choose(ctx context.Context, portal *Portal, o ChooseOptions) (string, error) {
	if portal != nil {
		if path, err := portal.choose(ctx, o); !errors.Is(err, ErrUnavailable) {
			return path, err
		}
	}
	return chooseWithProgram(ctx, o)
}

// choose uses the FileChooser interface of the desktop portal.
func (p *Portal) choose(ctx context.Context, o ChooseOptions) (string, error) {
	token := fmt.Sprintf("vfoxplus%d", time.Now().UnixNano())
	sender := strings.ReplaceAll(strings.TrimPrefix(p.conn.Names()[0], ":"), ".", "_")
	request := dbus.ObjectPath("/org/freedesktop/portal/desktop/request/" + sender + "/" + token)

	signals := make(chan *dbus.Signal, 8)
	p.conn.Signal(signals)
	defer p.conn.RemoveSignal(signals)
	match := []dbus.MatchOption{
		dbus.WithMatchObjectPath(request),
		dbus.WithMatchInterface("org.freedesktop.portal.Request"),
		dbus.WithMatchMember("Response"),
	}
	if err := p.conn.AddMatchSignal(match...); err != nil {
		return "", ErrUnavailable
	}
	defer func() { _ = p.conn.RemoveMatchSignal(match...) }()

	options := map[string]dbus.Variant{
		"handle_token": dbus.MakeVariant(token),
		"modal":        dbus.MakeVariant(true),
	}
	if o.Directory {
		options["directory"] = dbus.MakeVariant(true)
	}
	if o.Initial != "" {
		if info, err := os.Stat(o.Initial); err == nil && info.IsDir() {
			options["current_folder"] = dbus.MakeVariant(append([]byte(o.Initial), 0))
		}
	}
	if !o.Directory && len(o.Extensions) > 0 {
		type pattern struct {
			Kind    uint32
			Pattern string
		}
		type filter struct {
			Name     string
			Patterns []pattern
		}
		f := filter{Name: o.FilterLabel}
		for _, extension := range o.Extensions {
			f.Patterns = append(f.Patterns, pattern{0, "*." + extension})
		}
		options["filters"] = dbus.MakeVariant([]filter{f})
	}
	call := p.conn.Object(portalDestination, portalPath).CallWithContext(ctx,
		"org.freedesktop.portal.FileChooser.OpenFile", 0, "", o.Title, options)
	if call.Err != nil {
		return "", ErrUnavailable
	}
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case signal := <-signals:
			if signal.Path != request || len(signal.Body) < 2 {
				continue
			}
			code, _ := signal.Body[0].(uint32)
			results, _ := signal.Body[1].(map[string]dbus.Variant)
			if code != 0 {
				// The user cancelled, or the portal failed.
				return "", nil
			}
			var uris []string
			if v, ok := results["uris"]; ok {
				uris, _ = v.Value().([]string)
			}
			if len(uris) == 0 {
				return "", nil
			}
			return pathOfURI(uris[0]), nil
		}
	}
}

// pathOfURI is the path of a file URI.
func pathOfURI(uri string) string {
	parsed, err := url.Parse(uri)
	if err != nil || parsed.Scheme != "file" {
		return uri
	}
	return parsed.Path
}

// chooseWithProgram uses zenity or kdialog, which the desktop environments
// that lack a file chooser portal usually have.
func chooseWithProgram(ctx context.Context, o ChooseOptions) (string, error) {
	var cmd *exec.Cmd
	if zenity, err := exec.LookPath("zenity"); err == nil {
		args := []string{"--file-selection", "--title=" + o.Title}
		if o.Directory {
			args = append(args, "--directory")
		} else if len(o.Extensions) > 0 {
			patterns := make([]string, len(o.Extensions))
			for i, extension := range o.Extensions {
				patterns[i] = "*." + extension
			}
			args = append(args, "--file-filter="+o.FilterLabel+" | "+strings.Join(patterns, " "))
		}
		if o.Initial != "" {
			args = append(args, "--filename="+strings.TrimSuffix(o.Initial, "/")+"/")
		}
		cmd = exec.CommandContext(ctx, zenity, args...) //nolint:gosec // The program is zenity; there is no shell.
	} else if kdialog, err := exec.LookPath("kdialog"); err == nil {
		start := o.Initial
		if start == "" {
			start = "."
		}
		args := []string{"--title", o.Title}
		if o.Directory {
			args = append(args, "--getexistingdirectory", start)
		} else {
			args = append(args, "--getopenfilename", start)
			if len(o.Extensions) > 0 {
				patterns := make([]string, len(o.Extensions))
				for i, extension := range o.Extensions {
					patterns[i] = "*." + extension
				}
				args = append(args, strings.Join(patterns, " ")+"|"+o.FilterLabel)
			}
		}
		cmd = exec.CommandContext(ctx, kdialog, args...) //nolint:gosec // The program is kdialog; there is no shell.
	} else {
		return "", ErrUnavailable
	}
	out, err := cmd.Output()
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			// A cancelled chooser exits with a non-zero code.
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// OpenURL opens a web address in the default browser of the desktop.
func OpenURL(address string) {
	cmd := exec.Command("xdg-open", address) //nolint:gosec // The address goes to the browser; there is no shell.
	if err := cmd.Start(); err == nil {
		go func() { _ = cmd.Wait() }()
	}
}

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

// Package platform integrates the application with the Linux desktop: the
// appearance of the desktop from the XDG desktop portal, the file choosers
// of the portal, and links.
package platform

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	portalDestination = "org.freedesktop.portal.Desktop"
	portalPath        = "/org/freedesktop/portal/desktop"
	callTimeout       = 2 * time.Second
)

// Appearance is what the desktop says about how applications should look. A
// field is nil when the desktop does not say.
type Appearance struct {
	// Dark is whether the desktop prefers a dark appearance.
	Dark *bool
	// Accent is the accent colour of the desktop as ARGB.
	Accent *uint32
	// ReducedMotion is whether the user asked for less motion.
	ReducedMotion *bool
	// WindowButtons is how the desktop lays out the buttons of the title bar.
	WindowButtons *WindowButtons
}

// Portal talks to the XDG desktop portal on the session bus.
type Portal struct {
	conn *dbus.Conn
}

// Connect connects to the session bus. It returns an error when there is none,
// as in a container without a desktop.
func Connect() (*Portal, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, err
	}
	return &Portal{conn: conn}, nil
}

// Close releases the connection.
func (p *Portal) Close() {
	if p != nil && p.conn != nil {
		_ = p.conn.Close()
	}
}

// Read reads a setting of the portal.
func (p *Portal) read(namespace, key string) (dbus.Variant, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	var value dbus.Variant
	call := p.conn.Object(portalDestination, portalPath).CallWithContext(ctx,
		"org.freedesktop.portal.Settings.ReadOne", 0, namespace, key)
	if call.Err != nil || call.Store(&value) != nil {
		return dbus.Variant{}, false
	}
	return value, true
}

// Appearance reads the appearance settings of the desktop.
func (p *Portal) Appearance() Appearance {
	var a Appearance
	if v, ok := p.read("org.freedesktop.appearance", "color-scheme"); ok {
		if scheme, ok := v.Value().(uint32); ok {
			switch scheme {
			case 1:
				dark := true
				a.Dark = &dark
			case 2:
				dark := false
				a.Dark = &dark
			}
		}
	}
	if v, ok := p.read("org.freedesktop.appearance", "accent-color"); ok {
		a.Accent = ParseAccent(v.Value())
	}
	if v, ok := p.read("org.gnome.desktop.interface", "enable-animations"); ok {
		if enabled, ok := v.Value().(bool); ok {
			reduced := !enabled
			a.ReducedMotion = &reduced
		}
	}
	if v, ok := p.read("org.gnome.desktop.wm.preferences", "button-layout"); ok {
		if layout, ok := v.Value().(string); ok {
			buttons := ParseButtonLayout(layout)
			a.WindowButtons = &buttons
		}
	}
	return a
}

// WindowButtons are the buttons of a title bar and where they go: the names
// are "minimize", "maximize" and "close", from the end of the title bar that
// they are at, the left or the right.
type WindowButtons struct {
	Left, Right []string
}

// DefaultWindowButtons is the layout of Ubuntu, which most desktops that draw
// the title bar of an application follow.
func DefaultWindowButtons() WindowButtons {
	return WindowButtons{Right: []string{"minimize", "maximize", "close"}}
}

// ParseButtonLayout reads the layout of the buttons of the title bar that GNOME
// and GTK use, such as "appmenu:minimize,maximize,close": a colon divides the
// buttons at the left from those at the right. Of the names, only those of the
// window buttons are kept.
func ParseButtonLayout(layout string) WindowButtons {
	left, right, _ := strings.Cut(layout, ":")
	keep := func(list string) []string {
		var names []string
		for _, name := range strings.Split(list, ",") {
			switch name = strings.TrimSpace(name); name {
			case "minimize", "maximize", "close":
				names = append(names, name)
			}
		}
		return names
	}
	return WindowButtons{Left: keep(left), Right: keep(right)}
}

// ParseAccent converts the accent colour of the portal, a triple of
// components from 0 to 1, to ARGB. The portal reports components outside that
// range when no accent colour is set.
func ParseAccent(value any) *uint32 {
	components, ok := value.([]any)
	if !ok || len(components) != 3 {
		return nil
	}
	var argb uint32 = 0xFF000000
	for i, component := range components {
		f, ok := component.(float64)
		if !ok || f < 0 || f > 1 || math.IsNaN(f) {
			return nil
		}
		argb |= uint32(math.Round(f*255)) << (16 - 8*uint(i))
	}
	return &argb
}

// Watch calls onChange whenever the portal reports a changed setting, until
// the portal is closed.
func (p *Portal) Watch(onChange func()) {
	signals := make(chan *dbus.Signal, 16)
	p.conn.Signal(signals)
	err := p.conn.AddMatchSignal(
		dbus.WithMatchObjectPath(portalPath),
		dbus.WithMatchInterface("org.freedesktop.portal.Settings"),
		dbus.WithMatchMember("SettingChanged"),
	)
	if err != nil {
		return
	}
	go func() {
		for signal := range signals {
			if signal.Name == "org.freedesktop.portal.Settings.SettingChanged" {
				onChange()
			}
		}
	}()
}

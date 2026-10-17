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
	"context"
	"math"
	"time"

	"github.com/godbus/dbus/v5"
)

// AccentColor is the accent colour of the desktop as ARGB, from the XDG
// desktop portal (org.freedesktop.appearance accent-color), if the desktop
// sets one.
func AccentColor() (uint32, bool) {
	connection, err := dbus.ConnectSessionBus()
	if err != nil {
		return 0, false
	}
	defer func() { _ = connection.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	call := connection.Object("org.freedesktop.portal.Desktop", "/org/freedesktop/portal/desktop").
		CallWithContext(ctx, "org.freedesktop.portal.Settings.ReadOne", 0, "org.freedesktop.appearance", "accent-color")
	if call.Err != nil {
		return 0, false
	}
	var value dbus.Variant
	if err := call.Store(&value); err != nil {
		return 0, false
	}
	return argb(value)
}

// argb converts a (ddd) structure, possibly wrapped in variants, to ARGB. The
// portal reports components outside of 0–1 when no accent colour is set.
func argb(value any) (uint32, bool) {
	switch v := value.(type) {
	case dbus.Variant:
		return argb(v.Value())
	case []any:
		if len(v) != 3 {
			return 0, false
		}
		var channels [3]uint32
		for i, field := range v {
			component, ok := field.(float64)
			if !ok || component < 0 || component > 1 {
				return 0, false
			}
			channels[i] = uint32(math.Round(component * 255))
		}
		return 0xFF000000 | channels[0]<<16 | channels[1]<<8 | channels[2], true
	default:
		return 0, false
	}
}

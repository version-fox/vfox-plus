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

package trayhost

import (
	"bufio"
	"context"
	"errors"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/prop"
)

// privateBus starts a session bus of its own, which the connections of the test use, so that
// the tests need no desktop and cannot meet one.
func privateBus(t *testing.T) {
	t.Helper()
	daemon, err := exec.LookPath("dbus-daemon")
	if err != nil {
		t.Skip("no dbus-daemon to start a session bus with")
	}
	cmd := exec.Command(daemon, "--session", "--nofork", "--print-address=1") //nolint:gosec // The daemon of the test.
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	address, err := bufio.NewReader(out).ReadString('\n')
	if err != nil {
		t.Fatalf("the session bus gave no address: %v", err)
	}
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", strings.TrimSpace(address))
}

// layoutNode is a node of a dbusmenu layout as a server sends it.
type layoutNode struct {
	ID         int32
	Properties map[string]dbus.Variant
	Children   []dbus.Variant
}

// application is what makes an icon: it exports a StatusNotifierItem and its menu on the
// bus, as libayatana-appindicator3 does for the application, and registers the item.
type application struct {
	conn *dbus.Conn
	path dbus.ObjectPath

	mu     sync.Mutex
	events []string
}

const (
	itemPath = dbus.ObjectPath("/org/ayatana/NotificationItem/test")
	menuPath = dbus.ObjectPath("/org/ayatana/NotificationItem/test/Menu")
)

// newApplication makes the application. activates says whether the item takes Activate, which
// the libayatana-appindicator3 of Ubuntu 26.04 does not.
func newApplication(t *testing.T, activates bool) *application {
	t.Helper()
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	app := &application{conn: conn, path: itemPath}

	methods := map[string]any{}
	if activates {
		methods["Activate"] = func(x, y int32) *dbus.Error {
			app.note("activate")
			return nil
		}
	}
	if err := conn.ExportMethodTable(methods, itemPath, itemInterface); err != nil {
		t.Fatal(err)
	}
	if _, err := prop.Export(conn, itemPath, map[string]map[string]*prop.Prop{itemInterface: {
		"Id":            {Value: "test-1"},
		"Title":         {Value: "A test application"},
		"Status":        {Value: "Active", Emit: prop.EmitTrue},
		"Category":      {Value: "ApplicationStatus"},
		"IconName":      {Value: "icon-1"},
		"IconThemePath": {Value: "/tmp/test-icons"},
		"Menu":          {Value: menuPath},
		"ItemIsMenu":    {Value: false},
	}}); err != nil {
		t.Fatal(err)
	}

	entry := func(id int32, properties map[string]dbus.Variant) dbus.Variant {
		return dbus.MakeVariant(layoutNode{ID: id, Properties: properties, Children: []dbus.Variant{}})
	}
	menu := map[string]any{
		"AboutToShow": func(int32) (bool, *dbus.Error) { return false, nil },
		"GetLayout": func(int32, int32, []string) (uint32, layoutNode, *dbus.Error) {
			return 7, layoutNode{ID: 0, Properties: map[string]dbus.Variant{"children-display": dbus.MakeVariant("submenu")}, Children: []dbus.Variant{
				entry(3, map[string]dbus.Variant{"label": dbus.MakeVariant("Open"), "enabled": dbus.MakeVariant(true), "visible": dbus.MakeVariant(true)}),
				entry(4, map[string]dbus.Variant{"label": dbus.MakeVariant("Dimmed"), "enabled": dbus.MakeVariant(false)}),
				entry(5, map[string]dbus.Variant{"type": dbus.MakeVariant("separator")}),
				entry(6, map[string]dbus.Variant{"label": dbus.MakeVariant("Quit")}),
			}}, nil
		},
		"Event": func(id int32, event string, _ dbus.Variant, _ uint32) *dbus.Error {
			app.note(event + ":" + string(rune('0'+id)))
			return nil
		},
	}
	if err := conn.ExportMethodTable(menu, menuPath, menuInterface); err != nil {
		t.Fatal(err)
	}
	return app
}

func (a *application) note(event string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.events = append(a.events, event)
}

func (a *application) seen() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(a.events)
}

// register registers the item the way libayatana-appindicator3 does: by the object path.
func (a *application) register(t *testing.T, service string) {
	t.Helper()
	err := a.conn.Object(watcherName, watcherPath).Call(watcherInterface+".RegisterStatusNotifierItem", 0, service).Err
	if err != nil {
		t.Fatal(err)
	}
}

// within is a context for what should happen soon.
func within(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func startHost(t *testing.T) *Host {
	t.Helper()
	privateBus(t)
	host, err := Start()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = host.Close() })
	return host
}

func TestTheHostSeesTheIconThatAnApplicationRegisters(t *testing.T) {
	host := startHost(t)
	app := newApplication(t, false)
	app.register(t, string(itemPath))

	item, err := host.NextItem(within(t))
	if err != nil {
		t.Fatal(err)
	}
	if item.Service != app.conn.Names()[0] || item.Path != itemPath {
		t.Errorf("the item is %s%s, want %s%s", item.Service, item.Path, app.conn.Names()[0], itemPath)
	}
	properties, err := item.Properties()
	if err != nil {
		t.Fatal(err)
	}
	want := Properties{
		ID: "test-1", Title: "A test application", Status: "Active", Category: "ApplicationStatus",
		IconName: "icon-1", IconThemePath: "/tmp/test-icons", Menu: menuPath,
	}
	if properties != want {
		t.Errorf("properties = %+v, want %+v", properties, want)
	}
	if host.Registered() != 1 {
		t.Errorf("%d icons in the tray, want 1", host.Registered())
	}

	// The watcher says what it has registered, to the panels that ask.
	var registered []string
	if err := app.conn.Object(watcherName, watcherPath).Call("org.freedesktop.DBus.Properties.Get", 0, watcherInterface, "RegisteredStatusNotifierItems").Store(new(dbus.Variant)); err != nil {
		t.Fatal(err)
	}
	var variant dbus.Variant
	if err := app.conn.Object(watcherName, watcherPath).Call("org.freedesktop.DBus.Properties.Get", 0, watcherInterface, "RegisteredStatusNotifierItems").Store(&variant); err != nil {
		t.Fatal(err)
	}
	registered, _ = variant.Value().([]string)
	if len(registered) != 1 || !strings.HasSuffix(registered[0], string(itemPath)) {
		t.Errorf("registered items = %v", registered)
	}
	var hostRegistered dbus.Variant
	if err := app.conn.Object(watcherName, watcherPath).Call("org.freedesktop.DBus.Properties.Get", 0, watcherInterface, "IsStatusNotifierHostRegistered").Store(&hostRegistered); err != nil {
		t.Fatal(err)
	}
	if up, _ := hostRegistered.Value().(bool); !up {
		t.Error("a host is registered: libayatana-appindicator3 makes an icon of the old kind when there is none")
	}
}

func TestTheHostReadsTheMenuAndChoosesItsEntries(t *testing.T) {
	host := startHost(t)
	app := newApplication(t, false)
	app.register(t, string(itemPath))
	item, err := host.NextItem(within(t))
	if err != nil {
		t.Fatal(err)
	}

	entries, err := item.Entries()
	if err != nil {
		t.Fatal(err)
	}
	want := []Entry{
		{ID: 3, Label: "Open", Enabled: true, Visible: true},
		{ID: 4, Label: "Dimmed", Enabled: false, Visible: true},
		{ID: 5, Separator: true, Enabled: true, Visible: true},
		{ID: 6, Label: "Quit", Enabled: true, Visible: true},
	}
	if !slices.Equal(entries, want) {
		t.Fatalf("entries = %+v, want %+v", entries, want)
	}

	if err := item.Choose(0); err != nil {
		t.Fatal(err)
	}
	if err := item.Choose(3); err != nil {
		t.Fatal(err)
	}
	if got := app.seen(); !slices.Equal(got, []string{"clicked:3", "clicked:6"}) {
		t.Errorf("the application was sent %v", got)
	}
	// What a person cannot choose is not chosen.
	for _, index := range []int{1, 2, 4, -1} {
		if err := item.Choose(index); err == nil {
			t.Errorf("choosing the entry %d succeeded", index)
		}
	}
	if got := app.seen(); len(got) != 2 {
		t.Errorf("a refused choice reached the application: %v", got)
	}
}

func TestActivationIsCalledOrSaidToBeRefused(t *testing.T) {
	host := startHost(t)
	app := newApplication(t, true)
	app.register(t, string(itemPath))
	item, err := host.NextItem(within(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := item.Activate(); err != nil {
		t.Fatal(err)
	}
	if got := app.seen(); !slices.Equal(got, []string{"activate"}) {
		t.Errorf("the application was sent %v", got)
	}

	// An item that does not take the call, as the one of libayatana-appindicator3 0.5.94.
	host2 := &Host{conn: app.conn}
	plain := newApplicationAt(t, false, "/org/ayatana/NotificationItem/plain")
	refusing := &Item{h: host2, Service: plain.conn.Names()[0], Path: "/org/ayatana/NotificationItem/plain"}
	if err := refusing.Activate(); !errors.Is(err, ErrNoActivation) {
		t.Errorf("Activate on an item without it = %v, want %v", err, ErrNoActivation)
	}
}

// newApplicationAt is an application whose item is at another path.
func newApplicationAt(t *testing.T, activates bool, path dbus.ObjectPath) *application {
	t.Helper()
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	methods := map[string]any{}
	if activates {
		methods["Activate"] = func(int32, int32) *dbus.Error { return nil }
	}
	if err := conn.ExportMethodTable(methods, path, itemInterface); err != nil {
		t.Fatal(err)
	}
	return &application{conn: conn, path: path}
}

func TestTheHostSeesTheIconGoAndTheApplicationLeave(t *testing.T) {
	host := startHost(t)
	app := newApplication(t, false)
	app.register(t, string(itemPath))
	item, err := host.NextItem(within(t))
	if err != nil {
		t.Fatal(err)
	}

	// The application takes the icon away, as libayatana-appindicator3 does when it is destroyed.
	if err := app.conn.Emit(itemPath, itemInterface+".NewStatus", "Passive"); err != nil {
		t.Fatal(err)
	}
	if err := item.WaitPassive(within(t)); err != nil {
		t.Fatal(err)
	}
	if host.Registered() != 0 {
		t.Errorf("%d icons in the tray, want none", host.Registered())
	}
	short, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if err := item.WaitGone(short); err == nil {
		t.Error("the application is gone, but only its icon is")
	}

	// And then it leaves the bus.
	if err := app.conn.Close(); err != nil {
		t.Fatal(err)
	}
	if err := item.WaitGone(within(t)); err != nil {
		t.Fatal(err)
	}
}

func TestAnApplicationThatLeavesTheBusTakesItsIconAway(t *testing.T) {
	host := startHost(t)
	app := newApplication(t, false)
	app.register(t, string(itemPath))
	item, err := host.NextItem(within(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := app.conn.Close(); err != nil {
		t.Fatal(err)
	}
	if err := item.WaitPassive(within(t)); err != nil {
		t.Fatalf("an application that ended leaves no icon in the tray: %v", err)
	}
}

func TestOneHostOwnsTheNameOfTheWatcher(t *testing.T) {
	host := startHost(t)
	if _, err := Start(); err == nil {
		t.Error("a second host started beside the first")
	}
	if err := host.Close(); err != nil {
		t.Fatal(err)
	}
	if err := host.Close(); err != nil {
		t.Errorf("closing twice: %v", err)
	}
	// The name is free again.
	second, err := Start()
	if err != nil {
		t.Fatalf("the name of the watcher is not free after the host closed: %v", err)
	}
	_ = second.Close()
}

func TestAnItemIsRegisteredInTheFormsThatApplicationsUse(t *testing.T) {
	host := startHost(t)
	app := newApplication(t, false)
	name := app.conn.Names()[0]
	for _, service := range []string{string(itemPath), name + string(itemPath), name} {
		app.register(t, service)
		item, err := host.NextItem(within(t))
		if err != nil {
			t.Fatal(err)
		}
		wantPath := itemPath
		if service == name {
			wantPath = "/StatusNotifierItem" // The path that the specification gives an item registered by its name.
		}
		if item.Service != name || item.Path != wantPath {
			t.Errorf("registered as %q: the item is %s%s, want %s%s", service, item.Service, item.Path, name, wantPath)
		}
	}
	// Something that is no path is refused to the one that registers it.
	err := app.conn.Object(watcherName, watcherPath).Call(watcherInterface+".RegisterStatusNotifierItem", 0, name+"/bad path").Err
	if err == nil {
		t.Error("an item at a path that is not one was registered")
	}
}

func TestWaitingForAnIconThatNeverComesTimesOut(t *testing.T) {
	host := startHost(t)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := host.NextItem(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("NextItem = %v, want a timeout", err)
	}
}

func TestAnItemOfTheListOfTheWatcherIsTakenApart(t *testing.T) {
	cases := map[string]struct {
		service string
		path    dbus.ObjectPath
		ok      bool
	}{
		":1.11@/org/ayatana/NotificationItem/mygo_464_1": {":1.11", "/org/ayatana/NotificationItem/mygo_464_1", true}, // The extension of GNOME.
		":1.5/org/ayatana/NotificationItem/x":            {":1.5", "/org/ayatana/NotificationItem/x", true},
		":1.7":                                           {":1.7", "/StatusNotifierItem", true},
		"org.example.App":                                {"org.example.App", "/StatusNotifierItem", true},
		":1.9@/bad path":                                 {"", "", false},
		"":                                               {"", "", false},
		"@/org/x":                                        {"", "", false},
	}
	for entry, want := range cases {
		service, path, ok := splitItem(entry)
		if ok != want.ok || (ok && (service != want.service || path != want.path)) {
			t.Errorf("splitItem(%q) = %q %q %t, want %q %q %t", entry, service, path, ok, want.service, want.path, want.ok)
		}
	}
}

// A host of the desktop owns the name already, and the host that is attached to it sees the
// icons that are registered with it, and uses them, as the panel does.
func TestAHostThatIsAttachedSeesTheIconsOfTheHostOfTheDesktop(t *testing.T) {
	desktop := startHost(t)
	attached, err := Attach()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = attached.Close() })
	app := newApplication(t, false)
	app.register(t, string(itemPath))

	// Both see the item: the one that it was registered with, and the one that looks.
	if _, err := desktop.NextItem(within(t)); err != nil {
		t.Fatal(err)
	}
	item, err := attached.NextItem(within(t))
	if err != nil {
		t.Fatal(err)
	}
	if item.Service != app.conn.Names()[0] || item.Path != itemPath {
		t.Errorf("the item is %s%s, want %s%s", item.Service, item.Path, app.conn.Names()[0], itemPath)
	}
	entries, err := item.Entries()
	if err != nil || len(entries) != 4 || entries[0].Label != "Open" {
		t.Fatalf("entries = %+v, %v", entries, err)
	}
	if err := item.Choose(0); err != nil {
		t.Fatal(err)
	}
	if got := app.seen(); !slices.Equal(got, []string{"clicked:3"}) {
		t.Errorf("the application was sent %v", got)
	}
	if attached.Registered() != 1 {
		t.Errorf("%d icons in the tray, want 1", attached.Registered())
	}

	// The application takes the icon away, and the host that looks sees it go.
	if err := app.conn.Emit(itemPath, itemInterface+".NewStatus", "Passive"); err != nil {
		t.Fatal(err)
	}
	if err := item.WaitPassive(within(t)); err != nil {
		t.Fatal(err)
	}
	if attached.Registered() != 0 {
		t.Errorf("%d icons in the tray, want none", attached.Registered())
	}

	// Closing the host that looks leaves the name with the host of the desktop.
	if err := attached.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Attach(); err != nil {
		t.Errorf("the host of the desktop lost its name when a host that looked closed: %v", err)
	}
}

func TestAHostCannotBeAttachedToADesktopWithoutOne(t *testing.T) {
	privateBus(t)
	if _, err := Attach(); err == nil || !strings.Contains(err.Error(), "owns "+watcherName) {
		t.Errorf("Attach on a bus with no host = %v, want the lack of one", err)
	}
}

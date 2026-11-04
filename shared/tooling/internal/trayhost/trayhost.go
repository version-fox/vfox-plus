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

// Package trayhost is a stand-in for the host of tray icons of a Linux desktop: what the
// AppIndicator extension of GNOME is to the applications that ask it for an icon.
//
// An application that wants an icon in the tray makes a StatusNotifierItem on the session
// bus, with its menu as a dbusmenu, and registers it with the StatusNotifierWatcher, the
// service that owns org.kde.StatusNotifierWatcher. The panel is that service, and also the
// one that shows the icon: it reads the properties of the item (its title, its icon, where
// its menu is), reads the menu, and calls the item and the menu when the user clicks. A CI
// runner has no panel, and without a watcher libayatana-appindicator3 falls back to an icon of
// the old kind that nothing shows. Host owns the name on the session bus, as the panel does,
// and does what the panel does with an item, so that the end-to-end tests can see the
// icon that the application makes and use it, which is all that a person does with it.
//
// It speaks the protocols as the extension of GNOME does, and no more: there is no panel to
// draw. See https://freedesktop.org/wiki/Specifications/StatusNotifierItem/ and
// https://github.com/AyatanaIndicators/libdbusmenu/blob/master/libdbusmenu-glib/dbus-menu.xml.
package trayhost

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
)

// The names of the protocols.
const (
	watcherName      = "org.kde.StatusNotifierWatcher"
	watcherPath      = dbus.ObjectPath("/StatusNotifierWatcher")
	watcherInterface = "org.kde.StatusNotifierWatcher"
	itemInterface    = "org.kde.StatusNotifierItem"
	menuInterface    = "com.canonical.dbusmenu"
)

// ErrNoActivation is what an item answers to Activate when it does not take the call: the
// libayatana-appindicator3 of Ubuntu 26.04 has no activate signal, and then the panel gets
// UNKNOWN_METHOD and shows the menu instead.
var ErrNoActivation = errors.New("the item does not take Activate")

// Host is the host of the tray icons on the session bus: this one, or the one of the desktop that
// it was attached to.
type Host struct {
	conn    *dbus.Conn
	arrived chan *Item
	signals chan *dbus.Signal
	done    chan struct{}

	mu    sync.Mutex
	items []*Item
	props *prop.Properties
	// attached is true when the host is the desktop's, which this one only looks at.
	attached bool
}

// Start makes the host: it connects to the session bus and owns the name of the watcher on it.
// It fails when another host owns the name, which is the case on a desktop that has a tray: the
// tests are for the desktops that have none.
func Start() (*Host, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("connect to the session bus: %w", err)
	}
	h := &Host{
		conn:    conn,
		arrived: make(chan *Item, 16),
		signals: make(chan *dbus.Signal, 64),
		done:    make(chan struct{}),
	}
	if err := h.export(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	reply, err := conn.RequestName(watcherName, dbus.NameFlagDoNotQueue)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("own %s: %w", watcherName, err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		_ = conn.Close()
		return nil, fmt.Errorf("%s is owned by another host of tray icons", watcherName)
	}
	if err := h.listen(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	// The hosts of a desktop say that they are there, for the applications that wait for it.
	_ = conn.Emit(watcherPath, watcherInterface+".StatusNotifierHostRegistered")
	return h, nil
}

// Attach connects to the host of the tray icons that the desktop has, which owns the name of the
// watcher already: the AppIndicator extension of a GNOME Shell. It does not stand in for it and
// shows nothing: it sees the items that the applications register with it, as Host does the items
// that are registered with itself, and does with them what the panel does, so that a test can use
// the icon that the desktop shows. It fails when nothing owns the name.
func Attach() (*Host, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("connect to the session bus: %w", err)
	}
	h := &Host{
		conn:     conn,
		arrived:  make(chan *Item, 16),
		signals:  make(chan *dbus.Signal, 64),
		done:     make(chan struct{}),
		attached: true,
	}
	var owned bool
	if err := conn.BusObject().Call("org.freedesktop.DBus.NameHasOwner", 0, watcherName).Store(&owned); err != nil || !owned {
		_ = conn.Close()
		return nil, fmt.Errorf("nothing on the session bus owns %s: the desktop has no host of tray icons (%v)", watcherName, err)
	}
	if err := h.listen(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	go h.follow()
	return h, nil
}

// follow notes the items that the watcher of the desktop lists, as they come: the registrations
// are made to the desktop's host and not to this one, so what they are is read from the list that it
// keeps. The items are listed as the name of the bus of the application and the object of the item,
// joined by "@" (the extension of GNOME) or just joined.
func (h *Host) follow() {
	seen := map[string]bool{}
	for {
		select {
		case <-h.done:
			return
		case <-time.After(100 * time.Millisecond):
		}
		var listed dbus.Variant
		err := h.conn.Object(watcherName, watcherPath).Call("org.freedesktop.DBus.Properties.Get", 0, watcherInterface, "RegisteredStatusNotifierItems").Store(&listed)
		if err != nil {
			continue
		}
		items, _ := listed.Value().([]string)
		for _, entry := range items {
			if seen[entry] {
				continue
			}
			seen[entry] = true
			service, path, ok := splitItem(entry)
			if !ok {
				continue
			}
			item := &Item{h: h, Service: service, Path: path, passive: make(chan struct{}), gone: make(chan struct{})}
			h.mu.Lock()
			h.items = append(h.items, item)
			h.mu.Unlock()
			select {
			case h.arrived <- item:
			default:
			}
		}
	}
}

// splitItem takes an entry of RegisteredStatusNotifierItems apart: ":1.11@/org/ayatana/NotificationItem/x",
// ":1.11/org/ayatana/NotificationItem/x", or the name of a bus alone, which has the item of the specification.
func splitItem(entry string) (service string, path dbus.ObjectPath, ok bool) {
	switch {
	case strings.Contains(entry, "@"):
		name, rest, _ := strings.Cut(entry, "@")
		service, path = name, dbus.ObjectPath(rest)
	case strings.Contains(entry, "/"):
		name, rest, _ := strings.Cut(entry, "/")
		service, path = name, dbus.ObjectPath("/"+rest)
	default:
		service, path = entry, "/StatusNotifierItem"
	}
	return service, path, service != "" && path.IsValid()
}

// watcher is the exported object: the methods that the applications call.
type watcher struct{ h *Host }

// RegisterStatusNotifierItem is called by an application with the item that it made: the
// service is the object path of the item (libayatana-appindicator3), the name of the bus
// of the application (the spec's way, with /StatusNotifierItem as the path) or both.
func (w *watcher) RegisterStatusNotifierItem(sender dbus.Sender, service string) *dbus.Error {
	service, path := string(sender), dbus.ObjectPath(service)
	switch {
	case strings.HasPrefix(string(path), "/"):
	case strings.Contains(string(path), "/"):
		name, rest, _ := strings.Cut(string(path), "/")
		service, path = name, dbus.ObjectPath("/"+rest)
	default:
		service, path = string(path), "/StatusNotifierItem"
	}
	if !path.IsValid() {
		return dbus.MakeFailedError(fmt.Errorf("%q is not an object path", path))
	}
	item := &Item{h: w.h, Service: service, Path: path, passive: make(chan struct{}), gone: make(chan struct{})}
	w.h.mu.Lock()
	w.h.items = append(w.h.items, item)
	registered := make([]string, 0, len(w.h.items))
	for _, i := range w.h.items {
		registered = append(registered, i.Service+string(i.Path))
	}
	w.h.mu.Unlock()
	w.h.props.SetMust(watcherInterface, "RegisteredStatusNotifierItems", registered)
	_ = w.h.conn.Emit(watcherPath, watcherInterface+".StatusNotifierItemRegistered", item.Service+string(item.Path))
	select {
	case w.h.arrived <- item:
	default: // Nobody is waiting for the items, and the buffer is full: the oldest were never asked for.
	}
	return nil
}

// RegisterStatusNotifierHost is called by a panel that shows the items. The host is this one.
func (w *watcher) RegisterStatusNotifierHost(dbus.Sender, string) *dbus.Error { return nil }

// export puts the watcher on the bus with its properties and its description.
func (h *Host) export() error {
	w := &watcher{h: h}
	if err := h.conn.Export(w, watcherPath, watcherInterface); err != nil {
		return err
	}
	props, err := prop.Export(h.conn, watcherPath, map[string]map[string]*prop.Prop{
		watcherInterface: {
			"RegisteredStatusNotifierItems":  {Value: []string{}, Writable: false, Emit: prop.EmitTrue},
			"IsStatusNotifierHostRegistered": {Value: true, Writable: false, Emit: prop.EmitFalse},
			"ProtocolVersion":                {Value: int32(0), Writable: false, Emit: prop.EmitFalse},
		},
	})
	if err != nil {
		return err
	}
	h.props = props
	node := &introspect.Node{
		Name: string(watcherPath),
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			{
				Name:       watcherInterface,
				Methods:    introspect.Methods(w),
				Properties: props.Introspection(watcherInterface),
				Signals: []introspect.Signal{
					{Name: "StatusNotifierItemRegistered", Args: []introspect.Arg{{Type: "s"}}},
					{Name: "StatusNotifierItemUnregistered", Args: []introspect.Arg{{Type: "s"}}},
					{Name: "StatusNotifierHostRegistered"},
				},
			},
		},
	}
	return h.conn.Export(introspect.NewIntrospectable(node), watcherPath, "org.freedesktop.DBus.Introspectable")
}

// listen follows the items: their status, which says that the application took the icon away,
// and the applications, which say it by leaving the bus.
func (h *Host) listen() error {
	h.conn.Signal(h.signals)
	if err := h.conn.AddMatchSignal(dbus.WithMatchInterface(itemInterface), dbus.WithMatchMember("NewStatus")); err != nil {
		return err
	}
	if err := h.conn.AddMatchSignal(dbus.WithMatchSender("org.freedesktop.DBus"), dbus.WithMatchInterface("org.freedesktop.DBus"),
		dbus.WithMatchMember("NameOwnerChanged")); err != nil {
		return err
	}
	go h.dispatch()
	return nil
}

func (h *Host) dispatch() {
	for {
		select {
		case <-h.done:
			return
		case signal := <-h.signals:
			if signal == nil {
				return
			}
			switch signal.Name {
			case itemInterface + ".NewStatus":
				if status, _ := firstString(signal.Body); status == "Passive" {
					h.find(string(signal.Sender), signal.Path, func(item *Item) { item.markPassive() })
				}
			case "org.freedesktop.DBus.NameOwnerChanged":
				name, _ := firstString(signal.Body)
				if len(signal.Body) == 3 && signal.Body[2] == "" {
					h.find(name, "", func(item *Item) { item.markGone() })
				}
			}
		}
	}
}

// find calls fn for the items of the service, and of the path when it is not empty.
func (h *Host) find(service string, path dbus.ObjectPath, fn func(*Item)) {
	h.mu.Lock()
	var found []*Item
	for _, item := range h.items {
		if item.Service == service && (path == "" || item.Path == path) {
			found = append(found, item)
		}
	}
	h.mu.Unlock()
	for _, item := range found {
		fn(item)
	}
}

func firstString(body []any) (string, bool) {
	if len(body) == 0 {
		return "", false
	}
	s, ok := body[0].(string)
	return s, ok
}

// Close leaves the bus: the name of the watcher is free again.
func (h *Host) Close() error {
	select {
	case <-h.done:
		return nil
	default:
		close(h.done)
	}
	return h.conn.Close()
}

// NextItem waits for an application to register an item that has not been returned yet.
func (h *Host) NextItem(ctx context.Context) (*Item, error) {
	select {
	case item := <-h.arrived:
		return item, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("no application registered an icon in the tray: %w", ctx.Err())
	}
}

// Registered is how many items are registered and not yet gone or passive: the icons that
// a panel would show.
func (h *Host) Registered() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	var n int
	for _, item := range h.items {
		if !item.isPassive() && !item.isGone() {
			n++
		}
	}
	return n
}

// Item is an icon that an application registered.
type Item struct {
	h *Host
	// Service is the name of the bus of the application, and Path the object of its item.
	Service string
	Path    dbus.ObjectPath

	passive, gone         chan struct{}
	passiveOnce, goneOnce sync.Once
}

func (i *Item) markPassive() { i.passiveOnce.Do(func() { close(i.passive) }) }

// markGone: an application that left the bus has no icon either.
func (i *Item) markGone() {
	i.goneOnce.Do(func() { close(i.gone) })
	i.markPassive()
}

func (i *Item) isPassive() bool {
	select {
	case <-i.passive:
		return true
	default:
		return false
	}
}

func (i *Item) isGone() bool {
	select {
	case <-i.gone:
		return true
	default:
		return false
	}
}

// Properties are what a panel reads of an item to show it.
type Properties struct {
	ID       string
	Title    string
	Status   string
	Category string
	// IconName and IconThemePath say where the picture is: the panel looks for IconName in
	// the icons of the desktop and, first, in the directory IconThemePath.
	IconName      string
	IconThemePath string
	// Menu is the object of the menu of the item, which is a dbusmenu.
	Menu       dbus.ObjectPath
	ItemIsMenu bool
}

// Properties reads the properties of the item.
func (i *Item) Properties() (Properties, error) {
	var all map[string]dbus.Variant
	err := i.object(i.Path).Call("org.freedesktop.DBus.Properties.GetAll", 0, itemInterface).Store(&all)
	if err != nil {
		return Properties{}, fmt.Errorf("read the properties of %s%s: %w", i.Service, i.Path, err)
	}
	text := func(name string) string {
		s, _ := all[name].Value().(string)
		return s
	}
	p := Properties{
		ID: text("Id"), Title: text("Title"), Status: text("Status"), Category: text("Category"),
		IconName: text("IconName"), IconThemePath: text("IconThemePath"),
	}
	p.Menu, _ = all["Menu"].Value().(dbus.ObjectPath)
	p.ItemIsMenu, _ = all["ItemIsMenu"].Value().(bool)
	return p, nil
}

func (i *Item) object(path dbus.ObjectPath) dbus.BusObject { return i.h.conn.Object(i.Service, path) }

// Entry is an item of the menu, as the panel shows it.
type Entry struct {
	ID    int32
	Label string
	// Separator is true for the line between groups of entries, which has no label.
	Separator bool
	Enabled   bool
	Visible   bool
}

// layout is a node of the layout of a dbusmenu, (ia{sv}av): its id, its properties and its children.
type layout struct {
	ID         int32
	Properties map[string]dbus.Variant
	Children   []dbus.Variant
}

// Entries reads the menu of the item, as a panel does when it is about to show it: it tells the
// menu that it will, and asks for the layout.
func (i *Item) Entries() ([]Entry, error) {
	properties, err := i.Properties()
	if err != nil {
		return nil, err
	}
	if properties.Menu == "" {
		return nil, fmt.Errorf("the item %s%s has no menu", i.Service, i.Path)
	}
	menu := i.object(properties.Menu)
	var needsUpdate bool
	if err := menu.Call(menuInterface+".AboutToShow", 0, int32(0)).Store(&needsUpdate); err != nil {
		return nil, fmt.Errorf("the menu is about to show: %w", err)
	}
	var revision uint32
	var root layout
	if err := menu.Call(menuInterface+".GetLayout", 0, int32(0), int32(-1), []string{}).Store(&revision, &root); err != nil {
		return nil, fmt.Errorf("read the layout of the menu: %w", err)
	}
	entries := make([]Entry, 0, len(root.Children))
	for _, child := range root.Children {
		var node layout
		if err := child.Store(&node); err != nil {
			return nil, fmt.Errorf("read an entry of the menu: %w", err)
		}
		entries = append(entries, entryOf(node))
	}
	return entries, nil
}

func entryOf(node layout) Entry {
	flag := func(name string, fallback bool) bool {
		if v, ok := node.Properties[name]; ok {
			if b, ok := v.Value().(bool); ok {
				return b
			}
		}
		return fallback
	}
	entry := Entry{ID: node.ID, Enabled: flag("enabled", true), Visible: flag("visible", true)}
	if kind, _ := node.Properties["type"].Value().(string); kind == "separator" {
		entry.Separator = true
		return entry
	}
	entry.Label, _ = node.Properties["label"].Value().(string)
	return entry
}

// Choose chooses an entry of the menu, as a click on it does: the application is sent the event
// "clicked" for it. The index counts the entries of the menu, the separators too.
func (i *Item) Choose(index int) error {
	entries, err := i.Entries()
	if err != nil {
		return err
	}
	if index < 0 || index >= len(entries) {
		return fmt.Errorf("the menu has %d entries, none at %d", len(entries), index)
	}
	if entries[index].Separator || !entries[index].Enabled {
		return fmt.Errorf("the entry %d of the menu cannot be chosen: %+v", index, entries[index])
	}
	properties, err := i.Properties()
	if err != nil {
		return err
	}
	data := dbus.MakeVariant(int32(0))
	call := i.object(properties.Menu).Call(menuInterface+".Event", 0, entries[index].ID, "clicked", data, uint32(time.Now().Unix()))
	if call.Err != nil {
		return fmt.Errorf("choose %q: %w", entries[index].Label, call.Err)
	}
	return nil
}

// Activate calls Activate on the item, as a panel does for a double click, or for a click when the
// icon has no menu. It returns ErrNoActivation when the item does not take the call.
func (i *Item) Activate() error {
	err := i.object(i.Path).Call(itemInterface+".Activate", 0, int32(0), int32(0)).Err
	var failure dbus.Error
	if errors.As(err, &failure) && failure.Name == "org.freedesktop.DBus.Error.UnknownMethod" {
		return ErrNoActivation
	}
	return err
}

// WaitPassive waits for the application to take the icon away: its status becomes Passive, or
// it leaves the bus.
func (i *Item) WaitPassive(ctx context.Context) error {
	select {
	case <-i.passive:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("the icon is still in the tray: %w", ctx.Err())
	}
}

// WaitGone waits for the application that made the item to leave the bus.
func (i *Item) WaitGone(ctx context.Context) error {
	select {
	case <-i.gone:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("the application is still on the bus: %w", ctx.Err())
	}
}

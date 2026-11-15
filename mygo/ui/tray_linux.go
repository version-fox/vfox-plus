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
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
)

// The icon in the tray of Linux is an item of the StatusNotifierItem protocol, which the panels of
// the desktops show: the panel of KDE Plasma, of Xfce, and the AppIndicator extension of GNOME, which
// Ubuntu enables. The item, and its menu as a dbusmenu, are put on the session bus here, and not by a
// library: the protocol gives each icon a name of its own on the bus, and the panel takes the icon
// away when that name goes. So each time the window closes, the icon is an item under a name that no
// icon had before, and when the window opens the name is given up. A library that keeps one item for
// the life of the application leaves GNOME Shell with an icon that it has hidden, and the shell does
// not show it again.

const (
	sniWatcherName      = "org.kde.StatusNotifierWatcher"
	sniWatcherPath      = dbus.ObjectPath("/StatusNotifierWatcher")
	sniWatcherInterface = "org.kde.StatusNotifierWatcher"
	sniItemInterface    = "org.kde.StatusNotifierItem"
	sniItemPath         = dbus.ObjectPath("/StatusNotifierItem")
	sniMenuInterface    = "com.canonical.dbusmenu"
	sniMenuPath         = dbus.ObjectPath("/MenuBar")
	// sniNamePrefix begins the name of an icon on the bus: org.freedesktop.StatusNotifierItem-PID-N.
	sniNamePrefix = "org.freedesktop.StatusNotifierItem-"
	// sniRegisterWait is how long the watcher may take to take an icon.
	sniRegisterWait = 5 * time.Second
)

// sniInstances numbers the icons that the process makes, so that each one has a name of its own.
var sniInstances atomic.Uint64

// errNoTrayHost says that nothing on the desktop shows tray icons.
var errNoTrayHost = errors.New("nothing on the desktop shows tray icons")

// newNativeTray puts an icon in the tray of the desktop. It fails where there is no session bus, which
// is where no host of the icons can be. Where the bus is there, but nothing owns the name of the watcher
// yet, the icon waits: the watcher takes it when it starts (see watchWatcher).
func newNativeTray(options trayOptions) (trayIcon, error) {
	picture, err := sniPixmapOf(options.icon)
	if err != nil {
		return nil, fmt.Errorf("draw the icon: %w", err)
	}
	conn, err := connectSessionBus()
	if err != nil {
		return nil, errNoTrayHost
	}
	name := sniBusName(os.Getpid(), sniInstances.Add(1))
	reply, err := conn.RequestName(name, dbus.NameFlagDoNotQueue)
	if err == nil && reply != dbus.RequestNameReplyPrimaryOwner {
		err = errors.New("the name is taken")
	}
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("take the name %s on the session bus: %w", name, err)
	}
	tray := &sniTray{conn: conn, name: name, entries: options.entries, done: make(chan struct{})}
	if err := tray.export(options, picture); err != nil {
		tray.Destroy()
		return nil, fmt.Errorf("put the icon on the bus: %w", err)
	}
	// The watcher is followed before it is asked, so that a watcher which starts in between is not missed.
	if err := tray.watchWatcher(); err != nil {
		tray.Destroy()
		return nil, fmt.Errorf("follow the watcher: %w", err)
	}
	var owned bool
	if err := conn.BusObject().Call("org.freedesktop.DBus.NameHasOwner", 0, sniWatcherName).Store(&owned); err != nil {
		tray.Destroy()
		return nil, fmt.Errorf("ask the bus for the watcher: %w", err)
	}
	if owned {
		if err := tray.register(); err != nil {
			tray.Destroy()
			return nil, err
		}
	} else {
		// The icon is kept, and the watcher takes it when a desktop starts one (see watchWatcher).
		log.Print("the icon waits for a host of the tray icons: nothing on the desktop shows them yet")
	}
	return tray, nil
}

// connectSessionBus connects to the session bus that the desktop has. It does not start one where
// there is none, as dbus.ConnectSessionBus would: a desktop without a session bus has no host of the
// icons either, and the application has no business starting a bus to look for one.
func connectSessionBus() (*dbus.Conn, error) {
	conn, err := dbus.SessionBusPrivateNoAutoStartup()
	if err != nil {
		return nil, err
	}
	if err := conn.Auth(nil); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := conn.Hello(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

// sniBusName is the name of an icon on the session bus: the name of the specification for an item
// of an application, with the process and a number of its own.
func sniBusName(pid int, n uint64) string {
	return sniNamePrefix + strconv.Itoa(pid) + "-" + strconv.FormatUint(n, 10)
}

// sniTray is an icon in the tray: an item on the session bus under a name of its own, with its menu.
// The desktop shows the item while the name is on the bus, and Destroy gives the name up.
type sniTray struct {
	conn      *dbus.Conn
	name      string
	itemProps *prop.Properties
	// done is closed when the icon is destroyed, which ends the following of the watcher.
	done chan struct{}

	mu       sync.Mutex
	entries  []trayEntry
	revision uint32
	closed   bool
	once     sync.Once
}

// export puts the item and its menu on the bus, with what the desktop reads of them.
func (t *sniTray) export(options trayOptions, picture sniPixmap) error {
	props, err := prop.Export(t.conn, sniItemPath, sniItemPropertyMap(options.toolTip, picture))
	if err != nil {
		return err
	}
	t.itemProps = props
	item := &sniItem{activate: options.activate}
	if err := t.conn.Export(item, sniItemPath, sniItemInterface); err != nil {
		return err
	}
	if err := describe(t.conn, sniItemPath, sniItemInterface, item, props, []introspect.Signal{
		{Name: "NewTitle"},
		{Name: "NewToolTip"},
		{Name: "NewStatus", Args: []introspect.Arg{{Type: "s"}}},
	}); err != nil {
		return err
	}
	menuProps, err := prop.Export(t.conn, sniMenuPath, sniMenuPropertyMap())
	if err != nil {
		return err
	}
	menu := &sniMenu{tray: t}
	if err := t.conn.Export(menu, sniMenuPath, sniMenuInterface); err != nil {
		return err
	}
	return describe(t.conn, sniMenuPath, sniMenuInterface, menu, menuProps, []introspect.Signal{
		{Name: "LayoutUpdated", Args: []introspect.Arg{{Type: "u"}, {Type: "i"}}},
	})
}

// describe says what the object at path is, for the tools that ask for it (busctl introspect): its
// methods, its properties and its signals.
func describe(conn *dbus.Conn, path dbus.ObjectPath, iface string, object any, props *prop.Properties, signals []introspect.Signal) error {
	node := &introspect.Node{
		Name: string(path),
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			{
				Name:       iface,
				Methods:    introspect.Methods(object),
				Properties: props.Introspection(iface),
				Signals:    signals,
			},
		},
	}
	return conn.Export(introspect.NewIntrospectable(node), path, "org.freedesktop.DBus.Introspectable")
}

// sniItemPropertyMap are the properties of the item: what the icon is, its picture, its title and tool
// tip, its status (Active is shown, Passive is not) and its menu. The picture is in the item, with no
// name in a theme, and the item is a menu, so that a click on the icon shows the menu.
func sniItemPropertyMap(toolTip string, picture sniPixmap) prop.Map {
	return prop.Map{sniItemInterface: {
		"Category":            {Value: "ApplicationStatus", Emit: prop.EmitFalse},
		"Id":                  {Value: ApplicationID, Emit: prop.EmitFalse},
		"Title":               {Value: toolTip, Emit: prop.EmitTrue},
		"Status":              {Value: "Active", Emit: prop.EmitTrue},
		"WindowId":            {Value: int32(0), Emit: prop.EmitFalse},
		"IconName":            {Value: "", Emit: prop.EmitFalse},
		"IconPixmap":          {Value: []sniPixmap{picture}, Emit: prop.EmitFalse},
		"OverlayIconName":     {Value: "", Emit: prop.EmitFalse},
		"OverlayIconPixmap":   {Value: []sniPixmap{}, Emit: prop.EmitFalse},
		"AttentionIconName":   {Value: "", Emit: prop.EmitFalse},
		"AttentionIconPixmap": {Value: []sniPixmap{}, Emit: prop.EmitFalse},
		"AttentionMovieName":  {Value: "", Emit: prop.EmitFalse},
		"ToolTip":             {Value: sniToolTip{Title: toolTip}, Emit: prop.EmitTrue},
		"IconThemePath":       {Value: "", Emit: prop.EmitFalse},
		"ItemIsMenu":          {Value: true, Emit: prop.EmitFalse},
		"Menu":                {Value: sniMenuPath, Emit: prop.EmitFalse},
	}}
}

// sniMenuPropertyMap are the properties of the dbusmenu, which say the version of the protocol and
// how the menu reads.
func sniMenuPropertyMap() prop.Map {
	return prop.Map{sniMenuInterface: {
		"Version":       {Value: uint32(3), Emit: prop.EmitFalse},
		"TextDirection": {Value: "ltr", Emit: prop.EmitFalse},
		"Status":        {Value: "normal", Emit: prop.EmitFalse},
		"IconThemePath": {Value: []string{}, Emit: prop.EmitFalse},
	}}
}

// Update sets the title, the tool tip and the menu again, as when the language changed.
func (t *sniTray) Update(toolTip string, entries []trayEntry) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return
	}
	t.entries = entries
	t.revision++
	revision := t.revision
	t.mu.Unlock()
	t.itemProps.SetMust(sniItemInterface, "Title", toolTip)
	t.itemProps.SetMust(sniItemInterface, "ToolTip", sniToolTip{Title: toolTip})
	_ = t.conn.Emit(sniItemPath, sniItemInterface+".NewTitle")
	_ = t.conn.Emit(sniItemPath, sniItemInterface+".NewToolTip")
	_ = t.conn.Emit(sniMenuPath, sniMenuInterface+".LayoutUpdated", revision, int32(0))
}

// Destroy takes the icon away. The item becomes passive, and then gives its name up, which is how the
// desktop learns that the icon is gone. The bus answers the release only after it has had the
// signal of the status before it, so the status is out before the name is.
func (t *sniTray) Destroy() {
	t.once.Do(func() {
		t.mu.Lock()
		t.closed = true
		t.mu.Unlock()
		close(t.done)
		if t.itemProps != nil {
			t.itemProps.SetMust(sniItemInterface, "Status", "Passive")
		}
		_ = t.conn.Emit(sniItemPath, sniItemInterface+".NewStatus", "Passive")
		_, _ = t.conn.ReleaseName(t.name)
		_ = t.conn.Close()
	})
}

// register asks the watcher to take the icon, under the name of the icon on the bus.
func (t *sniTray) register() error {
	ctx, cancel := context.WithTimeout(context.Background(), sniRegisterWait)
	defer cancel()
	call := t.conn.Object(sniWatcherName, sniWatcherPath).CallWithContext(ctx, sniWatcherInterface+".RegisterStatusNotifierItem", 0, t.name)
	if call.Err != nil {
		return fmt.Errorf("register the icon with the watcher: %w", call.Err)
	}
	return nil
}

// watchWatcher follows the owner of the watcher on the bus, so that the watcher takes the icon when it
// starts after the icon was made, and again when it starts over: a shell that restarts has none of the
// icons that it had, and a desktop that has no watcher yet takes the icon when it gets one.
func (t *sniTray) watchWatcher() error {
	if err := t.conn.AddMatchSignal(
		dbus.WithMatchSender("org.freedesktop.DBus"),
		dbus.WithMatchInterface("org.freedesktop.DBus"),
		dbus.WithMatchMember("NameOwnerChanged"),
		dbus.WithMatchArg(0, sniWatcherName),
	); err != nil {
		return err
	}
	signals := make(chan *dbus.Signal, 8)
	t.conn.Signal(signals)
	go watchOwners(signals, t.done, func() {
		if t.isClosed() {
			return
		}
		if err := t.register(); err != nil {
			log.Print(err)
		}
	})
	return nil
}

// watchOwners calls register for each signal of the bus that says that the watcher has an owner. It
// returns when done is closed, or when the signals end.
func watchOwners(signals <-chan *dbus.Signal, done <-chan struct{}, register func()) {
	for {
		select {
		case <-done:
			return
		case signal, ok := <-signals:
			if !ok {
				return
			}
			if sniOwnerTaken(signal) {
				register()
			}
		}
	}
}

// sniOwnerTaken says whether a signal of the bus says that the watcher has an owner: the owner of its
// name has changed to one, from none or from another.
func sniOwnerTaken(signal *dbus.Signal) bool {
	if signal == nil || signal.Name != "org.freedesktop.DBus.NameOwnerChanged" || len(signal.Body) != 3 {
		return false
	}
	name, _ := signal.Body[0].(string)
	owner, _ := signal.Body[2].(string)
	return name == sniWatcherName && owner != ""
}

// isClosed says whether the icon is destroyed.
func (t *sniTray) isClosed() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.closed
}

// clickOf is what the entry with the id does when a desktop chooses it: nil for a separator, for the
// root of the menu, and for an id that the menu does not have.
func (t *sniTray) clickOf(id int32) func() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if id < 1 || int(id) > len(t.entries) {
		return nil
	}
	return t.entries[id-1].click
}

// propertiesOf are the properties of the node with the id, and false for an id that the menu does
// not have.
func (t *sniTray) propertiesOf(id int32) (map[string]dbus.Variant, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if id == 0 {
		return sniRootNodeProperties(), true
	}
	if id < 1 || int(id) > len(t.entries) {
		return nil, false
	}
	return sniNode(id, t.entries[id-1]).Properties, true
}

// ids are the ids of the nodes of the menu: the root, and the entries.
func (t *sniTray) ids() []int32 {
	t.mu.Lock()
	defer t.mu.Unlock()
	ids := []int32{0}
	for i := range t.entries {
		ids = append(ids, int32(i+1))
	}
	return ids
}

// sniItem is the object of the item. A desktop calls its Activate when the icon is activated and
// the item is not a menu. The item is one (ItemIsMenu), so a click shows the menu, from which the
// window comes back; Activate is for a desktop that calls it all the same.
type sniItem struct{ activate func() }

// Activate brings the window back, as the activation of an icon does on Windows.
func (i *sniItem) Activate(_, _ int32) *dbus.Error {
	if i.activate != nil {
		mygo.RunOnMain(i.activate)
	}
	return nil
}

// sniMenu is the menu of the icon, a dbusmenu: a desktop reads its layout and sends it the events of
// its entries. The root is 0, and the entries are 1 and up, in the order they have in the tray.
type sniMenu struct{ tray *sniTray }

// GetLayout is the layout of the menu, with the entries as the children of the root. The menu has one
// level, so the depth does not change it.
func (m *sniMenu) GetLayout(parentID, _ int32, _ []string) (uint32, sniMenuNode, *dbus.Error) {
	m.tray.mu.Lock()
	defer m.tray.mu.Unlock()
	if parentID != 0 {
		return m.tray.revision, sniMenuNode{ID: parentID}, nil
	}
	return m.tray.revision, sniLayout(m.tray.entries), nil
}

// GetGroupProperties are the properties of the nodes with the ids, or of all of them without ids.
func (m *sniMenu) GetGroupProperties(ids []int32, _ []string) ([]sniEntryProperties, *dbus.Error) {
	if len(ids) == 0 {
		ids = m.tray.ids()
	}
	var out []sniEntryProperties
	for _, id := range ids {
		if properties, ok := m.tray.propertiesOf(id); ok {
			out = append(out, sniEntryProperties{ID: id, Properties: properties})
		}
	}
	return out, nil
}

// GetProperty is one property of the node with the id.
func (m *sniMenu) GetProperty(id int32, name string) (dbus.Variant, *dbus.Error) {
	properties, ok := m.tray.propertiesOf(id)
	if !ok {
		return dbus.Variant{}, dbus.MakeFailedError(fmt.Errorf("the menu has no entry %d", id))
	}
	if value, ok := properties[name]; ok {
		return value, nil
	}
	return dbus.MakeVariant(""), nil
}

// Event runs the entry that a desktop chose: the event that says so is "clicked". The click runs on
// the main thread, where the window is.
func (m *sniMenu) Event(id int32, eventID string, _ dbus.Variant, _ uint32) *dbus.Error {
	if eventID != "clicked" {
		return nil
	}
	if click := m.tray.clickOf(id); click != nil {
		mygo.RunOnMain(click)
	}
	return nil
}

// EventGroup is Event for each of the events.
func (m *sniMenu) EventGroup(events []sniMenuEvent) ([]int32, *dbus.Error) {
	for _, event := range events {
		_ = m.Event(event.ID, event.EventID, event.Data, event.Timestamp)
	}
	return nil, nil
}

// AboutToShow says whether the menu must be read again before it shows. The entries change only with
// the revision of the layout, which the menu says with LayoutUpdated, so it need not.
func (m *sniMenu) AboutToShow(int32) (bool, *dbus.Error) { return false, nil }

// AboutToShowGroup is AboutToShow for a group of nodes: none of them needs to be read again.
func (m *sniMenu) AboutToShowGroup([]int32) ([]int32, []int32, *dbus.Error) { return nil, nil, nil }

// sniLayout is the layout of the menu as a desktop reads it: the root, with an entry for each entry
// of the tray as its children.
func sniLayout(entries []trayEntry) sniMenuNode {
	root := sniMenuNode{ID: 0, Properties: sniRootNodeProperties()}
	for i, entry := range entries {
		root.Children = append(root.Children, dbus.MakeVariant(sniNode(int32(i+1), entry)))
	}
	return root
}

// sniNode is an entry of the menu: a separator when the entry does nothing, and otherwise an item
// with its label.
func sniNode(id int32, entry trayEntry) sniMenuNode {
	if entry.click == nil {
		return sniMenuNode{ID: id, Properties: map[string]dbus.Variant{"type": dbus.MakeVariant("separator")}}
	}
	return sniMenuNode{ID: id, Properties: map[string]dbus.Variant{"label": dbus.MakeVariant(sniLabel(entry.label))}}
}

// sniRootNodeProperties are the properties of the root of the menu, whose children are its submenu.
func sniRootNodeProperties() map[string]dbus.Variant {
	return map[string]dbus.Variant{"children-display": dbus.MakeVariant("submenu")}
}

// sniLabel is the label of an entry. In a dbusmenu a single underscore marks the mnemonic of the
// character after it, so an underscore of the text is doubled.
func sniLabel(text string) string {
	return strings.ReplaceAll(text, "_", "__")
}

// sniMenuNode is a node of a dbusmenu layout, (ia{sv}av): its id, its properties and its children.
type sniMenuNode struct {
	ID         int32
	Properties map[string]dbus.Variant
	Children   []dbus.Variant
}

// sniEntryProperties are the properties of a node, (ia{sv}).
type sniEntryProperties struct {
	ID         int32
	Properties map[string]dbus.Variant
}

// sniMenuEvent is an event of a node, (isvu).
type sniMenuEvent struct {
	ID        int32
	EventID   string
	Data      dbus.Variant
	Timestamp uint32
}

// sniPixmap is a picture of an icon, as the protocol has it: its size, and its pixels from the top
// left, four bytes each, in the order alpha, red, green and blue.
type sniPixmap struct {
	Width  int32
	Height int32
	Pixels []byte
}

// sniToolTip is the tool tip of an item: an icon, and a title and a text of its own.
type sniToolTip struct {
	IconName    string
	IconPixmap  []sniPixmap
	Title       string
	Description string
}

// sniPixmapOf is the picture of an icon as the protocol takes it: the pixels of the PNG, with the
// alpha of each first, and not the picture of a file that the desktop would have to find.
func sniPixmapOf(picture []byte) (sniPixmap, error) {
	img, err := png.Decode(bytes.NewReader(picture))
	if err != nil {
		return sniPixmap{}, err
	}
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 || width > math.MaxInt32 || height > math.MaxInt32 {
		return sniPixmap{}, fmt.Errorf("the picture is %d by %d pixels", width, height)
	}
	pixels := make([]byte, 0, 4*width*height)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			pixels = append(pixels, c.A, c.R, c.G, c.B)
		}
	}
	return sniPixmap{Width: int32(width), Height: int32(height), Pixels: pixels}, nil
}

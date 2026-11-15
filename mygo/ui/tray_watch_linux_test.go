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
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
)

// ownerChanged is the signal of the bus that says that the owner of a name has changed.
func ownerChanged(name, previous, next string) *dbus.Signal {
	return &dbus.Signal{Name: "org.freedesktop.DBus.NameOwnerChanged", Body: []any{name, previous, next}}
}

// TestOnlyAWatcherThatGetsAnOwnerTakesTheIcon checks which signals of the bus make an icon register: the
// watcher getting an owner, from none or from another (a shell that starts again), and nothing else.
func TestOnlyAWatcherThatGetsAnOwnerTakesTheIcon(t *testing.T) {
	cases := []struct {
		name   string
		signal *dbus.Signal
		want   bool
	}{
		{"the watcher starts", ownerChanged(sniWatcherName, "", ":1.5"), true},
		{"the watcher starts again", ownerChanged(sniWatcherName, ":1.5", ":1.9"), true},
		{"the watcher goes", ownerChanged(sniWatcherName, ":1.5", ""), false},
		{"another name gets an owner", ownerChanged("org.example.Other", "", ":1.6"), false},
		{"another signal of the bus", &dbus.Signal{Name: "org.freedesktop.DBus.NameAcquired", Body: []any{sniWatcherName}}, false},
		{"no signal", nil, false},
	}
	for _, c := range cases {
		if got := sniOwnerTaken(c.signal); got != c.want {
			t.Errorf("%s: %t, want %t", c.name, got, c.want)
		}
	}
}

// TestTheIconRegistersEachTimeTheWatcherGetsAnOwnerUntilItIsDone checks the following of the watcher: the
// icon registers each time that the watcher gets an owner, and not once the following is done.
func TestTheIconRegistersEachTimeTheWatcherGetsAnOwnerUntilItIsDone(t *testing.T) {
	signals := make(chan *dbus.Signal, 8)
	done := make(chan struct{})
	registered := make(chan struct{}, 8)
	finished := make(chan struct{})
	go func() {
		watchOwners(signals, done, func() { registered <- struct{}{} })
		close(finished)
	}()
	signals <- ownerChanged(sniWatcherName, "", ":1.5")
	signals <- ownerChanged(sniWatcherName, ":1.5", "")
	signals <- ownerChanged(sniWatcherName, ":1.5", ":1.9")
	for i := 0; i < 2; i++ {
		select {
		case <-registered:
		case <-time.After(5 * time.Second):
			t.Fatalf("the icon registered %d times, want 2", i)
		}
	}
	if len(registered) != 0 {
		t.Errorf("the icon registered %d more times than the watcher got an owner", len(registered))
	}

	close(done)
	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("the following goes on after the icon is destroyed")
	}
	signals <- ownerChanged(sniWatcherName, "", ":1.12")
	if len(registered) != 0 {
		t.Error("the icon registers after it is destroyed")
	}
}

// TestAnIconThatWaitsHasNoConnectionAndKeepsTheTextToShow checks that an icon which waits for a host holds
// no connection to the bus, and keeps the text and the menu that it is given while it waits, for when it
// is shown.
func TestAnIconThatWaitsHasNoConnectionAndKeepsTheTextToShow(t *testing.T) {
	tray := newSNITray(trayOptions{toolTip: "vfox+"}, sniPixmap{})
	defer tray.Destroy()
	tray.Update("vfox+ updated", []trayEntry{{label: "Open vfox+", click: func() {}}, {}, {label: "Quit vfox+", click: func() {}}})
	if tray.conn != nil || tray.itemProps != nil {
		t.Error("an icon that waits holds a connection to the bus")
	}
	if tray.toolTip != "vfox+ updated" || len(tray.entries) != 3 {
		t.Errorf("the text %q and %d entries, want the text and the menu of the update", tray.toolTip, len(tray.entries))
	}
}

// TestAnIconDestroyedWhileItWaitsEndsTheWait checks that destroying an icon which waits for a host ends
// the wait at once, and that a text given afterwards is not kept.
func TestAnIconDestroyedWhileItWaitsEndsTheWait(t *testing.T) {
	tray := newSNITray(trayOptions{toolTip: "vfox+"}, sniPixmap{})
	finished := make(chan struct{})
	go func() {
		tray.waitForHost()
		close(finished)
	}()
	tray.Destroy()
	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("the wait goes on after the icon is destroyed")
	}
	tray.Update("late", []trayEntry{{label: "Late"}})
	if tray.toolTip != "vfox+" || len(tray.entries) != 0 {
		t.Errorf("a destroyed icon took the text %q and %d entries", tray.toolTip, len(tray.entries))
	}
}

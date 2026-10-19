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

package kit

import (
	"github.com/egoist/mygo/ui"
)

// Role is what a node is, as an accessibility tree would say it.
type Role string

// The roles of the nodes.
const (
	RoleButton    Role = "button"
	RoleTab       Role = "tab"
	RoleListItem  Role = "listitem"
	RoleMenuItem  Role = "menuitem"
	RoleSwitch    Role = "switch"
	RoleTextField Role = "textfield"
	RoleText      Role = "text"
	RoleDialog    Role = "dialog"
)

// Node is what a frame knows about one widget: where it is, what it is
// called and what a click on it does. Interactive widgets register one, and,
// when the registry collects text, every label does, too.
type Node struct {
	// Role says what the widget is.
	Role Role
	// Name is the accessible name: the label, or the tooltip of icon buttons.
	Name string
	// Text is the text that a label shows.
	Text string
	// Rect is where the widget was drawn last, in dp of the window. It is empty
	// for a widget that no frame has drawn yet.
	Rect ui.Rect
	// Enabled says whether the widget reacts to input.
	Enabled bool
	// Selected says whether the widget is selected or switched on.
	Selected bool
	// Modal says that the widget belongs to a dialog or a popup, which block
	// everything below them.
	Modal bool
	// Scope names what the widget is in: the title of the list item that holds it,
	// or "dialog", which finds a widget that the window has not drawn, because it
	// scrolled out of view.
	Scope string
	// Tooltip is shown when the pointer rests on the widget.
	Tooltip string
	// Click does what a click on the widget does.
	Click func()
	// Fill puts text into a text field and, with submit, presses Enter.
	Fill func(text string, submit bool)
	// Value reads the text of a text field.
	Value func() string
}

// Registry collects the nodes of a frame.
type Registry struct {
	// CollectText makes labels register, too. Only tests need them.
	CollectText bool

	nodes  []*Node
	modal  bool
	scope  string
	builds uint64
}

func (r *Registry) begin() {
	clear(r.nodes)
	r.nodes = r.nodes[:0]
	r.builds++
	r.modal = false
	r.scope = ""
}

// Builds counts the builds of the view the registry has seen.
func (r *Registry) Builds() uint64 { return r.builds }

// Add registers a node and returns it, so that the drawing of the widget can
// tell where it is.
func (r *Registry) Add(n Node) *Node {
	n.Modal = n.Modal || r.modal
	if n.Scope == "" {
		n.Scope = r.scope
	}
	node := &n
	r.nodes = append(r.nodes, node)
	return node
}

// SetModal makes the nodes added from now on modal, until it is called with
// false. Dialogs and menus call it around their content.
func (r *Registry) SetModal(modal bool) { r.modal = modal }

// EnterScope makes the nodes added from now on belong to what is called name, until
// the function it returns is called.
func (r *Registry) EnterScope(name string) (leave func()) {
	previous := r.scope
	r.scope = name
	return func() { r.scope = previous }
}

// Nodes returns the nodes of the last build, in the order they were built:
// later nodes are on top of earlier ones.
func (r *Registry) Nodes() []*Node { return r.nodes }

// Visible reports whether the widget has been drawn.
func (n *Node) Visible() bool { return n.Rect.W > 0 && n.Rect.H > 0 }

// Inside reports whether the node lies inside the other.
func (n *Node) Inside(other *Node) bool {
	return n.Visible() && other.Visible() &&
		n.Rect.X >= other.Rect.X && n.Rect.Y >= other.Rect.Y &&
		n.Rect.X+n.Rect.W <= other.Rect.X+other.Rect.W+0.5 && n.Rect.Y+n.Rect.H <= other.Rect.Y+other.Rect.H+0.5
}

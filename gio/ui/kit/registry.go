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

import "image"

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
	// Key identifies the widget, for popups that open at it.
	Key any
	// Role says what the widget is.
	Role Role
	// Name is the accessible name: the label, or the tooltip of icon buttons.
	Name string
	// Text is the text that a label shows.
	Text string
	// Rect is where the widget is. It is relative to the container that is
	// being laid out, and to the window once the frame is complete.
	Rect image.Rectangle
	// Enabled says whether the widget reacts to input.
	Enabled bool
	// Selected says whether the widget is selected or switched on.
	Selected bool
	// Hovered says whether the pointer is over the widget.
	Hovered bool
	// Focused says whether the widget has the keyboard focus.
	Focused bool
	// Modal says that the widget belongs to a dialog or a popup, which block
	// everything below them.
	Modal bool
	// Tooltip is shown when the pointer rests on the widget.
	Tooltip string
	// Click does what a click on the widget does.
	Click func()
	// Fill puts text into a text field and, with submit, presses Enter.
	Fill func(text string, submit bool)
}

// Registry collects the nodes of a frame.
type Registry struct {
	// CollectText makes labels register, too. Only tests need them.
	CollectText bool

	nodes  []Node
	modal  bool
	frames uint64
}

func (r *Registry) begin() {
	r.nodes = r.nodes[:0]
	r.frames++
}

// Frames counts the frames the registry has seen.
func (r *Registry) Frames() uint64 { return r.frames }

// Add registers a node, in the coordinates of the container being laid out.
func (r *Registry) Add(n Node) {
	n.Modal = n.Modal || r.modal
	r.nodes = append(r.nodes, n)
}

// Mark returns the position to pass to Shift and Truncate later.
func (r *Registry) Mark() int { return len(r.nodes) }

// Shift moves the nodes registered since mark by d, which is how a container
// tells the nodes of a child where the child went.
func (r *Registry) Shift(mark int, d image.Point) {
	if d == (image.Point{}) {
		return
	}
	r.ShiftRange(mark, len(r.nodes), d)
}

// ShiftRange moves the nodes from index from up to, not including, to.
func (r *Registry) ShiftRange(from, to int, d image.Point) {
	if d == (image.Point{}) {
		return
	}
	for i := from; i < to && i < len(r.nodes); i++ {
		r.nodes[i].Rect = r.nodes[i].Rect.Add(d)
	}
}

// Truncate forgets the nodes registered since mark, for a child that is laid
// out again.
func (r *Registry) Truncate(mark int) {
	if mark < len(r.nodes) {
		r.nodes = r.nodes[:mark]
	}
}

// SetModal makes the nodes added from now on modal, until it is called with
// false. Dialogs and menus call it around their content.
func (r *Registry) SetModal(modal bool) { r.modal = modal }

// Nodes returns the nodes of the last complete frame, in the order they were
// laid out: later nodes are on top of earlier ones.
func (r *Registry) Nodes() []Node { return r.nodes }

// Find returns the last node that has key, which is the one on top.
func (r *Registry) Find(key any) (Node, bool) {
	for i := len(r.nodes) - 1; i >= 0; i-- {
		if r.nodes[i].Key == key {
			return r.nodes[i], true
		}
	}
	return Node{}, false
}

// Visible reports whether any part of the node is inside the window.
func Visible(n Node, window image.Point) bool {
	return !n.Rect.Empty() && n.Rect.Overlaps(image.Rectangle{Max: window})
}

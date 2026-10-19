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

// TextFieldSpec describes a Material 3 filled text field, with a label that
// floats above the text, helper and error text, and optional icons.
type TextFieldSpec struct {
	// ID identifies the field among its siblings between builds.
	ID string
	// Label floats above the text once the field has the focus or a text.
	Label string
	// Hint is shown in an empty field that has no label, or that has the focus.
	Hint string
	// Helper is shown below the field.
	Helper string
	// Error is shown below the field instead of the helper, and makes the field
	// the colour of errors.
	Error string
	// Text is where the field keeps its text, for code that reads it when the
	// user asks to save. Without it, the field keeps the text itself and
	// follows Value.
	Text *string
	// Value is the text to show when Text is nil. The field keeps what the user
	// types until the value changes.
	Value string
	// OnChange runs with the text whenever the user changes it.
	OnChange func(text string)
	// OnSubmit runs with the text when the user presses Enter.
	OnSubmit func(text string)
	// OnBlur runs with the text when the field loses the focus.
	OnBlur func(text string)
	// LeadingIcon is an icon before the text.
	LeadingIcon string
	// Trailing builds what is at the end of the field, such as an icon button.
	Trailing func()
	// Lines is the number of lines the field shows at least; 1 for a field with
	// a single line.
	Lines int
	// MaxLines is the number of lines the field grows to before it scrolls.
	MaxLines int
	// Monospace draws the text in the monospaced font.
	Monospace bool
	// Dense makes the field lower.
	Dense bool
	// Disabled makes the field inert.
	Disabled bool
	// AutoFocus gives the field the focus when it first appears.
	AutoFocus bool
}

// fieldState is what a field keeps between builds.
type fieldState struct {
	text     string
	last     string
	wasFocus bool
}

// buffer is where the text of the field is: the string the caller gave, or the
// one the field keeps for itself, which follows the value.
func (st *fieldState) buffer(text *string, value string) *string {
	if text != nil {
		return text
	}
	if value != st.last {
		st.text, st.last = value, value
	}
	return &st.text
}

// TextField is a Material 3 filled text field.
func TextField(c Ctx, f TextFieldSpec) ui.Element {
	cs := c.Colors()
	if hasKey(f.ID) {
		c.Key(f.ID)
	}
	outer := ui.Column(c.Context).Gap(4)
	st := ui.Local(outer, "field", func() fieldState { return fieldState{text: f.Value, last: f.Value} })
	buf := st.buffer(f.Text, f.Value)

	lines := max(1, f.Lines)
	maxLines := max(lines, f.MaxLines)
	single := lines == 1 && maxLines == 1
	style := Type(BodyLarge)
	if f.Monospace || f.Dense {
		style = Type(BodyMedium)
	}
	hasLabel := f.Label != "" && !f.Dense
	hasError := f.Error != ""
	scale := c.Env.Theme.TextScale
	height := float32(56)
	padding := []float32{0, 16}
	switch {
	case f.Dense:
		height = 48
		padding = []float32{12, 16}
	case hasLabel && single:
		padding = []float32{22, 16, 10, 16}
	case hasLabel:
		padding = []float32{24, 16, 8, 16}
	case !single:
		padding = []float32{16, 16}
	}
	node := c.Env.Reg.Add(Node{
		Role: RoleTextField, Name: firstNonEmpty(f.Label, f.Hint), Enabled: !f.Disabled, Text: *buf,
		Value: func() string { return *buf },
		Click: func() {},
		Fill: func(text string, submit bool) {
			*buf = text
			if f.OnChange != nil {
				f.OnChange(text)
			}
			if submit && f.OnSubmit != nil {
				f.OnSubmit(text)
			}
		},
	})
	return outer.Children(func() {
		var input ui.Element
		box := ui.Row(c.Context).AlignItems(ui.Center)
		focused := st.wasFocus
		raised := focused || *buf != ""
		motion := c.Env.Theme.Motion
		labelProgress := box.AnimateTo("label", Flag(raised && hasLabel), motion.FastSpatial.Anim())
		focusProgress := box.AnimateTo("focus", Flag(focused), motion.FastEffects.Anim())

		container, content, labelColor := cs.SurfaceContainerHighest, cs.OnSurface, cs.OnSurfaceVariant
		if f.Disabled {
			container = cs.OnSurface.Alpha(0.04)
			content = cs.OnSurface.Alpha(0.38)
			labelColor = content
		}
		switch {
		case hasError:
			labelColor = cs.Error
		case focused:
			labelColor = cs.Primary
		}
		box.Padding(padding...).Gap(12).Radius(CornerMedium).Background(container).TextColor(content).
			FontSize(style.Size * scale).FixedLineHeight(style.Line * scale)
		if single {
			box.Height(height)
		} else {
			box.MinHeight(height)
		}
		box.Children(func() {
			if f.LeadingIcon != "" {
				Icon(c, f.LeadingIcon, 24, cs.OnSurfaceVariant, 0)
			}
			if single {
				input = ui.TextInputBase(c.Context, buf)
			} else {
				input = ui.TextAreaBase(c.Context, buf).Lines(lines, maxLines)
			}
			input.Grow(1).Shrink(1).Label(firstNonEmpty(f.Label, f.Hint))
			if f.Monospace {
				input.Font(MonoFamilies)
			}
			if f.Disabled {
				input.Disabled(true)
			}
			if f.Hint != "" && (!hasLabel || focused) {
				input.Placeholder(f.Hint)
			}
			if f.AutoFocus {
				input.AutoFocus()
			}
			if f.OnChange != nil {
				input.OnChange(func() { f.OnChange(*buf) })
			}
			if f.OnSubmit != nil && single {
				input.OnSubmit(func() { f.OnSubmit(*buf) })
			}
			if f.Trailing != nil {
				f.Trailing()
			}
		})
		focusedNow := input.Focused()
		if st.wasFocus && !focusedNow && f.OnBlur != nil {
			f.OnBlur(*buf)
		}
		if focusedNow != st.wasFocus {
			st.wasFocus = focusedNow
			c.Invalidate()
		}
		// A click anywhere in the field focuses the text, which only covers a
		// part of the field.
		if box.Clicked() {
			input.Focus()
		}
		// The outline: none at rest, 2 dp when focused, the error colour when
		// there is an error.
		rest := cs.Primary.Alpha(focusProgress)
		box.DrawOver(func(p *ui.Painter, r ui.Rect) {
			node.Rect = r
			radii := Uniform(CornerMedium)
			switch {
			case hasError && focusedNow:
				strokeRRect(p, r, radii, 2, cs.Error)
			case hasError:
				strokeRRect(p, r, radii, 1, cs.Error)
			default:
				strokeRRect(p, r, radii, 2*focusProgress, rest)
			}
		})
		if hasLabel {
			// The floating label.
			big, small := Type(BodyLarge), Type(BodySmall)
			label := big.WithSize(Lerp(big.Size, small.Size, labelProgress))
			centred := (height - label.Line*scale) / 2
			Line(c, f.Label, label, labelColor).Absolute().Top(Lerp(centred, 8, labelProgress)).
				Left(16 + leadingOffset(f.LeadingIcon)).Right(16).PassThrough()
		}
		if note := firstNonEmpty(f.Error, f.Helper); note != "" {
			noteColor := cs.OnSurfaceVariant
			if hasError {
				noteColor = cs.Error
			}
			Label(c, note, Type(BodySmall), noteColor).Margin(0, 16, 0, 16)
		}
	})
}

// leadingOffset is the room that the icon before the text takes.
func leadingOffset(icon string) float32 {
	if icon == "" {
		return 0
	}
	return 24 + 12
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// SearchBarSpec describes a search bar.
type SearchBarSpec struct {
	// ID identifies the bar among its siblings between builds.
	ID string
	// Hint is shown while the bar is empty.
	Hint string
	// Text is where the bar keeps its text; without it the bar keeps the text
	// itself and follows Value.
	Text *string
	// Value is the text to show when Text is nil.
	Value string
	// OnChange runs with the text whenever the user changes it.
	OnChange func(text string)
	// ClearTooltip names the clear button.
	ClearTooltip string
}

// SearchBar is a Material 3 search bar: a pill with a search icon, the text and a
// button that clears it.
func SearchBar(c Ctx, s SearchBarSpec) ui.Element {
	cs := c.Colors()
	if hasKey(s.ID) {
		c.Key(s.ID)
	}
	box := ui.Row(c.Context).AlignItems(ui.Center)
	st := ui.Local(box, "field", func() fieldState { return fieldState{text: s.Value, last: s.Value} })
	buf := st.buffer(s.Text, s.Value)
	node := c.Env.Reg.Add(Node{
		Role: RoleTextField, Name: s.Hint, Enabled: true, Text: *buf,
		Value: func() string { return *buf },
		Click: func() {},
		Fill: func(text string, _ bool) {
			*buf = text
			if s.OnChange != nil {
				s.OnChange(text)
			}
		},
	})
	scale := c.Env.Theme.TextScale
	style := Type(BodyLarge)
	focusProgress := box.AnimateTo("focus", Flag(st.wasFocus), c.Env.Theme.Motion.FastEffects.Anim())
	box.Height(56).Padding(0, 4, 0, 16).Gap(16).Radius(CornerFull).Background(cs.SurfaceContainerHigh).
		TextColor(cs.OnSurface).FontSize(style.Size * scale).FixedLineHeight(style.Line * scale)
	var input ui.Element
	box.Children(func() {
		Icon(c, "search", 24, cs.OnSurfaceVariant, 0)
		input = ui.TextInputBase(c.Context, buf).Grow(1).Shrink(1).Label(s.Hint)
		if s.Hint != "" {
			input.Placeholder(s.Hint)
		}
		if s.OnChange != nil {
			input.OnChange(func() { s.OnChange(*buf) })
		}
		if *buf != "" {
			IconButton(c, IconButtonSpec{
				ID: "clear", Icon: "close", Tooltip: s.ClearTooltip,
				OnClick: func() {
					*buf = ""
					if s.OnChange != nil {
						s.OnChange("")
					}
				},
			})
		}
	})
	focused := input.Focused()
	if focused != st.wasFocus {
		st.wasFocus = focused
		c.Invalidate()
	}
	if box.Clicked() {
		input.Focus()
	}
	ring := cs.Primary.Alpha(focusProgress)
	return box.DrawOver(func(p *ui.Painter, r ui.Rect) {
		node.Rect = r
		strokeRRect(p, r, Uniform(r.H/2), 2*focusProgress, ring)
	})
}

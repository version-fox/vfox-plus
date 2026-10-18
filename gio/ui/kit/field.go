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
	"image"

	"gioui.org/gesture"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/unit"
	"gioui.org/widget"
)

// TextField is a Material 3 filled text field, with a label that floats above
// the text, helper and error text, and optional icons.
type TextField struct {
	// ID identifies the field between frames, within its scope.
	ID string
	// Label floats above the text once the field has the focus or a text.
	Label string
	// Hint is shown in an empty field that has no label, or that has the
	// focus.
	Hint string
	// Helper is shown below the field.
	Helper string
	// Error is shown below the field instead of the helper, and makes the
	// field the colour of errors.
	Error string
	// Value is the text to show. The field keeps what the user types until the
	// value changes.
	Value string
	// OnChange runs with the text whenever the user changes it.
	OnChange func(text string)
	// OnSubmit runs with the text when the user presses Enter.
	OnSubmit func(text string)
	// OnBlur runs with the text when the field loses the focus.
	OnBlur func(text string)
	// LeadingIcon is an icon before the text.
	LeadingIcon string
	// Trailing is a widget at the end of the field, such as an icon button.
	Trailing Widget
	// Lines is the number of lines the field shows at least; 1 for a field
	// with a single line.
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

type fieldState struct {
	click     gesture.Click
	editor    widget.Editor
	lastValue string
	wasFocus  bool
	// silent says that the next change of the editor is a call of SetText: it
	// is not a change the user made, so OnChange does not hear of it.
	silent bool
	label  Anim
	focus  Anim
	init   bool
}

// setText replaces the text of the editor, which OnChange does not announce.
func (st *fieldState) setText(text string) {
	if st.editor.Text() != text {
		st.silent = true
		st.editor.SetText(text)
	}
}

// focusArea lays out the field, and gives the editor the focus when the user
// clicks anywhere in it: the editor itself only covers its text.
func (st *fieldState) focusArea(c Ctx, w Widget) layout.Dimensions {
	for {
		if _, ok := st.click.Update(c.Source); !ok {
			break
		}
		c.Execute(key.FocusCmd{Tag: &st.editor})
	}
	p := c.measure(c.Constraints, w)
	area := clip.Rect(image.Rectangle{Max: p.size()}).Push(c.Ops)
	st.click.Add(c.Ops)
	area.Pop()
	p.put(c, image.Point{})
	return p.dims
}

// FieldText returns the text that the field with the id holds, for code that reads
// a field when the user asks to save.
func FieldText(c Ctx, id string) string { return UseID[fieldState](c, id).editor.Text() }

// FieldFocused reports whether the field with the id has the keyboard focus.
func FieldFocused(c Ctx, id string) bool {
	return c.Focused(&UseID[fieldState](c, id).editor)
}

// SetFieldText replaces the text of the field with the id.
func SetFieldText(c Ctx, id, text string) {
	st := UseID[fieldState](c, id)
	st.setText(text)
	st.lastValue = text
}

// FocusField gives the field with the id the keyboard focus.
func FocusField(c Ctx, id string) { c.Execute(key.FocusCmd{Tag: &UseID[fieldState](c, id).editor}) }

// W is the field as a widget.
func (f TextField) W() Widget {
	return func(c Ctx) layout.Dimensions {
		st := UseID[fieldState](c, f.ID)
		theme := c.Env.Theme
		colors := theme.Colors
		lines := maxInt(1, f.Lines)
		maxLines := maxInt(lines, f.MaxLines)
		single := lines == 1 && maxLines == 1
		st.editor.SingleLine = single
		st.editor.Submit = single
		st.editor.ReadOnly = f.Disabled
		if !st.init {
			st.init = true
			st.setText(f.Value)
			st.lastValue = f.Value
			if f.AutoFocus {
				c.Execute(key.FocusCmd{Tag: &st.editor})
			}
		}
		if f.Value != st.lastValue {
			st.setText(f.Value)
			st.lastValue = f.Value
		}
		for {
			ev, ok := st.editor.Update(c.Context)
			if !ok {
				break
			}
			switch ev.(type) {
			case widget.ChangeEvent:
				if st.silent {
					st.silent = false
				} else if f.OnChange != nil {
					f.OnChange(st.editor.Text())
				}
			case widget.SubmitEvent:
				if f.OnSubmit != nil {
					f.OnSubmit(st.editor.Text())
				}
			}
		}
		focused := c.Focused(&st.editor)
		if st.wasFocus && !focused && f.OnBlur != nil {
			f.OnBlur(st.editor.Text())
		}
		st.wasFocus = focused
		hasText := st.editor.Len() > 0
		raised := focused || hasText
		labelProgress := st.label.F(c, boolFloat(raised && f.Label != ""), theme.Motion.FastSpatial)
		focusProgress := st.focus.F(c, boolFloat(focused), theme.Motion.FastEffects)

		inputStyle := Type(BodyLarge)
		if f.Monospace || f.Dense {
			inputStyle = Type(BodyMedium)
		}
		hasError := f.Error != ""
		container := colors.SurfaceContainerHighest
		content := colors.OnSurface
		labelColor := colors.OnSurfaceVariant
		if f.Disabled {
			container = Alpha(colors.OnSurface, 0.04)
			content = Alpha(colors.OnSurface, 0.38)
			labelColor = content
		}
		if hasError {
			labelColor = colors.Error
		} else if focused {
			labelColor = colors.Primary
		}

		field := func(c Ctx) layout.Dimensions {
			width := c.Constraints.Max.X
			pad := c.DP(16)
			leading := 0
			var leadingP placed
			if f.LeadingIcon != "" {
				leadingP = c.measure(loose(image.Pt(c.DP(24), c.DP(24))), Icon(f.LeadingIcon, 24, colors.OnSurfaceVariant, 0))
				leading = leadingP.size().X + c.DP(12)
			}
			trailing := 0
			var trailingP placed
			if f.Trailing != nil {
				trailingP = c.measure(loose(image.Pt(width/2, c.DP(56))), f.Trailing)
				trailing = trailingP.size().X + c.DP(4)
			}
			inner := maxInt(0, width-2*pad-leading-trailing)
			lineHeight := c.SP(inputStyle.Line)
			textHeight := lineHeight * lines
			top := c.DP(8)
			labelHeight := 0
			if f.Label != "" && !f.Dense {
				labelHeight = c.DP(16)
			}
			height := maxInt(c.DP(56), top+labelHeight+lineHeight*lines+c.DP(8))
			if f.Dense {
				height = maxInt(c.DP(48), c.DP(12)*2+lineHeight*lines)
			}
			if single && !f.Dense {
				height = c.DP(56)
			}
			if !single {
				height = maxInt(height, top*2+labelHeight+textHeight)
			}
			size := image.Pt(width, height)
			rect := image.Rectangle{Max: size}
			r := c.Round(CornerMedium)
			FillRRect(c.Ops, rect, r, container)
			// The outline: none at rest, 2 dp when focused, the error colour
			// otherwise when there is an error.
			switch {
			case hasError && focused:
				StrokeRRect(c.Ops, rect, r, float32(c.DP(2)), colors.Error)
			case hasError:
				StrokeRRect(c.Ops, rect, r, float32(c.DP(1)), colors.Error)
			default:
				StrokeRRect(c.Ops, rect, r, float32(c.DP(2))*focusProgress, Alpha(colors.Primary, focusProgress))
			}
			if f.LeadingIcon != "" {
				leadingP.put(c, image.Pt(pad, (height-leadingP.size().Y)/2))
			}
			if f.Trailing != nil {
				trailingP.put(c, image.Pt(width-pad/2-trailingP.size().X, (height-trailingP.size().Y)/2))
			}

			// The text, below the floating label.
			textTop := (height - textHeight) / 2
			if f.Label != "" && !f.Dense {
				textTop = top + labelHeight + (height-top-labelHeight-c.DP(8)-textHeight)/2
				if single {
					textTop = height - c.DP(8) - lineHeight - c.DP(2)
				}
			}
			textX := pad + leading
			textArea := image.Rect(textX, textTop, textX+inner, textTop+maxInt(textHeight, lineHeight*maxLines))
			if !single {
				textArea.Max.Y = minInt(height-c.DP(8), textTop+lineHeight*maxLines)
			}
			// The hint, while the field is empty and shows no label.
			if !hasText && (f.Label == "" || focused) && f.Hint != "" {
				hint := c.measure(loose(image.Pt(inner, textHeight)), Line(f.Hint, inputStyle, Alpha(colors.OnSurfaceVariant, 0.7), optionalMono(f.Monospace)))
				hint.put(c, image.Pt(textX, textTop+(lineHeight-hint.size().Y)/2))
			}
			material := colorOp(c.Ops, content)
			selection := colorOp(c.Ops, Alpha(colors.Primary, 0.3))
			face := theme.Fonts.UI(inputStyle.Weight)
			if f.Monospace {
				face = theme.Fonts.Mono()
			}
			st.editor.LineHeight = unit.Sp(inputStyle.Line)
			st.editor.LineHeightScale = 1
			macroCs := layout.Constraints{
				Min: image.Pt(inner, textArea.Dy()),
				Max: image.Pt(inner, textArea.Dy()),
			}
			if single {
				macroCs.Min.Y = lineHeight
				macroCs.Max.Y = lineHeight
			}
			editorP := c.measure(macroCs, func(c Ctx) layout.Dimensions {
				return st.editor.Layout(c.Context, theme.Shaper(), face, unit.Sp(inputStyle.Size), material, selection)
			})
			editorP.put(c, textArea.Min)
			// The floating label.
			if f.Label != "" && !f.Dense {
				big, small := Type(BodyLarge), Type(BodySmall)
				style := big.WithSize(Lerp(big.Size, small.Size, labelProgress))
				labelP := c.measure(loose(image.Pt(inner, c.DP(32))), Line(f.Label, style, labelColor))
				centered := (height - labelP.size().Y) / 2
				raisedY := top
				labelP.put(c, image.Pt(textX, int(Lerp(float32(centered), float32(raisedY), labelProgress))))
			}
			focusRing(c, interaction{keyboard: false}, rect, r)
			return layout.Dimensions{Size: size}
		}

		var children []FlexChild
		children = append(children, Rigid(func(c Ctx) layout.Dimensions {
			cs := c.Constraints
			c.Constraints = layout.Constraints{Min: image.Pt(cs.Max.X, 0), Max: cs.Max}
			// A click anywhere in the field focuses the editor: the editor
			// itself only covers the text.
			dims := st.focusArea(c, field)
			c.Env.Reg.Add(Node{
				Key: c.key(f.ID), Role: RoleTextField, Name: firstNonEmpty(f.Label, f.Hint), Text: st.editor.Text(),
				Rect: image.Rectangle{Max: dims.Size}, Enabled: !f.Disabled, Focused: focused,
				Click: func() { c.Execute(key.FocusCmd{Tag: &st.editor}); c.Env.Services.Invalidate() },
				Fill: func(text string, submit bool) {
					st.setText(text)
					st.lastValue = f.Value
					if f.OnChange != nil {
						f.OnChange(text)
					}
					if submit && f.OnSubmit != nil {
						f.OnSubmit(text)
					}
					c.Env.Services.Invalidate()
				},
			})
			return dims
		}))
		if note := firstNonEmpty(f.Error, f.Helper); note != "" {
			style := Type(BodySmall)
			col := colors.OnSurfaceVariant
			if hasError {
				col = colors.Error
			}
			children = append(children, Rigid(Pad(Inset{16, 4, 16, 0}, Label(note, style, col))))
		}
		return Column(FlexOpts{Cross: CrossStretch, Min: true}, children...)(c)
	}
}

func optionalMono(mono bool) LabelOption {
	if mono {
		return Monospace()
	}
	return func(*labelOptions) {}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// SearchBar is a Material 3 search bar: a pill with a search icon, the text
// and a button that clears it.
type SearchBar struct {
	// ID identifies the bar between frames, within its scope.
	ID string
	// Hint is shown while the bar is empty.
	Hint string
	// Value is the text to show.
	Value string
	// OnChange runs with the text whenever the user changes it.
	OnChange func(text string)
	// ClearTooltip names the clear button.
	ClearTooltip string
}

// W is the bar as a widget.
func (s SearchBar) W() Widget {
	return func(c Ctx) layout.Dimensions {
		st := UseID[fieldState](c, s.ID)
		theme := c.Env.Theme
		colors := theme.Colors
		if !st.init {
			st.init = true
			st.setText(s.Value)
			st.lastValue = s.Value
		}
		if s.Value != st.lastValue {
			st.setText(s.Value)
			st.lastValue = s.Value
		}
		st.editor.SingleLine = true
		for {
			ev, ok := st.editor.Update(c.Context)
			if !ok {
				break
			}
			if _, ok := ev.(widget.ChangeEvent); ok {
				if st.silent {
					st.silent = false
				} else if s.OnChange != nil {
					s.OnChange(st.editor.Text())
				}
			}
		}
		focused := c.Focused(&st.editor)
		hasText := st.editor.Len() > 0
		return func(c Ctx) layout.Dimensions {
			width := c.Constraints.Max.X
			height := c.DP(56)
			size := image.Pt(width, height)
			rect := image.Rectangle{Max: size}
			r := Uniform(float32(height) / 2)
			FillRRect(c.Ops, rect, r, colors.SurfaceContainerHigh)
			if focused {
				StrokeRRect(c.Ops, rect, r, float32(c.DP(2)), colors.Primary)
			}
			pad := c.DP(16)
			icon := c.measure(loose(image.Pt(c.DP(24), c.DP(24))), Icon("search", 24, colors.OnSurfaceVariant, 0))
			icon.put(c, image.Pt(pad, (height-icon.size().Y)/2))
			textX := pad + c.DP(24) + c.DP(16)
			trailing := 0
			if hasText {
				clear := c.measure(loose(image.Pt(c.DP(48), c.DP(48))), IconButton{
					ID: s.ID + ".clear", Icon: "close", Tooltip: s.ClearTooltip,
					OnClick: func() {
						st.setText("")
						if s.OnChange != nil {
							s.OnChange("")
						}
					},
				}.W())
				trailing = clear.size().X
				clear.put(c, image.Pt(width-clear.size().X-c.DP(4), (height-clear.size().Y)/2))
			}
			inner := maxInt(0, width-textX-pad-trailing)
			lineHeight := c.SP(Type(BodyLarge).Line)
			textTop := (height - lineHeight) / 2
			if !hasText && s.Hint != "" {
				hint := c.measure(loose(image.Pt(inner, lineHeight)), Line(s.Hint, Type(BodyLarge), colors.OnSurfaceVariant))
				hint.put(c, image.Pt(textX, textTop+(lineHeight-hint.size().Y)/2))
			}
			material := colorOp(c.Ops, colors.OnSurface)
			selection := colorOp(c.Ops, Alpha(colors.Primary, 0.3))
			st.editor.LineHeight = unit.Sp(Type(BodyLarge).Line)
			st.editor.LineHeightScale = 1
			ed := c.measure(layout.Constraints{Min: image.Pt(inner, lineHeight), Max: image.Pt(inner, lineHeight)}, func(c Ctx) layout.Dimensions {
				return st.editor.Layout(c.Context, theme.Shaper(), theme.Fonts.UI(400), unit.Sp(Type(BodyLarge).Size), material, selection)
			})
			ed.put(c, image.Pt(textX, textTop))
			c.Env.Reg.Add(Node{
				Key: c.key(s.ID), Role: RoleTextField, Name: s.Hint, Text: st.editor.Text(),
				Rect: rect, Enabled: true, Focused: focused,
				Click: func() { c.Execute(key.FocusCmd{Tag: &st.editor}); c.Env.Services.Invalidate() },
				Fill: func(text string, _ bool) {
					st.setText(text)
					if s.OnChange != nil {
						s.OnChange(text)
					}
					c.Env.Services.Invalidate()
				},
			})
			return layout.Dimensions{Size: size}
		}(c)
	}
}

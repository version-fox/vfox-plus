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

package webheap

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// A scene takes the web interface where a screenshot for the documentation is
// wanted, beyond the five pages that -tab opens: the details of a plugin with its
// chosen distribution, a dialog that is open, a filter chip that is chosen. Each
// step waits for the page to show something and clicks it, the way a person
// does; the expressions run in the page.

// step is one move of a scene.
type step struct {
	// click is a JavaScript expression for the element to click. The step waits
	// until the page has the element.
	click string
	// wait is a JavaScript expression that has to be true before the scene goes on.
	wait string
}

// pick clicks the element that the selector names, when it is there.
func pick(selector string) step {
	return step{click: fmt.Sprintf("document.querySelector(%q)", selector)}
}

// exists waits until the page has an element that the selector names.
func exists(selector string) step {
	return step{wait: fmt.Sprintf("!!document.querySelector(%q)", selector)}
}

// until waits for a JavaScript expression.
func until(expression string) step { return step{wait: expression} }

// concat joins the steps of several parts of a scene.
func concat(parts ...[]step) []step {
	var all []step
	for _, part := range parts {
		all = append(all, part...)
	}
	return all
}

// fitsTheWindow waits until the page is no wider than the window, and what each of the two panes of the SDKs holds is
// no wider than the pane. The tabs of a plugin that did not fit made the column of the details grow, and with it the
// page; and a pane can be overrun by its content while the page still fits, when the overflow stays inside the window.
func fitsTheWindow() step {
	return until(`(() => {
		const fits = (element) => !element || element.scrollWidth <= element.clientWidth + 1;
		return fits(document.documentElement) && ["#sdkGrid", ".pane-detail"].every((selector) => fits(document.querySelector(selector)));
	})()`)
}

// plugin opens the details of the plugin that the list of the SDKs shows. In a wide window the details of another plugin
// are open before the click, so the step waits for the ones of this plugin.
func plugin(name string) []step {
	return []step{
		pick(fmt.Sprintf(`[data-sdk="%s"]`, name)),
		exists(fmt.Sprintf(`[data-sdk="%s"][aria-current="true"]`, name)),
		until(fmt.Sprintf(`document.querySelector("#sdkDetail .t-title-l code")?.textContent === %q`, name)),
		exists("#detailTabs"), fitsTheWindow(),
	}
}

// moreWays picks one of the ways to add an SDK that are in the menu next to the choice of the distribution.
func moreWays(item string) []step {
	return []step{pick("#addMore"), pick(fmt.Sprintf(`#menu [data-item="%s"]`, item))}
}

// tab opens a tab of the details: installed, available or options.
func tab(name string) []step {
	return []step{pick(fmt.Sprintf(`#detailTabs [data-tab="%s"]`, name))}
}

// scrollTo moves the page so that the element that the selector names is at the top of it, under the
// bar of the page, which a page longer than the window needs for a picture of what is further down.
func scrollTo(selector string) step {
	return until(fmt.Sprintf("(() => { const element = document.querySelector(%q); if (!element) return false; "+
		"element.style.scrollMarginTop = '72px'; element.scrollIntoView({block: 'start'}); return true; })()", selector))
}

// connections opens the settings and waits for the card of the connections to be drawn.
var connections = []step{pick(`[data-tab="settings"]`), exists("#webList .rowitem")}

// versions waits for the list of the available versions to be drawn.
var versions = []step{exists("#vList .vrow")}

// scenes are the pages that -scene can open. The plugins are the ones that the
// fixtures of `vfox` install (VFOX_PLUS_FAKE_INVENTORY=profiles).
var scenes = map[string][]step{}

func init() {
	javaAvailable := concat(plugin("java"), tab("available"), versions)
	options := concat(plugin("nodejs"), tab("options"), []step{exists("[data-var]")})
	scenes["java-available"] = javaAvailable
	scenes["java-choose"] = concat(javaAvailable, []step{pick("#chooseBtn"), exists("#chList .item")})
	scenes["java-temurin"] = concat(javaAvailable, []step{
		pick("#chooseBtn"), pick(`#chList [data-v="tem"]`),
		// The button names the choice and the list is the one of Temurin.
		until(`document.querySelector("#chooseBtn").textContent.includes("Temurin") && !!document.querySelector("#vList .vrow")`),
		pick(`#vChips [data-v="LTS"]`), exists(`#vChips [data-v="LTS"][aria-pressed="true"]`),
	})
	scenes["java-install-name"] = concat(javaAvailable, []step{
		pick("#addMore"), pick(`#menu [data-item="by-name"]`), pick(`#nameChips [data-name="21-tem"]`),
		until(`document.getElementById("nameMeaning").textContent !== ""`),
	})
	scenes["flutter-chips"] = concat(plugin("flutter"), tab("available"), versions, []step{
		pick(`#vChips [data-v="stable"]`), exists(`#vChips [data-v="stable"][aria-pressed="true"]`),
	})
	// The card of the connections, and the code that it shows to pair a phone: they need the application to
	// allow other devices (the setting webRemote) and to have an address to give.
	scenes["connections"] = concat(connections, []step{scrollTo("#webCard")})
	scenes["connections-pair"] = concat(connections, []step{pick("#webPair"), exists("#pairBox .qr"), scrollTo("#pairBox")})
	// The home page, once its lists have come, is still at its own address: a page that is not shown does not move the
	// address to a plugin, which a reload would take for the page that was asked for.
	scenes["home"] = []step{
		until(`document.querySelectorAll("#homeActive .item").length > 0`),
		until(`!location.hash.startsWith("#/sdk")`),
		fitsTheWindow(),
	}
	scenes["options"] = options
	scenes["options-mirror"] = concat(options, []step{
		pick(`[data-var="VFOX_NODEJS_MIRROR"]`), pick(`#varChips [data-value="https://npmmirror.com/mirrors/node"]`),
	})
	// What vfox 1.1.0 adds to a plugin: the dialog that links an SDK that is already on the machine, the
	// aliases of a plugin, and the dialog that names one.
	scenes["link-manual"] = concat(javaAvailable, moreWays("link-manual"), []step{exists("#linkVer")})
	scenes["options-aliases"] = concat(options, []step{exists("#aliasAdd"), scrollTo("#aliasHead")})
	scenes["alias-set"] = concat(options, []step{pick("#aliasAdd"), exists("#aliasIn")})
	// The rows that have a choice of a few buttons: the version that the version files of a plugin name, and the
	// languages. The choice did not fit beside its label on a phone, and covered it. The last step of each waits for the
	// choices to be under their labels, at the same edge, so that a layout that goes back to that does not make a picture.
	scenes["options-legacy"] = concat(options, []step{exists("#legacyStrategy"), scrollTo("#legacyHead"), underTheirLabels("legacyStrategy")})
	scenes["settings-languages"] = []step{
		pick(`[data-tab="settings"]`), exists("#langSeg"), scrollTo("#l-lang"), underTheirLabels("langSeg", "pageLangSeg"),
	}
	// A page that does not fit its window is not one to take a picture of, whatever the scene shows.
	for name, steps := range scenes {
		scenes[name] = concat(steps, []step{fitsTheWindow()})
	}
}

// underTheirLabels waits until each of the choices that the identifiers name is under the label that comes before it,
// and starts at the same edge. Beside its label a choice starts above the end of the label, which is what it did when
// it covered it.
func underTheirLabels(ids ...string) step {
	list := make([]string, len(ids))
	for i, id := range ids {
		list[i] = fmt.Sprintf("%q", id)
	}
	return until(fmt.Sprintf(`[%s].every((id) => {
		const choice = document.getElementById(id);
		if (!choice) return false;
		const at = choice.getBoundingClientRect(), label = choice.previousElementSibling.getBoundingClientRect();
		return at.top >= label.bottom - 1 && Math.abs(at.left - label.left) < 1;
	})`, strings.Join(list, ", ")))
}

// sceneNames are the names that -scene takes, in order.
func sceneNames() []string {
	names := make([]string, 0, len(scenes))
	for name := range scenes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// clickWhenThere is the expression that clicks the element that expression names,
// and is true once it did.
func clickWhenThere(expression string) string {
	return fmt.Sprintf("(() => { const element = %s; if (!element) return false; element.click(); return true; })()", expression)
}

// play runs the steps of a scene, with a pause after each so that the page can
// finish moving before the next click or the picture.
func (s *session) play(ctx context.Context, name string) error {
	for i, move := range scenes[name] {
		expression, what := move.wait, "the page of the scene "+name
		if move.click != "" {
			expression, what = clickWhenThere(move.click), "an element of the scene "+name
		}
		if err := s.waitFor(ctx, expression, fmt.Sprintf("%s (step %d: %s)", what, i+1, strings.Join(strings.Fields(expression), " "))); err != nil {
			return err
		}
		if err := sleep(ctx, 300*time.Millisecond); err != nil {
			return err
		}
	}
	return nil
}

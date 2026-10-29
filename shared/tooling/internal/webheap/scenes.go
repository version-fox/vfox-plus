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

// plugin opens the details of the plugin that the list of the SDKs shows.
func plugin(name string) []step {
	return []step{pick(fmt.Sprintf(`[data-sdk="%s"]`, name)), exists("#detailTabs")}
}

// tab opens a tab of the details: installed, available or options.
func tab(name string) []step {
	return []step{pick(fmt.Sprintf(`#detailTabs [data-tab="%s"]`, name))}
}

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
		pick("#byNameBtn"), pick(`#nameChips [data-name="21-tem"]`),
		until(`document.getElementById("nameMeaning").textContent !== ""`),
	})
	scenes["flutter-chips"] = concat(plugin("flutter"), tab("available"), versions, []step{
		pick(`#vChips [data-v="stable"]`), exists(`#vChips [data-v="stable"][aria-pressed="true"]`),
	})
	scenes["options"] = options
	scenes["options-mirror"] = concat(options, []step{
		pick(`[data-var="VFOX_NODEJS_MIRROR"]`), pick(`#varChips [data-value="https://npmmirror.com/mirrors/node"]`),
	})
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

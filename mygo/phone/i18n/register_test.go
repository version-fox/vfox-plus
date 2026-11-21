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

package i18n

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// mustPanic runs fn, and fails the test unless it panics.
func mustPanic(t *testing.T, what string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s did not panic", what)
		}
	}()
	fn()
}

func TestRegisteredMessagesAreInBothLanguages(t *testing.T) {
	if len(registered) == 0 {
		t.Fatal("no message was registered")
	}
	for key := range registered {
		for _, lang := range []Language{English, Chinese} {
			if TextIn(lang, key) == "" || TextIn(lang, key) == key {
				t.Errorf("the registered message %q has no %s text", key, lang)
			}
		}
	}
}

func TestRegisterRefusesAKeyThatHasAMessage(t *testing.T) {
	mustPanic(t, "a registration of the web's key back", func() {
		Register(map[string][2]string{"back": {"Back", "返回"}})
	})
	mustPanic(t, "a registration of a key registered already", func() {
		Register(map[string][2]string{"errOther": {"Again.", "再次。"}})
	})
}

func TestRegisterRefusesABlankText(t *testing.T) {
	mustPanic(t, "a blank English text", func() {
		Register(map[string][2]string{"blankEnglishTest": {"", "文本"}})
	})
	mustPanic(t, "a blank Chinese text", func() {
		Register(map[string][2]string{"blankChineseTest": {"Text", ""}})
	})
}

func TestTextOfAKeyThatIsNotRegisteredIsTheKey(t *testing.T) {
	if got := TextIn(Chinese, "no-such-key"); got != "no-such-key" {
		t.Errorf("an unknown key reads %q, want the key itself", got)
	}
}

// TestTheCodeUsesOnlyRegisteredKeys reads the code of the phone app with the
// parser, and checks that every message key written as a literal in a call of
// i18n.Text or i18n.TextIn is in the catalogs. A key that is not in them would show
// as its own name, which no user should see. Keys computed at run time are not
// checked here; the tests of the screens cover them.
func TestTheCodeUsesOnlyRegisteredKeys(t *testing.T) {
	root := ".."
	checked := 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case "i18n", "dist", ".mygo", "resources", "testdata":
				if path != root {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := selector.X.(*ast.Ident)
			if !ok || pkg.Name != "i18n" {
				return true
			}
			keyArg := -1
			switch selector.Sel.Name {
			case "Text":
				keyArg = 0
			case "TextIn":
				keyArg = 1
			}
			if keyArg < 0 || len(call.Args) <= keyArg {
				return true
			}
			lit, ok := call.Args[keyArg].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			key, err := strconv.Unquote(lit.Value)
			if err != nil {
				t.Errorf("%s: the key %s is not a string", path, lit.Value)
				return true
			}
			checked++
			if _, ok := catalogs[English][key]; !ok {
				t.Errorf("%s: the key %q is used but not registered", path, key)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked == 0 {
		t.Fatal("no message key was found in the code: the scan is looking in the wrong place")
	}
}

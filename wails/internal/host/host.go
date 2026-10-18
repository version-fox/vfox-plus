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

// Package host answers the host protocol of the web front end
// (web/src/hosts/protocol.ts): it spawns vfox, reads and writes the few files
// the application needs and talks to the desktop. The user interface, the vfox
// client and the task queue are the web front end of ../web.
package host

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// AppID is the application ID, also the name of the desktop entry.
const AppID = "dev.vfox.VfoxPlus"

// ProcessEventName is the event that carries the output and the exit of the
// processes started by the front end.
const ProcessEventName = "vfox-plus:process"

// Desktop is what the host needs from the application shell.
type Desktop interface {
	// OpenURL opens url in the default browser.
	OpenURL(url string) error
	// PickDirectory lets the user choose a directory; "" if they cancel.
	PickDirectory(initial string) (string, error)
	// PickFile lets the user choose a file, optionally of a type; "" if they cancel.
	PickFile(initial, label string, extensions []string) (string, error)
	// CopyText copies text to the clipboard.
	CopyText(text string) error
	// Emit sends a process event to the front end, in order.
	Emit(event Event)
	// Exit ends the application with code.
	Exit(code int)
}

// Request is a request of the front end.
type Request struct {
	ID     int64             `json:"id"`
	Method string            `json:"method"`
	Params []json.RawMessage `json:"params"`
}

// Service is the service that Wails binds for the front end. Its only exported
// method, Call, is reached as
// "github.com/version-fox/vfox-plus/wails/internal/host.Service.Call".
type Service struct {
	desktop   Desktop
	processes *Processes
}

// NewService creates the service.
func NewService(desktop Desktop) *Service {
	return &Service{desktop: desktop, processes: NewProcesses()}
}

// Call answers a request of the host protocol with {id, result} or {id,
// error: {kind, message}}, where the kind "start" means that an executable
// could not be started.
func (s *Service) Call(request Request) map[string]any {
	result, err := s.dispatch(request.Method, params(request.Params))
	if err != nil {
		kind := "error"
		var start *StartError
		if errors.As(err, &start) {
			kind = "start"
		}
		return map[string]any{"id": request.ID, "error": map[string]string{"kind": kind, "message": err.Error()}}
	}
	return map[string]any{"id": request.ID, "result": result}
}

// Shutdown terminates the processes that still run.
func (s *Service) Shutdown() {
	s.processes.KillAll()
}

// params are the parameters of a request, read by position.
type params []json.RawMessage

func (p params) decode(index int, target any) error {
	if index >= len(p) {
		return fmt.Errorf("parameter %d is missing", index)
	}
	if err := json.Unmarshal(p[index], target); err != nil {
		return fmt.Errorf("parameter %d: %w", index, err)
	}
	return nil
}

func (p params) string(index int) (string, error) {
	var value string
	err := p.decode(index, &value)
	return value, err
}

// optionalString is "" for a missing or null parameter.
func (p params) optionalString(index int) string {
	var value *string
	if p.decode(index, &value) != nil || value == nil {
		return ""
	}
	return *value
}

func (p params) launch() (Launch, error) {
	var launch Launch
	if err := p.decode(0, &launch.Executable); err != nil {
		return launch, err
	}
	if err := p.decode(1, &launch.Arguments); err != nil {
		return launch, err
	}
	if err := p.decode(2, &launch.WorkingDirectory); err != nil {
		return launch, err
	}
	err := p.decode(3, &launch.Environment)
	return launch, err
}

func (s *Service) dispatch(method string, p params) (any, error) {
	switch method {
	case "environment":
		return environment(), nil
	case "run":
		launch, err := p.launch()
		if err != nil {
			return nil, err
		}
		return Run(launch)
	case "start":
		launch, err := p.launch()
		if err != nil {
			return nil, err
		}
		return s.processes.Start(launch, s.desktop.Emit)
	case "kill":
		var id int
		if err := p.decode(0, &id); err != nil {
			return nil, err
		}
		s.processes.Kill(id)
		return nil, nil
	case "exists", "isDirectory", "isExecutable":
		path, err := p.string(0)
		if err != nil {
			return nil, err
		}
		return map[string]func(string) bool{"exists": Exists, "isDirectory": IsDirectory, "isExecutable": IsExecutable}[method](path), nil
	case "readTextFile":
		path, err := p.string(0)
		if err != nil {
			return nil, err
		}
		text, err := ReadTextFile(path)
		if text == nil {
			return nil, err
		}
		return *text, err
	case "writeTextFile":
		path, err := p.string(0)
		if err != nil {
			return nil, err
		}
		text, err := p.string(1)
		if err != nil {
			return nil, err
		}
		return nil, WriteTextFile(path, text)
	case "createDirectory", "remove":
		path, err := p.string(0)
		if err != nil {
			return nil, err
		}
		if method == "remove" {
			return nil, Remove(path)
		}
		return nil, CreateDirectory(path)
	case "createTemporaryDirectory":
		parent, err := p.string(0)
		if err != nil {
			return nil, err
		}
		prefix, err := p.string(1)
		if err != nil {
			return nil, err
		}
		return CreateTemporaryDirectory(parent, prefix)
	case "copyFile":
		from, err := p.string(0)
		if err != nil {
			return nil, err
		}
		to, err := p.string(1)
		if err != nil {
			return nil, err
		}
		return nil, CopyFile(from, to)
	case "systemTemporaryDirectory":
		return SystemTemporaryDirectory(), nil
	case "accentColor":
		if color, ok := AccentColor(); ok {
			return color, nil
		}
		return nil, nil
	case "openUrl":
		url, err := p.string(0)
		if err != nil {
			return nil, err
		}
		return nil, s.desktop.OpenURL(url)
	case "pickDirectory":
		return orNull(s.desktop.PickDirectory(p.optionalString(0)))
	case "pickFile":
		var extensions []string
		_ = p.decode(2, &extensions)
		return orNull(s.desktop.PickFile(p.optionalString(0), p.optionalString(1), extensions))
	case "copyText":
		text, err := p.string(0)
		if err != nil {
			return nil, err
		}
		return nil, s.desktop.CopyText(text)
	case "exit":
		var code int
		if err := p.decode(0, &code); err != nil {
			code = 1
		}
		s.desktop.Exit(code)
		return nil, nil
	case "log":
		message, err := p.string(0)
		if err != nil {
			return nil, err
		}
		_, err = fmt.Fprintln(os.Stdout, message)
		return nil, err
	default:
		return nil, fmt.Errorf("unknown method %s", method)
	}
}

// orNull turns the "" of a cancelled chooser into null.
func orNull(path string, err error) (any, error) {
	if err != nil || path == "" {
		return nil, err
	}
	return path, nil
}

// environment is the environment of the application process.
func environment() map[string]string {
	variables := map[string]string{}
	for _, entry := range os.Environ() {
		if key, value, ok := strings.Cut(entry, "="); ok {
			variables[key] = value
		}
	}
	return variables
}

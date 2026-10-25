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

package pair

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	core "github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/core/vfox"
	"github.com/version-fox/vfox-plus/mygo/state"
)

// loadTimeout bounds how long the web API waits for vfox. The desktop
// interface waits indefinitely; a browser fetch must not hang forever when
// vfox or the network stalls.
const loadTimeout = 2 * time.Minute

// awaitLoad waits for done or the timeout.
func awaitLoad(done <-chan struct{}) bool {
	timer := time.NewTimer(loadTimeout)
	defer timer.Stop()
	select {
	case <-done:
		return true
	case <-timer.C:
		return false
	}
}

// VfoxDTO is the availability of vfox for the web interface.
type VfoxDTO struct {
	Availability string `json:"availability"`
	Version      string `json:"version,omitempty"`
	Error        string `json:"error,omitempty"`
}

// TaskDTO is one state-changing task for the web interface.
type TaskDTO struct {
	ID           int      `json:"id"`
	Operation    string   `json:"operation"`
	Sdk          string   `json:"sdk,omitempty"`
	Version      string   `json:"version,omitempty"`
	CommandLine  string   `json:"commandLine"`
	Status       string   `json:"status"`
	Progress     *float64 `json:"progress,omitempty"`
	CreatedAt    string   `json:"createdAt"`
	StartedAt    string   `json:"startedAt,omitempty"`
	FinishedAt   string   `json:"finishedAt,omitempty"`
	ExitCode     *int     `json:"exitCode,omitempty"`
	ErrorMessage string   `json:"errorMessage,omitempty"`
	LastLine     string   `json:"lastLine,omitempty"`
	Log          []string `json:"log"`
	Dropped      int      `json:"droppedLines"`
}

// taskDTO converts task, keeping the newest tailLines log lines.
func taskDTO(task *state.VfoxTask, tailLines int) TaskDTO {
	dto := TaskDTO{
		ID:          task.ID,
		Operation:   string(task.Command.Operation),
		Sdk:         task.Command.Sdk,
		Version:     task.Command.Version,
		CommandLine: task.Command.CommandLine(),
		Status:      string(task.Status()),
		CreatedAt:   task.CreatedAt.Format(time.RFC3339),
		Dropped:     task.DroppedLines(),
	}
	if progress, ok := task.Progress(); ok {
		dto.Progress = &progress
	}
	if at, ok := task.StartedAt(); ok {
		dto.StartedAt = at.Format(time.RFC3339)
	}
	if at, ok := task.FinishedAt(); ok {
		dto.FinishedAt = at.Format(time.RFC3339)
	}
	if code, ok := task.ExitCode(); ok {
		dto.ExitCode = &code
	}
	if message, ok := task.ErrorMessage(); ok {
		dto.ErrorMessage = message
	}
	if last, ok := task.LastLine(); ok {
		dto.LastLine = last
	}
	log := task.Log()
	if tailLines > 0 && len(log) > tailLines {
		log = log[len(log)-tailLines:]
	}
	dto.Log = log
	if dto.Log == nil {
		dto.Log = []string{}
	}
	return dto
}

// vfoxDTO reports the availability of vfox.
func vfoxDTO(model *state.AppModel) VfoxDTO {
	dto := VfoxDTO{Availability: string(model.Vfox.Availability())}
	if version, ok := model.Vfox.Version(); ok {
		dto.Version = version
	}
	if err := model.Vfox.Error(); err != nil {
		dto.Error = err.Error()
	}
	return dto
}

// serveInventory answers the installed SDKs, the registry and the vfox
// configuration in one fetch. Each section loads on demand and carries its
// own loading and error state, like the desktop interface.
func (s *Server) serveInventory(writer http.ResponseWriter, request *http.Request) {
	if !s.requireSession(writer, request) {
		return
	}
	model := s.requireModel(writer)
	if model == nil {
		return
	}
	if request.Method != http.MethodGet {
		http.Error(writer, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}
	installedDone := model.Installed.EnsureLoaded()
	registryDone := model.Registry.EnsureLoaded()
	configDone := model.VfoxConfig.EnsureLoaded()
	ok := awaitLoad(installedDone) && awaitLoad(registryDone) && awaitLoad(configDone)
	installed, _ := model.Installed.Value()
	installedErr := model.Installed.Error()
	registry, _ := model.Registry.Value()
	registryErr := model.Registry.Error()
	config, _ := model.VfoxConfig.Value()
	configErr := model.VfoxConfig.Error()
	type section[T any] struct {
		Loading bool   `json:"loading"`
		Error   string `json:"error,omitempty"`
		Value   T      `json:"value"`
	}
	errText := func(err error) string {
		if err == nil {
			return ""
		}
		return err.Error()
	}
	writeJSON(writer, map[string]interface{}{
		"vfox":      vfoxDTO(model),
		"installed": section[[]core.InstalledSdk]{Loading: model.Installed.IsLoading(), Error: errText(installedErr), Value: orEmpty(installed)},
		"registry":  section[[]core.AvailablePlugin]{Loading: model.Registry.IsLoading(), Error: errText(registryErr), Value: orEmpty(registry)},
		"config":    section[[]core.ConfigEntry]{Loading: model.VfoxConfig.IsLoading(), Error: errText(configErr), Value: orEmpty(config)},
		"complete":  ok,
	})
}

func orEmpty[T any](value []T) []T {
	if value == nil {
		return []T{}
	}
	return value
}

// serveSdks answers plugin metadata and installable versions:
// /api/sdks/<name>/info and /api/sdks/<name>/versions.
func (s *Server) serveSdks(writer http.ResponseWriter, request *http.Request) {
	if !s.requireSession(writer, request) {
		return
	}
	model := s.requireModel(writer)
	if model == nil {
		return
	}
	if request.Method != http.MethodGet {
		http.Error(writer, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}
	rest := strings.TrimPrefix(request.URL.Path, "/api/sdks/")
	name, kind, _ := strings.Cut(rest, "/")
	name = strings.TrimSpace(name)
	if name == "" || strings.Contains(name, "/") {
		http.Error(writer, "No such SDK.", http.StatusNotFound)
		return
	}
	detail := model.SdkDetail(name)
	defer detail.Dispose()
	switch kind {
	case "info":
		if !awaitLoad(detail.Info.EnsureLoaded()) {
			http.Error(writer, "Loading the plugin metadata timed out.", http.StatusGatewayTimeout)
			return
		}
		info, ok := detail.Info.Value()
		if !ok {
			if err := detail.Info.Error(); err != nil {
				writeVfoxError(writer, err)
				return
			}
			http.Error(writer, "No such plugin.", http.StatusNotFound)
			return
		}
		writeJSON(writer, info)
	case "versions":
		if !awaitLoad(detail.Versions.EnsureLoaded()) {
			http.Error(writer, "Searching versions timed out.", http.StatusGatewayTimeout)
			return
		}
		versions, _ := detail.Versions.Value()
		if err := detail.Versions.Error(); err != nil {
			writeVfoxError(writer, err)
			return
		}
		writeJSON(writer, orEmpty(versions))
	default:
		http.Error(writer, "No such endpoint.", http.StatusNotFound)
	}
}

// taskRequest starts a state-changing task from the web interface. Project
// scopes are rejected: the browser cannot name a directory on the machine
// that runs vfox, so only the global scope is offered there.
type taskRequest struct {
	Operation string `json:"operation"`
	Sdk       string `json:"sdk"`
	Version   string `json:"version"`
	Scope     string `json:"scope"`
	Source    string `json:"source"`
	Alias     string `json:"alias"`
	Name      string `json:"name"`
}

// serveTasks lists tasks and starts new ones.
func (s *Server) serveTasks(writer http.ResponseWriter, request *http.Request) {
	if !s.requireSession(writer, request) {
		return
	}
	model := s.requireModel(writer)
	if model == nil {
		return
	}
	switch request.Method {
	case http.MethodGet:
		tasks := make([]TaskDTO, 0)
		for _, task := range model.Tasks.Tasks() {
			tasks = append(tasks, taskDTO(task, 50))
		}
		writeJSON(writer, tasks)
	case http.MethodPost:
		if !s.requireSameOrigin(writer, request) {
			return
		}
		var req taskRequest
		if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
			http.Error(writer, "Invalid request.", http.StatusBadRequest)
			return
		}
		task, err := startTask(model, req)
		if err != nil {
			var user *userError
			if errors.As(err, &user) {
				http.Error(writer, user.Message, http.StatusBadRequest)
				return
			}
			var vfoxErr *vfox.VfoxError
			if errors.As(err, &vfoxErr) {
				http.Error(writer, vfoxErr.Message, http.StatusBadRequest)
				return
			}
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		writer.WriteHeader(http.StatusCreated)
		writeJSON(writer, taskDTO(task, 50))
	default:
		http.Error(writer, "Method not allowed.", http.StatusMethodNotAllowed)
	}
}

// userError is a rejected task request: invalid input, not a vfox failure.
type userError struct {
	Message string
}

func (e *userError) Error() string { return e.Message }

// startTask builds the command of req and queues it.
func startTask(model *state.AppModel, req taskRequest) (*state.VfoxTask, error) {
	cli := model.Cli
	must := func(command vfox.VfoxCommand, err error) (*state.VfoxTask, error) {
		if err != nil {
			return nil, err
		}
		return model.Run(command), nil
	}
	switch req.Operation {
	case "install":
		if req.Sdk == "" || req.Version == "" {
			return nil, &userError{"An SDK and a version are required."}
		}
		return must(cli.Install(req.Sdk, req.Version))
	case "uninstall":
		if req.Sdk == "" || req.Version == "" {
			return nil, &userError{"An SDK and a version are required."}
		}
		return must(cli.Uninstall(req.Sdk, req.Version))
	case "use":
		if req.Sdk == "" || req.Version == "" {
			return nil, &userError{"An SDK and a version are required."}
		}
		if req.Scope != "" && req.Scope != "global" {
			return nil, &userError{"Only the global scope is offered in the browser; pin projects in the desktop application."}
		}
		return must(cli.Use(req.Sdk, req.Version, core.UseScopeGlobal, ""))
	case "unuse":
		if req.Sdk == "" {
			return nil, &userError{"An SDK is required."}
		}
		return must(cli.Unuse(req.Sdk, core.UseScopeGlobal, ""))
	case "add":
		if req.Source == "" {
			return nil, &userError{"A plugin source is required."}
		}
		if !vfox.IsRemotePluginSource(req.Source) {
			return nil, &userError{"Only plugin URLs can be added from the browser; upload local files instead."}
		}
		return model.AddPluginFromSource(req.Source, req.Alias)
	case "remove":
		if req.Sdk == "" {
			return nil, &userError{"A plugin is required."}
		}
		return must(cli.RemovePlugin(req.Sdk))
	case "update":
		if req.Sdk == "" {
			return model.Run(cli.UpdateAllPlugins()), nil
		}
		return must(cli.UpdatePlugin(req.Sdk))
	default:
		return nil, &userError{"Unknown operation."}
	}
}

// serveTask answers one task and its actions: GET, POST cancel, POST retry
// and DELETE finished.
func (s *Server) serveTask(writer http.ResponseWriter, request *http.Request) {
	if !s.requireSession(writer, request) {
		return
	}
	model := s.requireModel(writer)
	if model == nil {
		return
	}
	rest := strings.TrimPrefix(request.URL.Path, "/api/tasks/")
	idText, action, _ := strings.Cut(rest, "/")
	id, err := strconv.Atoi(idText)
	if err != nil || strings.Contains(idText, "/") {
		if rest == "finished" && request.Method == http.MethodDelete {
			if !s.requireSameOrigin(writer, request) {
				return
			}
			model.Tasks.ClearFinished()
			writeJSON(writer, map[string]bool{"cleared": true})
			return
		}
		http.Error(writer, "No such task.", http.StatusNotFound)
		return
	}
	var task *state.VfoxTask
	for _, known := range model.Tasks.Tasks() {
		if known.ID == id {
			task = known
		}
	}
	if task == nil {
		http.Error(writer, "No such task.", http.StatusNotFound)
		return
	}
	switch {
	case request.Method == http.MethodGet && action == "":
		writeJSON(writer, taskDTO(task, 500))
	case request.Method == http.MethodPost && action == "cancel":
		if !s.requireSameOrigin(writer, request) {
			return
		}
		model.Tasks.Cancel(task)
		writeJSON(writer, taskDTO(task, 50))
	case request.Method == http.MethodPost && action == "retry":
		if !s.requireSameOrigin(writer, request) {
			return
		}
		writeJSON(writer, taskDTO(model.Tasks.Retry(task), 50))
	default:
		http.Error(writer, "Method not allowed.", http.StatusMethodNotAllowed)
	}
}

// servePlugins adds a plugin from a URL or an uploaded file.
func (s *Server) servePlugins(writer http.ResponseWriter, request *http.Request) {
	if !s.requireSession(writer, request) {
		return
	}
	model := s.requireModel(writer)
	if model == nil {
		return
	}
	if request.Method != http.MethodPost {
		http.Error(writer, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireSameOrigin(writer, request) {
		return
	}
	source, alias, cleanup, err := pluginSource(writer, request)
	if err != nil {
		var user *userError
		if errors.As(err, &user) {
			http.Error(writer, user.Message, http.StatusBadRequest)
			return
		}
		http.Error(writer, "Cannot read the upload.", http.StatusBadRequest)
		return
	}
	defer cleanup()
	task, err := model.AddPluginFromSource(source, alias)
	if err != nil {
		var vfoxErr *vfox.VfoxError
		if errors.As(err, &vfoxErr) {
			http.Error(writer, vfoxErr.Message, http.StatusBadRequest)
			return
		}
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	writer.WriteHeader(http.StatusCreated)
	writeJSON(writer, taskDTO(task, 50))
}

// pluginSource reads a plugin source from JSON or from an uploaded file. It
// returns a cleanup for the uploaded copy, if any.
func pluginSource(writer http.ResponseWriter, request *http.Request) (source, alias string, cleanup func(), err error) {
	cleanup = func() {}
	contentType := request.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		// Bound the upload before parsing: ParseMultipartForm alone keeps
		// up to 32 MiB in memory and spills the rest to disk without a
		// total limit.
		request.Body = http.MaxBytesReader(writer, request.Body, 33<<20)
		if err := request.ParseMultipartForm(32 << 20); err != nil { //nolint:gosec // Bounded above; G120 does not track MaxBytesReader on the body field.
			return "", "", cleanup, err
		}
		alias = request.FormValue("alias")
		file, header, err := request.FormFile("file")
		if err != nil {
			return "", "", cleanup, &userError{"An uploaded file or a source URL is required."}
		}
		defer func() { _ = file.Close() }()
		name := filepath.Base(header.Filename)
		if name == "" || name == "." || name == ".." {
			name = "plugin.lua"
		}
		content, err := io.ReadAll(file)
		if err != nil {
			return "", "", cleanup, err
		}
		dir, err := os.MkdirTemp("", "vfox-plus-upload-")
		if err != nil {
			return "", "", cleanup, err
		}
		cleanup = func() { _ = os.RemoveAll(dir) }
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, content, 0o600); err != nil {
			return "", "", cleanup, err
		}
		return path, alias, cleanup, nil
	}
	var req taskRequest
	if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
		return "", "", cleanup, &userError{"A source URL is required."}
	}
	if req.Source == "" {
		return "", "", cleanup, &userError{"A source URL is required."}
	}
	// Browsers add plugins by URL or by upload. A local path would hand an
	// HTTP-supplied filesystem path to os.ReadFile in PreparePluginSource,
	// so it is rejected here; the desktop application keeps accepting it.
	if !vfox.IsRemotePluginSource(req.Source) {
		return "", "", cleanup, &userError{"Only plugin URLs can be added from the browser; upload local files instead."}
	}
	return req.Source, req.Alias, cleanup, nil
}

// serveConfig edits one vfox configuration key: PUT sets it, DELETE resets
// it to its default. Either way the configuration is reloaded afterwards.
func (s *Server) serveConfig(writer http.ResponseWriter, request *http.Request) {
	if !s.requireSession(writer, request) {
		return
	}
	model := s.requireModel(writer)
	if model == nil {
		return
	}
	if !s.requireSameOrigin(writer, request) {
		return
	}
	switch request.Method {
	case http.MethodPut:
		var req struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		if err := json.NewDecoder(request.Body).Decode(&req); err != nil || strings.TrimSpace(req.Key) == "" {
			http.Error(writer, "A key is required.", http.StatusBadRequest)
			return
		}
		if err := model.VfoxConfig.Set(req.Key, req.Value); err != nil {
			writeVfoxError(writer, err)
			return
		}
		writeJSON(writer, map[string]bool{"saved": true})
	case http.MethodDelete:
		key := strings.TrimSpace(request.URL.Query().Get("key"))
		if key == "" {
			http.Error(writer, "A key is required.", http.StatusBadRequest)
			return
		}
		if err := model.VfoxConfig.Reset(key); err != nil {
			writeVfoxError(writer, err)
			return
		}
		writeJSON(writer, map[string]bool{"reset": true})
	default:
		http.Error(writer, "Method not allowed.", http.StatusMethodNotAllowed)
	}
}

// serveSettings reads and writes the GUI settings.
func (s *Server) serveSettings(writer http.ResponseWriter, request *http.Request) {
	if !s.requireSession(writer, request) {
		return
	}
	model := s.requireModel(writer)
	if model == nil {
		return
	}
	switch request.Method {
	case http.MethodGet:
		writeJSON(writer, settingsDTO(model.Settings.Settings()))
	case http.MethodPut:
		if !s.requireSameOrigin(writer, request) {
			return
		}
		var req struct {
			VfoxExecutable   *string           `json:"vfoxExecutable"`
			ExtraEnvironment map[string]string `json:"extraEnvironment"`
			Language         *string           `json:"language"`
		}
		if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
			http.Error(writer, "Invalid request.", http.StatusBadRequest)
			return
		}
		if req.Language != nil {
			switch settings.LanguagePreference(*req.Language) {
			case settings.LanguageSystem, settings.LanguageEnglish, settings.LanguageSimplifiedChinese:
			default:
				http.Error(writer, "Unknown language.", http.StatusBadRequest)
				return
			}
		}
		model.Settings.Update(func(current settings.AppSettings) settings.AppSettings {
			if req.VfoxExecutable != nil {
				current.VfoxExecutable = strings.TrimSpace(*req.VfoxExecutable)
			}
			if req.ExtraEnvironment != nil {
				current.ExtraEnvironment = req.ExtraEnvironment
			}
			if req.Language != nil {
				current.Language = settings.LanguagePreference(*req.Language)
			}
			return current
		})
		writeJSON(writer, settingsDTO(model.Settings.Settings()))
	default:
		http.Error(writer, "Method not allowed.", http.StatusMethodNotAllowed)
	}
}

// settingsDTO is the editable subset of the GUI settings.
func settingsDTO(current settings.AppSettings) map[string]interface{} {
	return map[string]interface{}{
		"vfoxExecutable":   current.VfoxExecutable,
		"extraEnvironment": current.ExtraEnvironment,
		"language":         string(current.Language),
	}
}

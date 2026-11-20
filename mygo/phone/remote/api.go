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

package remote

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"unicode"
)

// MaxPluginFile is the largest plugin file that the phone sends: 32 MiB. The server
// takes an upload of 33 MiB, the file and the form around it, and keeps up to 32 MiB
// of it in memory (mygo/pair, pluginSource).
const MaxPluginFile = 32 << 20

// Inventory returns the installed SDKs, the plugin registry and the vfox
// configuration (GET /api/inventory).
func (c *Client) Inventory(ctx context.Context) (Inventory, error) {
	return result[Inventory](ctx, c, http.MethodGet, "/api/inventory", nil, nil)
}

// SdkInfo returns the metadata of the plugin name (GET /api/sdks/<name>/info).
func (c *Client) SdkInfo(ctx context.Context, name string) (PluginInfo, error) {
	path, err := sdkPath(name, "/info")
	if err != nil {
		return PluginInfo{}, err
	}
	return result[PluginInfo](ctx, c, http.MethodGet, path, nil, nil)
}

// Versions returns the versions that the plugin name can install. A non-empty
// search asks the plugin to list another choice for this call only, such as the
// short name of a Java distribution (GET /api/sdks/<name>/versions?search=).
func (c *Client) Versions(ctx context.Context, name, search string) ([]AvailableVersion, error) {
	path, err := sdkPath(name, "/versions")
	if err != nil {
		return nil, err
	}
	var query url.Values
	if search != "" {
		query = url.Values{"search": {search}}
	}
	return result[[]AvailableVersion](ctx, c, http.MethodGet, path, query, nil)
}

// Profile returns the profile of the plugin name (GET /api/sdks/<name>/profile).
// A plugin with no profile gives (nil, nil): the server answers 404 for it, which
// is not an error here.
func (c *Client) Profile(ctx context.Context, name string) (*Profile, error) {
	path, err := sdkPath(name, "/profile")
	if err != nil {
		return nil, err
	}
	profile, err := result[*Profile](ctx, c, http.MethodGet, path, nil, nil)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return profile, nil
}

// Aliases returns the aliases of the plugin name (GET /api/sdks/<name>/aliases).
func (c *Client) Aliases(ctx context.Context, name string) ([]VersionAlias, error) {
	path, err := sdkPath(name, "/aliases")
	if err != nil {
		return nil, err
	}
	return result[[]VersionAlias](ctx, c, http.MethodGet, path, nil, nil)
}

// Doctor returns the health of vfox and of the installed SDKs. Empty sdk and
// version ask for all of them (GET /api/doctor?sdk=&version=).
func (c *Client) Doctor(ctx context.Context, sdk, version string) (DoctorReport, error) {
	query := url.Values{}
	if sdk != "" {
		query.Set("sdk", sdk)
	}
	if version != "" {
		query.Set("version", version)
	}
	return result[DoctorReport](ctx, c, http.MethodGet, "/api/doctor", query, nil)
}

// SetSearch chooses what the plugin name lists, by a value of its profile; an
// empty value is the default of the plugin (PUT /api/sdks/<name>/search).
func (c *Client) SetSearch(ctx context.Context, name, value string) error {
	path, err := sdkPath(name, "/search")
	if err != nil {
		return err
	}
	return c.exchange(ctx, http.MethodPut, path, nil, searchRequest{Value: value}, nil)
}

// SetVariable sets the environment variable key of the plugin name; an empty
// value takes the variable away (PUT /api/sdks/<name>/variables).
func (c *Client) SetVariable(ctx context.Context, name, key, value string) error {
	path, err := sdkPath(name, "/variables")
	if err != nil {
		return err
	}
	return c.exchange(ctx, http.MethodPut, path, nil, variableRequest{Key: key, Value: value}, nil)
}

// PutConfig sets a key of the vfox configuration (PUT /api/config).
func (c *Client) PutConfig(ctx context.Context, key, value string) error {
	return c.exchange(ctx, http.MethodPut, "/api/config", nil, configRequest{Key: key, Value: value}, nil)
}

// DeleteConfig resets a key of the vfox configuration to its default
// (DELETE /api/config?key=).
func (c *Client) DeleteConfig(ctx context.Context, key string) error {
	return c.exchange(ctx, http.MethodDelete, "/api/config", url.Values{"key": {key}}, nil, nil)
}

// Tasks returns the tasks, with the newest 50 log lines of each (GET /api/tasks).
func (c *Client) Tasks(ctx context.Context) ([]Task, error) {
	return result[[]Task](ctx, c, http.MethodGet, "/api/tasks", nil, nil)
}

// Task returns one task with its newest 500 log lines (GET /api/tasks/<id>).
// The server answers 404 for a task it does not have.
func (c *Client) Task(ctx context.Context, id int) (Task, error) {
	return result[Task](ctx, c, http.MethodGet, taskPath(id), nil, nil)
}

// TaskLog returns the complete output of a failed task as plain text
// (GET /api/tasks/<id>/log). The server answers 404 when it kept no complete
// output for the task, which is then ErrNotFound.
func (c *Client) TaskLog(ctx context.Context, id int) (string, error) {
	answer, err := c.call(ctx, http.MethodGet, taskPath(id)+"/log", nil, nil)
	if err != nil {
		return "", err
	}
	return string(answer), nil
}

// StartTask queues a task and returns it (POST /api/tasks).
func (c *Client) StartTask(ctx context.Context, request TaskRequest) (Task, error) {
	return result[Task](ctx, c, http.MethodPost, "/api/tasks", nil, request)
}

// CancelTask cancels a task (POST /api/tasks/<id>/cancel).
func (c *Client) CancelTask(ctx context.Context, id int) error {
	return c.exchange(ctx, http.MethodPost, taskPath(id)+"/cancel", nil, nil, nil)
}

// RetryTask runs a task again (POST /api/tasks/<id>/retry).
func (c *Client) RetryTask(ctx context.Context, id int) error {
	return c.exchange(ctx, http.MethodPost, taskPath(id)+"/retry", nil, nil, nil)
}

// ClearFinished forgets the finished tasks (DELETE /api/tasks/finished).
func (c *Client) ClearFinished(ctx context.Context) error {
	return c.exchange(ctx, http.MethodDelete, "/api/tasks/finished", nil, nil, nil)
}

// AddPlugin adds a plugin from a URL and returns the task that adds it; alias is
// optional (POST /api/plugins).
func (c *Client) AddPlugin(ctx context.Context, source, alias string) (Task, error) {
	return result[Task](ctx, c, http.MethodPost, "/api/plugins", nil, pluginRequest{Source: source, Alias: alias})
}

// UploadPlugin adds a plugin from a file and returns the task that adds it; alias is
// optional (POST /api/plugins, as multipart/form-data with the fields file and alias,
// as the web interface sends them). name is the name of the file, which the server
// keeps for vfox, and which must end in .zip or .lua: those are the plugins that vfox
// adds from a file. size is the number of bytes that content holds, at most
// MaxPluginFile. content is read as it is sent, and a content that has more or fewer
// bytes than size fails the upload before the form is completed (ErrFileChanged), so
// the server never takes a plugin that was cut short. A file that is not a plugin or is
// too large is refused before anything is sent (ErrNotPluginFile, ErrPluginFileTooLarge).
func (c *Client) UploadPlugin(ctx context.Context, name string, content io.Reader, size int64, alias string) (Task, error) {
	name = path.Base(strings.ReplaceAll(name, "\\", "/"))
	// A control character would end a line of the header of the part that holds the file.
	if lower := strings.ToLower(name); (!strings.HasSuffix(lower, ".zip") && !strings.HasSuffix(lower, ".lua")) ||
		strings.ContainsFunc(name, unicode.IsControl) {
		return Task{}, ErrNotPluginFile
	}
	if size < 0 || size > MaxPluginFile {
		return Task{}, ErrPluginFileTooLarge
	}
	// The form is written around the file: what comes before it, then the file, then the
	// closing delimiter, which the writer would put after the last part (Close).
	var head bytes.Buffer
	form := multipart.NewWriter(&head)
	if alias != "" {
		if err := form.WriteField("alias", alias); err != nil {
			return Task{}, fmt.Errorf("remote: make the form: %w", err)
		}
	}
	if _, err := form.CreateFormFile("file", name); err != nil {
		return Task{}, fmt.Errorf("remote: make the form: %w", err)
	}
	tail := "\r\n--" + form.Boundary() + "--\r\n"
	body := &formBody{
		contentType: form.FormDataContentType(),
		length:      int64(head.Len()) + size + int64(len(tail)),
		body:        io.MultiReader(&head, &exactReader{reader: content, size: size}, strings.NewReader(tail)),
	}
	return result[Task](ctx, c, http.MethodPost, "/api/plugins", nil, body)
}

// exactReader reads a file that is said to have size bytes, and fails with ErrFileChanged,
// as the file is read, when it has more or fewer. The error ends the request before the
// closing delimiter of the form is sent, so the form is never complete, and the server
// cannot take a plugin that was cut short or padded. The transport would find the mismatch
// by itself, but only after the body, and a server that answers first hides it from the
// caller.
type exactReader struct {
	reader io.Reader
	size   int64
	read   int64
}

// Read implements io.Reader. The bytes beyond the size that the file is said to have are
// the ones that show that it grew, and it does not pass them on.
func (e *exactReader) Read(p []byte) (int, error) {
	n, err := e.reader.Read(p)
	e.read += int64(n)
	switch {
	case e.read > e.size:
		return n - int(e.read-e.size), ErrFileChanged
	case errors.Is(err, io.EOF) && e.read < e.size:
		return n, ErrFileChanged
	}
	return n, err
}

// Settings returns the settings of the desktop application (GET /api/settings).
func (c *Client) Settings(ctx context.Context) (AppSettings, error) {
	return result[AppSettings](ctx, c, http.MethodGet, "/api/settings", nil, nil)
}

// PutSettings changes the settings that patch sets, and returns them all
// (PUT /api/settings).
func (c *Client) PutSettings(ctx context.Context, patch SettingsPatch) (AppSettings, error) {
	return result[AppSettings](ctx, c, http.MethodPut, "/api/settings", nil, patch)
}

// Web returns how other devices connect to the desktop (GET /api/web).
func (c *Client) Web(ctx context.Context) (WebState, error) {
	return result[WebState](ctx, c, http.MethodGet, "/api/web", nil, nil)
}

// PutWeb changes how other devices connect, and returns the new state
// (PUT /api/web).
func (c *Client) PutWeb(ctx context.Context, patch WebPatch) (WebState, error) {
	return result[WebState](ctx, c, http.MethodPut, "/api/web", nil, patch)
}

// WebPairing makes a pairing link and its QR code for another device
// (POST /api/web/pairing). host names the address of the desktop to use; empty
// lets the server choose the likeliest one.
func (c *Client) WebPairing(ctx context.Context, host string) (PairingCode, error) {
	return result[PairingCode](ctx, c, http.MethodPost, "/api/web/pairing", nil, hostRequest{Host: host})
}

// Quit ends the desktop application (POST /api/quit). The server answers first
// and quits a moment later, so the connection ends after this call returns.
func (c *Client) Quit(ctx context.Context) error {
	return c.exchange(ctx, http.MethodPost, "/api/quit", nil, nil, nil)
}

// taskPath returns the path of the task with id.
func taskPath(id int) string {
	return "/api/tasks/" + strconv.Itoa(id)
}

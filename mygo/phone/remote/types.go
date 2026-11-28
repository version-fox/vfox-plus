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
	"slices"
	"strings"
)

// The types of the answers of the remote API. Their JSON keys are those that the
// server writes (mygo/pair/webapi.go, webprofile.go and web.go). Some models of
// the desktop core have no JSON tags, so the server writes their Go field names;
// the types here carry the same names as tags. The contract tests decode the
// server's answers strictly to keep these in step.

// Text is a text in English and, when the server has one, in Chinese.
type Text struct {
	EN string `json:"en"`
	ZH string `json:"zh,omitempty"`
}

// In returns the Chinese text when chinese is set and the server sent one, and
// the English text otherwise.
func (t Text) In(chinese bool) string {
	if chinese && t.ZH != "" {
		return t.ZH
	}
	return t.EN
}

// Availability values of vfox, in VfoxStatus.Availability.
const (
	AvailabilityUnknown  = "unknown"
	AvailabilityChecking = "checking"
	AvailabilityReady    = "ready"
	AvailabilityNotFound = "notFound"
	AvailabilityBroken   = "broken"
)

// Inventory is what GET /api/inventory answers: vfox, the installed SDKs, the
// plugin registry and the vfox configuration. Each section loads on its own;
// Complete is true when all of them have loaded.
type Inventory struct {
	Vfox      VfoxStatus                 `json:"vfox"`
	Installed Section[[]InstalledSdk]    `json:"installed"`
	Registry  Section[[]AvailablePlugin] `json:"registry"`
	Config    Section[[]ConfigEntry]     `json:"config"`
	Complete  bool                       `json:"complete"`
}

// Section is one part of the inventory.
type Section[T any] struct {
	Loading bool   `json:"loading"`
	Error   string `json:"error,omitempty"`
	Value   T      `json:"value"`
}

// VfoxStatus says whether vfox runs, which version it is, and how the shells of the
// computer activate it, the shell of its person first. A computer with an older vfox+
// says nothing about its shells.
type VfoxStatus struct {
	Availability string       `json:"availability"`
	Version      string       `json:"version,omitempty"`
	Error        string       `json:"error,omitempty"`
	Activations  []Activation `json:"activations,omitempty"`
}

// Kinds of activations, in Activation.Kind.
const (
	// ActivationStartup: the command is a line for the startup file of the shell.
	ActivationStartup = "startup"
	// ActivationOnce: the command is run once in the shell, and changes its startup
	// file itself.
	ActivationOnce = "once"
	// ActivationScript: the command shows a directory, into which the script at URL
	// goes.
	ActivationScript = "script"
)

// Activation is how a shell comes to use the versions that vfox selects.
type Activation struct {
	// Shell is the name that `vfox activate` knows the shell by.
	Shell string `json:"shell"`
	// Name is what the shell is called.
	Name string `json:"name"`
	// Kind says what the command is for.
	Kind string `json:"kind"`
	// File is the startup file of an ActivationStartup.
	File string `json:"file,omitempty"`
	// Command is the line or the command.
	Command string `json:"command"`
	// URL is the script of an ActivationScript.
	URL string `json:"url,omitempty"`
}

// InstalledSdk is an installed plugin and the versions that it manages.
type InstalledSdk struct {
	Name       string   `json:"Name"`
	Versions   []string `json:"Versions"`
	Current    string   `json:"Current"`
	HasCurrent bool     `json:"HasCurrent"`
}

// AvailablePlugin is a plugin of the registry.
type AvailablePlugin struct {
	Name     string `json:"Name"`
	Homepage string `json:"Homepage"`
	Official bool   `json:"Official"`
}

// ConfigEntry is one key of the vfox configuration. Value is empty when the key
// is not set.
type ConfigEntry struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

// PluginInfo is the metadata of an installed plugin.
type PluginInfo struct {
	Name        string `json:"Name"`
	Version     string `json:"Version"`
	Homepage    string `json:"Homepage"`
	InstallPath string `json:"InstallPath"`
	Description string `json:"Description"`
}

// VersionAddition is a tool that comes with an SDK version, such as npm.
type VersionAddition struct {
	Name    string `json:"Name"`
	Version string `json:"Version"`
}

// AvailableVersion is a version that can be installed. Parts and Facets are set
// when the plugin has a profile that describes the versions.
type AvailableVersion struct {
	Version   string              `json:"Version"`
	Note      string              `json:"Note"`
	HasNote   bool                `json:"HasNote"`
	Additions []VersionAddition   `json:"Additions"`
	Installed bool                `json:"Installed"`
	Parts     *VersionParts       `json:"parts,omitempty"`
	Facets    map[string][]string `json:"facets,omitempty"`
}

// VersionParts is a version taken apart by the profile of its plugin.
type VersionParts struct {
	Version string   `json:"version"`
	Source  *Choice  `json:"source,omitempty"`
	Flavors []string `json:"flavors,omitempty"`
}

// Choice is one value of a profile choice: a distribution, a mirror or a
// search.
type Choice struct {
	Value      string   `json:"value"`
	Name       Text     `json:"name"`
	Maker      Text     `json:"maker"`
	Aliases    []string `json:"aliases,omitempty"`
	Popular    bool     `json:"popular,omitempty"`
	Everything bool     `json:"everything,omitempty"`
}

// Profile describes a plugin: the choices that change what it lists, the
// variables it reads and the facets of its versions. GET
// /api/sdks/<name>/profile answers it, or 404 when the plugin has none.
type Profile struct {
	Plugin         string                  `json:"plugin"`
	Search         *ProfileSearch          `json:"search,omitempty"`
	Shorthands     []Shorthand             `json:"shorthands"`
	Variables      []Variable              `json:"variables"`
	LegacyFiles    []string                `json:"legacyFiles"`
	LegacyStrategy bool                    `json:"legacyStrategy"`
	Facets         []Facet                 `json:"facets"`
	Guide          Text                    `json:"guide"`
	Tips           []Text                  `json:"tips"`
	Docs           []Link                  `json:"docs"`
	Installed      map[string]VersionParts `json:"installed,omitempty"`
	// Capabilities is what the plugin can do; a desktop that does not say is
	// older (see CanDetect).
	Capabilities *Capabilities `json:"capabilities,omitempty"`
}

// Capabilities is what a plugin can do, as its files tell.
type Capabilities struct {
	Detect bool `json:"detect"`
}

// CanDetect reports whether the plugin may find installs that are on the
// machine already, which are offered to be linked.
func (p *Profile) CanDetect() bool {
	return p == nil || p.Capabilities == nil || p.Capabilities.Detect
}

// ProfileSearch is the choice of what the plugin lists. Current is the value
// that the plugin lists now.
type ProfileSearch struct {
	Label   Text     `json:"label"`
	Default string   `json:"default"`
	Current string   `json:"current"`
	Choices []Choice `json:"choices"`
}

// Shorthand is a short name that the plugin accepts, with its meaning.
type Shorthand struct {
	Name    string `json:"name"`
	Meaning Text   `json:"meaning"`
}

// Variable kinds of Variable.Kind.
const (
	VariableURL    = "url"
	VariableChoice = "choice"
	VariableToggle = "toggle"
	VariableText   = "text"
)

// Variable is an environment variable of a plugin. Value is what vfox runs with
// now, empty when the variable is not set. On and Off are the values of a
// toggle, of which an empty Off takes the variable away. Unvetted are the values
// of the profile that the application did not ship: they are shown, and asked
// about, before they are set.
type Variable struct {
	Key      string   `json:"key"`
	Label    Text     `json:"label"`
	Help     Text     `json:"help"`
	Kind     string   `json:"kind"`
	Presets  []Preset `json:"presets"`
	On       string   `json:"on,omitempty"`
	Off      string   `json:"off,omitempty"`
	Example  string   `json:"example,omitempty"`
	Advanced bool     `json:"advanced,omitempty"`
	Unvetted []string `json:"unvetted,omitempty"`
	Value    string   `json:"value"`
}

// NeedsConfirmation reports whether the value comes from a profile that the
// application did not ship, so that it is shown, and asked about, before it is
// set. An empty value takes the variable away, which asks nothing.
func (v Variable) NeedsConfirmation(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && slices.Contains(v.Unvetted, value)
}

// Preset is a value that a variable offers.
type Preset struct {
	Value string `json:"value"`
	Name  Text   `json:"name"`
}

// Facet is a filter of the versions of a plugin, with its values.
type Facet struct {
	ID     string   `json:"id"`
	Label  Text     `json:"label"`
	Values []string `json:"values"`
}

// Link is a documentation link of a plugin.
type Link struct {
	Title Text `json:"title"`
	URL   Text `json:"url"`
}

// VersionAlias is one alias of vfox: <sdk>@<alias> points at a version.
type VersionAlias struct {
	Sdk     string `json:"sdk"`
	Alias   string `json:"alias"`
	Target  string `json:"target"`
	Display string `json:"display"`
}

// DoctorReport is the health of vfox and of the installed SDKs.
type DoctorReport struct {
	OK       bool            `json:"ok"`
	Findings []DoctorFinding `json:"findings"`
}

// DoctorFinding is one problem, warning or notice of a DoctorReport.
type DoctorFinding struct {
	ID       string   `json:"id"`
	Severity string   `json:"severity"`
	Sdk      string   `json:"sdk"`
	Version  string   `json:"version"`
	Message  string   `json:"message"`
	Hints    []string `json:"hints"`
	Fix      []string `json:"fix"`
	Ignored  bool     `json:"ignored"`
}

// Task is a vfox operation that runs in the queue of the desktop. Log holds the
// newest lines (the last 50 in a list, 500 for one task); HasFullLog reports
// whether TaskLog can answer the complete output of a failed task.
type Task struct {
	ID           int      `json:"id"`
	Operation    string   `json:"operation"`
	Sdk          string   `json:"sdk,omitempty"`
	Version      string   `json:"version,omitempty"`
	Alias        string   `json:"alias,omitempty"`
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
	DroppedLines int      `json:"droppedLines"`
	HasFullLog   bool     `json:"hasFullLog,omitempty"`
}

// Status values of Task.Status.
const (
	TaskQueued    = "queued"
	TaskRunning   = "running"
	TaskSucceeded = "succeeded"
	TaskFailed    = "failed"
	TaskCancelled = "cancelled"
)

// Operations that a TaskRequest starts, in TaskRequest.Operation. They are not
// the names that a Task reports: see the Operation constants for those.
const (
	RequestInstall     = "install"
	RequestUninstall   = "uninstall"
	RequestUse         = "use"
	RequestUnuse       = "unuse"
	RequestAdd         = "add"
	RequestRemove      = "remove"
	RequestUpdate      = "update"
	RequestLink        = "link"
	RequestLinkDetect  = "linkDetect"
	RequestAliasSet    = "aliasSet"
	RequestAliasUnset  = "aliasUnset"
	RequestDoctorAdopt = "doctorAdopt"
)

// Operations of a Task, in Task.Operation, named as vfox names them. An install
// that also selects the version reports OperationInstallUse; an update without a
// plugin reports OperationUpdateAllPlugins.
const (
	OperationInstall          = "install"
	OperationInstallUse       = "installUse"
	OperationUninstall        = "uninstall"
	OperationUse              = "use"
	OperationUnuse            = "unuse"
	OperationAddPlugin        = "addPlugin"
	OperationRemovePlugin     = "removePlugin"
	OperationUpdatePlugin     = "updatePlugin"
	OperationUpdateAllPlugins = "updateAllPlugins"
	OperationLink             = "link"
	OperationLinkDetect       = "linkDetect"
	OperationAliasSet         = "aliasSet"
	OperationAliasUnset       = "aliasUnset"
	OperationDoctorAdopt      = "doctorAdopt"
)

// TaskRequest starts a task. Only the fields that the operation uses are sent.
type TaskRequest struct {
	// Operation is one of the Request constants.
	Operation string `json:"operation"`
	Sdk       string `json:"sdk,omitempty"`
	Version   string `json:"version,omitempty"`
	Scope     string `json:"scope,omitempty"`
	Source    string `json:"source,omitempty"`
	Alias     string `json:"alias,omitempty"`
	Path      string `json:"path,omitempty"`
	Force     bool   `json:"force,omitempty"`
	Yes       bool   `json:"yes,omitempty"`
	Use       bool   `json:"use,omitempty"`
	Reinstall bool   `json:"reinstall,omitempty"`
}

// Language values of AppSettings.Language and SettingsPatch.Language.
const (
	LanguageSystem            = "system"
	LanguageEnglish           = "english"
	LanguageSimplifiedChinese = "simplifiedChinese"
)

// AppSettings are the settings of the desktop application that the API reads.
type AppSettings struct {
	VfoxExecutable   string            `json:"vfoxExecutable"`
	ExtraEnvironment map[string]string `json:"extraEnvironment"`
	Language         string            `json:"language"`
	UpdateMirror     string            `json:"updateMirror"`
}

// SettingsPatch changes the settings that are set. A nil field is left as it
// is; a non-nil one, even an empty string, is written. ExtraEnvironment is a
// pointer so that an empty map (no variables) is sent too.
type SettingsPatch struct {
	VfoxExecutable   *string            `json:"vfoxExecutable,omitempty"`
	ExtraEnvironment *map[string]string `json:"extraEnvironment,omitempty"`
	Language         *string            `json:"language,omitempty"`
	UpdateMirror     *string            `json:"updateMirror,omitempty"`
}

// WebState says how other devices connect to the desktop, and what the server
// is doing about it.
type WebState struct {
	Remote        bool     `json:"remote"`
	Port          int      `json:"port"`
	Lightweight   bool     `json:"lightweight"`
	Listening     bool     `json:"listening"`
	Addresses     []string `json:"addresses"`
	PublicAddress string   `json:"publicAddress"`
	Tunnel        bool     `json:"tunnel"`
	Fingerprint   string   `json:"fingerprint"`
	SelfSigned    bool     `json:"selfSigned"`
	Error         string   `json:"error"`
}

// WebPatch changes the settings of WebState that are set; a nil field is left as
// it is.
type WebPatch struct {
	Remote      *bool `json:"remote,omitempty"`
	Port        *int  `json:"port,omitempty"`
	Lightweight *bool `json:"lightweight,omitempty"`
}

// PairingCode is a pairing link for another device, and the QR code of it as an
// SVG document.
type PairingCode struct {
	URL string `json:"url"`
	SVG string `json:"svg"`
}

// The bodies that the calls send. They are not part of the API surface.

type searchRequest struct {
	Value string `json:"value"`
}

type variableRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type configRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type hostRequest struct {
	Host string `json:"host"`
}

type pluginRequest struct {
	Source string `json:"source"`
	Alias  string `json:"alias,omitempty"`
}

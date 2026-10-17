# User interface specification

This document describes what every vfox-plus implementation shows and does.
Visual details — colours, shapes, type and motion — are in
[the design system](design-system.md); texts are in
[`shared/i18n`](../shared/i18n) and referred to here by their keys, such as
`navHome`. Icons are referred to by their names in
[`shared/icons.json`](../shared/icons.json), such as `sdks`.

## Window and navigation

The application has five top-level destinations:

| Destination | Label key     | Icon       | Purpose                                              |
| ----------- | ------------- | ---------- | ---------------------------------------------------- |
| Home        | `navHome`     | `home`     | Overview and shortcuts.                              |
| SDKs        | `navSdks`     | `sdks`     | Installed plugins and their versions.                |
| Plugins     | `navPlugins`  | `plugins`  | The plugin registry.                                 |
| Tasks       | `navTasks`    | `tasks`    | Running and finished operations. Shows a badge with the number of queued and running tasks. |
| Settings    | `navSettings` | `settings` | Appearance, language, vfox executable, vfox configuration, about. |

The layout follows the Material 3 window size classes of the window width:

- **Compact** (< 600 dp): a navigation bar at the bottom.
- **Medium and wider** (≥ 600 dp): the navigation rail at the start. Its menu
  button toggles between the collapsed rail (labels below icons) and the
  expanded rail (labels next to icons); the choice is remembered
  (`railExpanded`). The rail holds a floating action button, `navAddPlugin`,
  that opens Plugins.

Pages sit on rounded panes, 16 dp from the window edges. Switching destinations
uses the fade-through transition. The selected destination's icon is filled;
the others are outlined.

The minimum window size is 360 × 480 dp. The default size is 1280 × 820 dp.

## Common states

Every view that loads data has four states:

- **Loading**: the loading indicator, centred, with a short text when the wait
  can be long (`availableLoading`).
- **Error**: the error icon on a soft-burst shape, a title saying what failed
  (`sdksLoadError`, …), vfox's error message and a retry button
  (`actionRetry`).
- **Empty**: an icon on an expressive shape, a title, an explanation and the
  action that resolves it (for example `sdksEmptyTitle` with a button to
  Plugins).
- **Content**. While reloading, the previous content stays visible.

Long-running operations never block the interface: they become tasks (see
[Tasks](#tasks)).

## vfox unavailable

While vfox is being located, every page except Settings shows the loading
indicator with `vfoxCheckingTitle`. If no executable is found it shows
`vfoxMissingTitle` / `vfoxMissingBody` with three actions: open the
installation guide (https://vfox.dev/guides/quick-start.html), choose the
executable (`vfoxLocate`) and retry. If the executable does not report a
version it shows `vfoxBrokenTitle` / `vfoxBrokenBody` with the same actions.
Settings stay reachable in every case.

## Home

- Header: the application name, `vfox <version> · <executable path>`, and a
  refresh button.
- Three statistic tiles: number of plugins, of installed versions and of
  plugins with a version in use. Each tile has its own container colour
  (primary, secondary, tertiary) and a large decorative shape partly hidden
  behind its edge. Tapping a tile opens SDKs.
- `homeActiveTitle`: a segmented list of the plugins that have a version in use,
  with their avatar and version. Tapping an item opens it on the SDKs page.
  Without any, the list shows `homeActiveEmpty`.
- A card explaining how to use the versions in a terminal (`homeShellTitle`,
  `homeShellBody`) with the line `eval "$(vfox activate bash)"` and a copy
  button.
- Two shortcut cards: `homeExploreTitle` (opens Plugins) and `homeDocsTitle`
  (opens https://vfox.dev/).

## SDKs

A list-detail layout. When the page is at least 840 dp wide, the list (360 dp)
and the detail pane are side by side; the detail pane shows `sdksSelectPrompt`
until a plugin is selected and fades through when the selection changes. On
narrower pages the list and the detail replace each other along the horizontal
shared axis, and the detail has a back button.

### List

- Header with a refresh button and a search bar (`sdksSearchHint`) that filters
  by name.
- A segmented list of the installed plugins: avatar, name, current version
  (`sdksCurrent` or `sdksNoCurrent`) and number of installed versions
  (`sdksVersionCount`). The selected plugin is highlighted.
- Empty: `sdksEmptyTitle` / `sdksEmptyBody` with `navAddPlugin`.

### Detail

- Header: a 72 dp avatar, the name, a tag with the current version and one with
  the plugin version (`sdkPluginVersion`), and the plugin description. Actions:
  update the plugin (`sdkUpdatePlugin`; replaced by a loading indicator while it
  runs) and a menu with `actionOpenHomepage` and `sdkRemovePlugin`. On narrow
  panes the actions move to a row above the avatar.
- A connected button group switches between `sdkTabInstalled` (with the count)
  and `sdkTabAvailable`.

**Installed** lists the installed versions, newest first:

- The current version has a filled `current` icon, an emphasized label and the
  tag `versionInUse`; its menu offers `versionUseInProject`, `versionUnuse`,
  `versionCopyPath` and `versionUninstall`.
- Other versions have a split button: `versionUse` makes the version the global
  default; its menu offers `versionUseGlobally` and `versionUseInProject`.
  Their overflow menu offers `versionCopyPath` and `versionUninstall`.
- While a task runs for a version, its actions are replaced by a loading
  indicator.
- Empty: `sdkInstalledEmpty` with `sdkBrowseVersions`.

**Available** runs `vfox search` the first time it is opened:

- A filter field (`availableSearchHint`) matching versions and notes, a
  connected button group `availableFilterAll` / `availableFilterTagged`
  (versions with a note such as LTS) / `availableFilterInstalled`, and the
  number of listed versions (`availableCount`).
- A lazily built segmented list. Each version shows its note as a tag and the
  bundled tools (`npm 11.19.1`). The trailing action is `versionInstall`; for
  an installed version, a disabled `versionInstalled`; while installing, a wavy
  progress indicator with `taskDownloading` and a cancel button.

### Confirmations and dialogs

- Removing a plugin: `sdkRemoveTitle` / `sdkRemoveBody` (with the number of
  versions that will be deleted), destructive confirmation `sdkRemoveConfirm`.
- Uninstalling: `versionUninstallTitle` with `versionUninstallBody`, or
  `versionUninstallCurrentBody` for the current version.
- Using in a project: `useProjectTitle` / `useProjectBody`, a directory field
  with a browse button; the directory must exist (`useProjectInvalid`).

Dialogs scale in on a spring.

## Plugins

- Header: `pluginsTitle`, `pluginsSubtitle`; actions: update all plugins,
  refresh, and `pluginsAddCustom`.
- A search bar (`pluginsSearchHint`, matching name and homepage) and a
  connected button group `pluginsFilterAll` / `pluginsFilterOfficial` /
  `pluginsFilterCommunity`.
- A responsive grid of cards: avatar, name, a `verified` icon for official
  plugins (tooltip `pluginsOfficial`), the repository path, a button that opens
  the homepage, and `pluginsAdd` — or `pluginsAdded` for installed plugins,
  which opens the plugin on the SDKs page.
- `pluginsAddCustom` asks for a source (URL or path of a `.zip` archive or a
  `.lua` script, with a file chooser limited to these types, labelled
  `pluginsCustomFileType`) and an optional alias. vfox moves local `.lua` files
  away, so the application passes it a temporary copy (see
  [the CLI contract](vfox-cli-contract.md#known-quirks-of-vfox-1012)).

## Tasks

A segmented list of tasks, newest first, with `tasksClearFinished` in the
header. Each task shows:

- a status: loading indicator (running), `queued`, `succeeded`, `error`, or
  `cancel` (cancelled);
- its title (`taskInstall`, `taskUse`, …) and status text; failed tasks show
  vfox's error message;
- for running installations, the wavy progress indicator;
- cancel (queued or running), retry (failed or cancelled) and a toggle that
  expands the output: the command line and the output in a monospaced, selectable
  block with a copy button.

When a task finishes while another page is shown, a snack bar reports it
(`taskSucceeded`). Failures (`taskFailed` plus vfox's message) stay until
dismissed and offer `taskView`, which opens Tasks.

## Settings

Grouped in segmented lists:

- **Appearance**: theme (`themeSystem` / `themeLight` / `themeDark` as a
  connected button group), `settingsSystemColor` (switch), eight colour swatches
  (choosing one turns the desktop colour off; the selected swatch morphs into a
  flower shape), and the language (`languageSystem` / `English` / `简体中文`;
  language names are always written in their own language).
- **vfox**: the executable with its origin (`settingsExecutableDetected`,
  `settingsExecutableCustom` or `settingsExecutableMissing`) and version, a
  browse button and, for a custom executable, `settingsExecutableDetect`; the
  extra environment variables as `KEY=VALUE` lines with validation
  (`settingsEnvironmentInvalid`).
- **vfox configuration**: an editor for each key listed in
  [the CLI contract](vfox-cli-contract.md#configuration) — switches for booleans,
  a menu for `legacyVersionFile.strategy`, text fields (saved on Enter or when
  leaving the field; clearing a field resets the key) for the rest, and
  `configOther` for keys the GUI does not know.
- **About**: application name, version and license, links to the source code
  and the vfox documentation, and the open-source licenses of the dependencies,
  including the bundled Material Symbols font (Apache-2.0, Google).

## Accessibility

- Every icon-only button has a tooltip, which also serves as its accessible
  name.
- Selected items, toggles and button groups expose their selected state.
- Headers are marked as headers.
- All motion stops or becomes instantaneous when the platform asks for reduced
  motion.
- Text follows the platform text scale; layouts wrap rather than clip.

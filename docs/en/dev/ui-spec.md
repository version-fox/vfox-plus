# User interface specification

> [简体中文](../../zh-CN/dev/ui-spec.md)

This document describes what vfox-plus shows and does.
Visual details — colours, shapes, type and motion — are in
[the design system](design-system.md); texts are in
[`shared/i18n`](../../../shared/i18n) and referred to here by their keys, such as
`navHome`. Icons are referred to by their names in
[`shared/icons.json`](../../../shared/icons.json), such as `sdks`.

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
  and `sdkTabAvailable`, and, for a plugin that has a
  [profile](architecture.md#plugin-profiles), `sdkTabOptions` (icon `options`).
  The tabs fade through.

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
- For a plugin with a profile, a version name is taken apart under its
  title: the distribution (`Temurin`, `OpenJDK`) and the flavours (`JavaFX`,
  `x64`, `OpenHarmony`) as a second line.

**Available** runs `vfox search` the first time it is opened. For a plugin that
has a profile, the first row holds:

- the **search button** `availableChoiceButton` ("Distribution: Temurin"; the
  label of the choice is the profile's, the icon `registry`), which opens the
  [choices dialog](#choices-and-names) of what the search lists, when the
  profile has such a search (`java`); the choice becomes the argument of
  `vfox search` and is remembered per plugin while the application runs, and the
  list of the earlier choice goes away at once;
- `availableInstallByName`, an outlined button that opens the
  [install by name dialog](#choices-and-names). Every plugin has it.

The row stays while the list loads or fails, so that another choice can be made.
Then:

- A filter field (`availableSearchHint`) matching versions and notes, a
  connected button group `availableFilterAll` / `availableFilterTagged`
  (versions with a note such as LTS) / `availableFilterInstalled`, and the
  number of listed versions (`availableCount`).
- **Filter chips**, one row per facet of the profile that at least one listed
  version has a value of, with the label of the facet in front (`Support`:
  LTS, JavaFX; `Channel`: stable, beta, dev, OpenHarmony; `Architecture`: x64,
  arm64), and `availableClearFilters` while a chip is on. A chip is a filter
  chip of Material 3 (see [the design system](design-system.md#components)):
  chips of one facet are alternatives, facets narrow each other and so do the
  filter field and the button group.
- A lazily built segmented list. Each version shows its note as a tag and the
  bundled tools (`npm 11.19.1`); for a search that lists several
  distributions, also a tag with the distribution. The trailing action is
  `versionInstall`; for an installed version, a disabled `versionInstalled`;
  while installing, a wavy progress indicator with `taskDownloading` and a
  cancel button.

**Options** is built from the profile of the plugin, in segmented lists:

- the guide to the version names of the plugin (`optionsGuideTitle`);
- `optionsVariables`: one item per environment variable the plugin reads (the
  GitHub mirror of Java and Flutter, the download mirror of Node.js, the
  storage mirror of Flutter, the C library of Node.js, tagged
  `optionsVariableAdvanced`), with the variable name, its value (with the name
  of the mirror, if it is one that was checked) or `optionsVariableUnset`, and
  `optionsVariableChange`, which opens the [variable dialog](#choices-and-names);
- `optionsLegacy`: the version files the plugin lets vfox read, as tags
  (`.sdkmanrc`; `.node-version` and `.nvmrc`), or `optionsLegacyNone`; the
  switch `configLegacyEnable` of the vfox configuration; and, where the strategy
  matters (Node.js), `configLegacyStrategy`, or `optionsLegacyStrategyIgnored`
  where it does not (Java);
- `optionsShorthands`: the names the plugin takes besides the listed versions,
  each with its meaning and `optionsShorthandTry`, which opens the install by
  name dialog with that name;
- `optionsTips`, the notes of the profile;
- `optionsDocs`: links that open the documentation of the plugin in the
  language of the application.

### Confirmations and dialogs

- Removing a plugin: `sdkRemoveTitle` / `sdkRemoveBody` (with the number of
  versions that will be deleted), destructive confirmation `sdkRemoveConfirm`.
- Uninstalling: `versionUninstallTitle` with `versionUninstallBody`, or
  `versionUninstallCurrentBody` for the current version.
- Using in a project: `useProjectTitle` / `useProjectBody`, a directory field
  with a browse button; the directory must exist (`useProjectInvalid`).

Dialogs scale in on a spring.

#### Choices and names

- **Choices** (`searchChoose…`): a dialog titled with the label of the choice
  (`Distribution`), a filter field (`searchChooseHint`, matching the short name,
  the long name, the name and the maker, in both languages) and a list of the
  choices in the order of the profile, the popular ones first; each shows its
  name, its maker and its short name, a check mark where it is the current one
  and `searchChoiceDefault` where it is the one vfox lists without an argument.
  Choosing one closes the dialog and lists it.
- **Install by name** (`installName…`): a field `installNameField`, which takes
  what follows `<plugin>@` in `vfox install` (helper `installNameHelp`), and, for
  a plugin with a profile, `installNameExamples`, one filter chip for each
  shorthand that fills the field, and the meaning of the shorthand that is
  typed. An empty name (`installNameRequired`) or one that has a space or a dash in
  front (`installNameInvalid`) is refused. The confirmation queues
  `vfox install --yes <plugin>@<name>` and shows Tasks.
- **Variable** (`variable…`): the help of the variable and its name; for an
  address, a monospaced field `variableAddress` (an address needs `http://` or
  `https://`, `variableInvalidAddress`); `variablePresets`, a chip for every
  value that was checked, and a chip `variableDefaultOption` that takes the
  variable away. The chips of a choice (`musl`, `glibc`) are its only values
  (`variableInvalidChoice`). The value goes to the extra environment of the
  settings, which the settings page shows too.

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
  [the CLI contract](vfox-cli-contract.md#known-quirks)).

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
  block with a copy button. Only the newest 500 lines of a finished task are
  kept; older lines are counted, not shown (`taskDroppedLines`).

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

# Design system

> [简体中文](../../zh-CN/dev/design-system.md)

vfox-plus follows [Material 3 Expressive](https://m3.material.io/), the 2025
evolution of Material Design that adds springs, a richer shape library, emphasized
typography and new components to Material 3. This document lists the tokens and
components the application uses, and where the mygo implementation
defines them.

## Colour

- Colour schemes are generated from a **seed colour** with the Material colour
  utilities, *tonal spot* variant, for light and dark themes, in the **2021**
  specification of the colour utilities, so that the application shows the
  same colours as the specification.
- The seed is the **desktop accent colour** when the user allows it and the
  desktop exposes one (XDG desktop portal `org.freedesktop.appearance
  accent-color`, falling back to the GTK theme), otherwise the colour chosen in
  Settings. The default is the orange of the vfox logo, `#F28C1B`.
- Large surfaces use the surface container roles: the window is `surface`, page
  panes `surfaceContainerLow`, list items `surfaceContainer`, prominent cards
  `surfaceContainerHighest`. Primary, secondary and tertiary containers mark the
  avatars.
- Destructive actions use `error`.
- The web interface (the `--m3-*` variables of `mygo/pair/webapp.html`) and the
  Android app (`values/colors.xml` and `values-night/colors.xml`) hold the 48 roles
  of the scheme as literal values, and `go test ./pair` in `mygo` fails when they
  are not the roles that `mygo/ui/mcu` computes for the seed.

mygo: `PaletteFor` in `mygo/ui/kit/theme.go`, computed from `mygo/ui/mcu` (a
Go port of the Material colour utilities, tonal spot).

## Shape

### Corner radius scale

| Token                  | Radius | Used for                                         |
| ---------------------- | -----: | ------------------------------------------------ |
| none                   |   0 dp |                                                  |
| extra small            |   4 dp | Inner corners of segmented lists, tooltips       |
| small                  |   8 dp | Pressed buttons, inner corners of button groups, chips |
| medium                 |  12 dp | Text fields, snack bars                          |
| large                  |  16 dp | Cards, menus, floating action buttons            |
| large increased        |  20 dp | Outer corners of segmented lists, plugin cards   |
| extra large            |  28 dp | Page panes, dialogs                              |
| extra large increased  |  32 dp |                                                  |
| extra extra large      |  48 dp |                                                  |
| full                   |  50 %  | Buttons at rest, search bars, tags               |

mygo: the `Corner*` constants in `mygo/ui/kit/draw.go`.

### Shape library and morphing

The expressive shapes of the
[Material shape library](https://m3.material.io/styles/shape/overview-principles)
give the interface its character:

- **Avatars** of plugins use a shape derived from a stable hash of the plugin
  name (9-sided cookie, 4-leaf clover, sunny, soft burst, 6-sided cookie, gem,
  pentagon, flower, puffy diamond, 4-sided cookie, arch, 8-leaf clover) and one
  of three colour pairs: primary fixed dim, secondary and tertiary container.
  Not secondary container, which marks selected list items, nor primary
  container, which some seeds (the vfox orange among them) make just as light.
  Hovering spins the shape by 30° on the fast spatial spring.
- **Empty and error states** show their icon on a large shape.
- **Colour swatches** morph from a circle into a flower, turning slightly, when
  selected.
- The **loading indicator** morphs through soft burst, 9-sided cookie, pentagon,
  pill, sunny, 4-sided cookie and oval.

Shapes come from `mygo/ui/shapes`, a Go port of `androidx.graphics.shapes`;
its `Polygon` and `Morph` types turn a shape, or a morph between two shapes,
into a path that the components of `mygo/ui/kit` draw.

### Shape morphing on interaction

Interactive elements change shape as they are used:

- Buttons are fully rounded at rest and their corners shrink to 8 dp while
  pressed.
- Connected button groups have fully rounded outer corners and 8 dp inner
  corners; the selected button becomes fully rounded and a pressed button's inner
  corners shrink to 4 dp.
- Segmented list items have 20 dp outer and 4 dp inner corners; the inner
  corners grow to 12 dp while hovered, pressed or selected.
- The trailing button of a split button becomes a circle while its menu is open.

## Typography

The application uses the platform's sans-serif font (Ubuntu on Ubuntu, with
Noto Sans CJK for Chinese) and the Material 3 type scale. Material 3 Expressive
adds an **emphasized** variant of each role with a heavier weight, used for page
titles, list titles that need attention (the current
version) and card titles.

| Role                       | Baseline weight | Emphasized weight |
| -------------------------- | --------------: | ----------------: |
| Display, headline, title L |             400 |               500 |
| Title M, title S           |             500 |               700 |
| Body L, body M             |             400 |               500 |
| Label L, label M           |             500 |               700 |

mygo: `Emph` in `mygo/ui/kit/theme.go`.

## Copy and density

The three interfaces say as little as they can. A text earns its place when it helps a
person to understand, to judge or to act; what the layout already says is not said again.

- **A label is not explained.** A title and its control say what a row does. A line
  under it is for what the control cannot say: the value (`Default`), a consequence
  that a person has to weigh, or a limit that an error would name later. What a
  setting is, where it is stored and what a mirror is belong in the documentation.
- **A format is an example in the empty field** (`j17`, `17-system`, `KEY=value`), not
  a caption under it. The field names the mistake when one is made.
- **An empty list has no sentence.** It has what fills it: the button that names an
  alias, or an icon, a title and the action.
- **A page is not announced by a card.** Nothing counts what the list below it holds,
  and nothing is a shortcut to a page that the navigation reaches.
- **No pane waits for a choice.** Side by side, the list has its first item open, the
  one in use; the ways to add something besides the list are one menu, not a button each.
- **A section has a title and no caption, and a page has its title in the bar and no
  subtitle.**
- **A consequence is said once**, in the confirmation that asks, and not again in the
  row that opens it.
- **A name is explained where it is typed.** The guide of a plugin is in the dialog of
  the names, not in its options. The notes of a profile are the few sentences that no
  control says, at most three.
- **An icon tells rows apart.** Rows that start with the same icon have an icon that
  says nothing; the rows of a settings list each have their own.
- **A row has one action.** The others are in its menu: a split button, or the overflow
  button. The segments of a segmented control are one action.
- **Sizes are the ones of the type scale** (11, 12, 14, 16, 22, 24, 28, 32, 36, 45 and
  57 px); the monospace of code, 13 px, is read as a size of its own.

### Budgets

The census of a screen is taken by `webheap -audit` for the web interface, which the Android
app shows as well, and by `TestTheScreensAreWithinTheBudgets` for the desktop application
([how to take them](contributing.md#auditing-the-interface)). The test draws 21 screens in
eleven windows: 1280 dp wide in the light and in the dark theme, 700, 412 and 360 dp wide (the
narrowest that the application opens in), and, with the text that the system makes larger, 130 % and
150 % at 360 dp, 150 % at 412 dp, and 200 % at 360, 700 and 1280 dp. It fails when one of them is over
a budget. A dp of the desktop is a px of the web interface. A screen is held to:

| Census | Budget |
| ------ | -----: |
| Texts under the contrast of WCAG AA (4.5:1; 3:1 for large text) | 0 |
| Page wider than its window | no |
| Controls under 24 px | 0 |
| Sizes off the type scale | 0 |
| Rows of a list with two buttons that have words on them | 0 |
| Lists in which three rows or more start with the same icon | 0 |
| Sentences of eight words or more | 3 |
| Small lines of four words or more | 3 |
| Titles and texts of controls that an ellipsis cuts short, in a line or in the lines they may have | 0 |
| Controls and texts that a row of a list holds and that are drawn beyond its padding | 0 |
| Icons that the table of the icons does not have (desktop only) | 0 |

**Larger text.** The text of the system may be larger, up to 200 %, and the budgets hold at every
size in every window. What holds text is as wide as the text: where there is no room for it, a
control goes under the text that it was beside (the end of an item of a list, the actions of the
header of a page and of a dialog), a group of segments gives up its padding and its icons and then
is a dropdown (the tabs, the theme, the language, the filters, the addresses of the pairing), and a chip,
a tag, the name of a choice and the choice that a dropdown shows go on as many lines as they need, and are
not cut by an ellipsis. What has a room that does not grow is not read larger than 130 % (the labels of the
navigation bar).

**Long words.** A word that has no space to break at, a name, an address, a path or a link, and that is
longer than its room, breaks at any character (`kit.Breakable`, which an item of a list does for its
strings): it is not cut with an ellipsis, it does not stick out of what holds it, and what is beside it
goes under it before it is broken, so only a word that does not fit alone is. A box that holds such a
word has to be told it may be narrower than the word (`MinWidth(0)`): the least width of a box in a row is
that of its longest word, and it carries up through every box that holds it.

The widths that follow the text are measured (the segments, a button, the lines of a fingerprint, the room
of an item) or grow with the text scale (a field, the cards of the registry), and none is a width that was
measured once with one font. The census takes 200 % in the narrowest window, in one of the rail and in the
widest, which are the three layouts, and 130 % and 150 % where the room is the least. The layouts do not
promise that what holds at 200 % holds below it, so the sizes in between are looked at and not assumed:
`TestTheScreensAreWithinTheBudgetsAtEverySizeOfText` takes 115, 130, 150 and 175 % in four widths when
`VFOX_PLUS_CENSUS_SWEEP=1` asks for it, and is to be run after a change of a layout.

**Fonts.** Nothing of the layout is written for the width of Ubuntu Sans, which the pictures are in: the
widths follow the text as the font sets it, and text at 200 % is twice as wide as at 100 %, which is
more than the fonts of a system differ by. `TestTheScreensAreWithinTheBudgetsInTheFontsThatTheSystemChooses` draws the scenes in what the system gives
for `sans-serif` and `monospace` (its log says which font that is on the machine that runs it), in
the narrowest window at 100 % and 200 %, and in the widest.

## Motion

Material 3 Expressive replaces durations and easing curves with **springs**.
*Spatial* springs move, resize and reshape and may overshoot; *effects* springs
change colour and opacity and never overshoot. Each comes in three speeds:
fast for small components, default for partial-screen changes, slow for
full-screen changes. All springs have unit mass.

| Spring          | Expressive (damping ratio / stiffness) | Standard |
| --------------- | -------------------------------------: | -------: |
| Fast spatial    |                              0.6 / 800 | 0.9 / 1400 |
| Default spatial |                              0.8 / 380 | 0.9 / 700  |
| Slow spatial    |                              0.8 / 200 | 0.9 / 300  |
| Fast effects    |                               1 / 3800 | 1 / 3800   |
| Default effects |                               1 / 1600 | 1 / 1600   |
| Slow effects    |                                1 / 800 | 1 / 800    |

vfox-plus uses the **expressive** scheme:

| Element                                              | Spring            |
| ---------------------------------------------------- | ----------------- |
| Button, button group, split button and list shapes   | fast spatial      |
| Hover spin of avatars, selection of colour swatches  | fast spatial      |
| Navigation rail expanding and collapsing             | default spatial   |
| Navigation indicator growing, icon fill              | fast spatial / fast effects |
| Dialogs scaling in                                   | default spatial (scale), default effects (fade) |
| Page transitions (fade through, shared axis)         | slow effects      |
| Progress of the wavy indicator, wave amplitude        | default spatial / default effects |
| Loading indicator morph                              | 0.6 / 200, every 650 ms, plus a 4666 ms rotation |

A spring keeps its velocity when its target changes mid-flight, so interrupted
animations stay smooth. Where an API only accepts a duration and a curve
(dialogs), the spring is sampled into a curve over the time it needs to settle;
page transitions, which bring their own curves, last as long as the slow effects
spring needs to settle. When the platform asks for reduced motion, springs jump
to their target and indeterminate indicators stop.

The web interface is such an API: CSS has durations and curves, no springs. Its
`--sp-*` variables are the six springs, each as `--sp-NAME-t`, the milliseconds
that it needs to settle, and `--sp-NAME`, its step response sampled into a
`linear()` curve with as few stops as keep the line within 0.6 % of the curve.
Every `transition` and `animation` takes its time and its curve from them; a
dialog scales in on the default spatial spring while it fades in on the default
effects spring, which makes two animations. A test in `mygo/pair` recomputes the
variables from `kit.Expressive` and fails, printing the lines to paste, when they
differ, and when a declaration has a duration or a curve of its own (the wavy
progress and the loading indicator, which run until the work is done, are the
exception). A browser without `linear()` keeps the Bézier curves that the page
had before. The Android app takes the springs from its theme (`motionSpring*`):
`SpringAnimation` for what moves and a `SpringInterpolator` for the transitions
of its screens.

mygo: the springs and `AnimateTo` in `mygo/ui/kit/motion.go`, drawn only while
they move, so the window is idle when nothing changes.

### Transitions

- Between top-level destinations: **fade through**.
- From the SDK list to an SDK on narrow windows: **shared axis**, horizontal.
- Between SDKs in the detail pane: **fade through**.
- Pages opened from the application's own routes (licenses): **fade forwards**.

### Refresh rate

A frame has 8.3 ms at 120 Hz, and every client keeps within it:

- mygo: the toolkit draws at the refresh rate of the display. Where the CPU draws
  the window (no GPU, as in virtual machines and WSL), a transition that scales
  redraws every pixel of the page in each frame and takes 10 to 40 ms, so pages,
  dialogs and menus only fade there (`kit.Env.Software`, set from
  `Painter.Accelerated`).
- Web interface and the Android app, which shows it in a WebView: animate
  `transform` and `opacity`, which the compositor runs without the main thread
  of the page, and not `left`, `top` or `height`. A long list sets
  `content-visibility: auto`, so that opening its page lays out only what is on
  the screen (the plugin cards took 58 ms to lay out on a phone six times slower
  than a desktop, 16 ms with it).
- Android: many phones keep ordinary windows at 60 Hz, so the app asks for the
  display mode with the highest refresh rate at the current resolution
  (`HighRefreshRate.kt`).

## Components

| Component | Specification | mygo |
| --- | --- | --- |
| Navigation rail (collapsed 96 dp, expanded 240 dp, with menu button and FAB) | [Navigation rail](https://m3.material.io/components/navigation-rail) | `NavRail` in `ui/kit/nav.go` |
| Navigation bar (compact windows) | [Navigation bar](https://m3.material.io/components/navigation-bar) | `NavBar` in `ui/kit/nav.go` |
| Connected button group (single selection) | [Button groups](https://m3.material.io/components/button-groups) | `ButtonGroup` in `ui/kit/group.go` |
| Split button | [Split button](https://m3.material.io/components/split-button) | `Button` in `ui/kit/button.go` |
| Icon buttons (expressive sizes and shapes) | [Icon buttons](https://m3.material.io/components/icon-buttons) | `IconButton` in `ui/kit/button.go` |
| Common buttons with pressed shape morph | [Buttons](https://m3.material.io/components/buttons) | `Button` in `ui/kit/button.go` |
| Loading indicator (contained and uncontained) | [Loading indicator](https://m3.material.io/components/loading-indicator) | `LoadingIndicator` in `ui/kit/loading.go` |
| Wavy linear progress indicator (determinate and indeterminate) | [Progress indicators](https://m3.material.io/components/progress-indicators) | `WavyProgress` in `ui/kit/loading.go` |
| Segmented lists | [Lists](https://m3.material.io/components/lists) | `ListGroup`, `ListItem` in `ui/kit/list.go` |
| Filter chips | [Chips](https://m3.material.io/components/chips) | `FilterChip` in `ui/kit/chip.go` |
| Search bar, cards, dialogs, menus, snack bars, switches, text fields | Material 3 | `SearchBar`, `Card`, `Menu`, dialogs and snack bars in `ui/kit`, themed in `ui/kit/theme.go` |

### Android app

The app builds its screens from the components of Material Components 1.14.0
(Views, not Compose: [why](packaging-android.md#why-views-and-not-compose)),
whose Expressive styles are the defaults of its theme.

| Component | In the app |
| --- | --- |
| Buttons (filled, tonal, text) and icon buttons | `Widget.Material3Expressive.Button*`, which come with the shape morph; a layout only sets the size (`materialSizeOverlay`, Medium) |
| Floating toolbar | `FloatingToolbarLayout` with `Widget.Material3.FloatingToolbar.Vibrant`, on the scanner: the torch and "Enter manually" |
| Cards | filled `MaterialCardView` with 28 dp corners: the command on `surfaceContainerHigh`, the last server on `secondaryContainer` |
| Text field | outlined `TextInputLayout` with the 4 dp corners of the specification; its end icon pastes and clears |
| Loading indicator | `LoadingIndicator` in the place of the Connect button |
| Wavy progress indicator | `Widget.Material3Expressive.LinearProgressIndicator.Wavy`, while the page of the session loads |
| Dialogs | `MaterialAlertDialogBuilder`, in the style of the Expressive theme; the certificate fingerprint sits in a `surfaceContainerHighest` card |
| Shapes | the cookie behind the fox of the pairing screen and the soft burst behind the icon of the error state (`ExpressiveShapes`) |

### Wavy progress indicator

- Track and active indicator: 4 dp thick, rounded ends; a 4 dp gap between them.
- Active indicator colour `primary`, track `secondaryContainer`.
- The wave has an amplitude of 3 dp and travels one wavelength per second. Its
  wavelength is 40 dp when determinate and 20 dp when indeterminate.
- Determinate: the wave flattens below 10 % and above 95 %, and a 2 dp stop
  indicator marks the end of the track.

### Filter chip

- 32 dp high, 12 dp padding, label `labelLarge`. Off: an 8 dp pill with a 1 dp
  `outlineVariant` outline and `onSurfaceVariant` content. On: a
  `secondaryContainer` pill with `onSecondaryContainer` content, no outline, a
  check mark in front.
- The chip morphs with the fast spatial spring when its state changes: the
  corners grow from 8 dp to the full height as the filter comes on, the outline
  fades out, the fill fades in, and a check mark of 18 dp slides in by widening
  the chip; while the chip is pressed its corners pinch to 4 dp.
- A row of chips wraps; the chips of one row are alternatives. The web interface
  draws the same chip in CSS (`.chip`), with the same radii and the same spring
  (`--sp-fast-spatial`).

### Loading indicator

- 48 dp by default; the active indicator is 38/48 of the size.
- Uncontained: `primary`. Contained: `onPrimaryContainer` on a
  `primaryContainer` circle.

## Icons

Icons are [Material Symbols](https://fonts.google.com/icons), **Rounded** style,
at weight 400, grade 0 and optical size 24: the icon of the selected navigation
destination is filled (`FILL` 1) and the others are outlined (`FILL` 0). The
icons and their meanings are listed in
[`shared/icons.json`](../../../shared/icons.json), which also pins where they come
from: the npm package
[`@iconify-json/material-symbols`](https://www.npmjs.com/package/@iconify-json/material-symbols)
(Apache-2.0, 1.3 MB; its version and the SHA-512 of its tarball). `vfxtool prepare`
downloads it once into the cache, checks it and cuts the SVGs of the symbols in use
out of it, `SYMBOL.svg` outlined and `SYMBOL-fill.svg` filled, so that no icon is
committed. mygo draws an SVG in the colour of the text (`ui.Icon`), so the
application carries no icon font; the fill that the navigation and the buttons
animate is the cross-fade of the outlined and the filled icon, and the SVGs
(about 60 KB) are parsed when an icon is first drawn.

The Android app draws the same symbols as vector drawables, which nothing in the
repository holds or generates: they are the Android XML that
[google/material-design-icons](https://github.com/google/material-design-icons)
publishes for each symbol (Apache-2.0, like the repository), and the build imports
them. That repository keeps them in git, not as a library, so `settings.gradle.kts`
has an Ivy repository over its raw files, for the group `symbols` and for nothing
else, and `app/build.gradle.kts` lists the symbols with the commit
(`10ecc14790612ca8fd164e3f4ffe9c16fc50d5b7`, of 2026-10-23) as
dependencies, `symbols:SYMBOL:COMMIT@xml`, with the classifier `fill1` for a filled
one: the files `symbols/android/SYMBOL/materialsymbolsrounded/SYMBOL_24px.xml` and
`SYMBOL_fill1_24px.xml`. Gradle fetches them once into its cache, which a build
offline uses afterwards, and a task for each variant (`copyMaterialSymbolsRelease`,
`copyMaterialSymbolsDebug`) puts them, unchanged, as `ic_NAME.xml` among the generated
resources that the Android plugin merges with the others
(`app/build/generated/res/copyMaterialSymbolsRelease/drawable`). A file has the colour of the
theme (`colorControlNormal`) unless a view tints it, and Google marks the symbols that
turn around in a right-to-left language (`autoMirrored`, here `copy`, `forward` and
`help`). The app uses `ic_close` (`close`), `ic_copy` (`content_copy`), `ic_forward` (`arrow_forward`), `ic_help` (`help`), `ic_history` (`history`), `ic_keyboard` (`keyboard`), `ic_link` (`link`), `ic_lock` (`lock`), `ic_paste` (`content_paste`), `ic_retry` (`replay`), `ic_scan_qr` (`qr_code_scanner`), `ic_torch_off` (`flashlight_off`), `ic_torch_on_fill` (`flashlight_on`, filled) and `ic_unreachable` (`cloud_off`). They and the npm package above are two copies of one
set, Material Symbols Rounded at weight 400 and 24 px, so the interfaces look the
same.

### Application icon

The application icon is the fox of the vfox logo, redrawn in the Material 3
Expressive shape language
([`shared/packaging/linux/dev.vfox.VfoxPlus.svg`](../../../shared/packaging/linux/dev.vfox.VfoxPlus.svg)).

- **Character.** The same fox as the avatar of the version-fox organisation:
  an orange face with cream cheeks that form a "V", cream inner ears, a dark
  nose and round dark eyes with a highlight. It has no outline, and its corners
  are rounded with large radii, like the shapes of the library. The two ears and
  the two cheek tufts make a silhouette that is still a fox at 16 px.
- **Plate and halo.** An indigo plate, the colour opposite the fur, and behind the
  fox the 9-sided cookie of the shape library as a soft halo.
- **Colour.** Fur `#FFB63F` to `#FF8419` (top to bottom), cheeks `#FFFFFF` to
  `#FFE7CB`, inner ears `#FFE3C4`, eyes and nose `#17123F`; plate `#5444E4` to
  `#1B1268` (top left to bottom right); halo white at 12 % to 3 %.
- **Legibility.** One large, simple shape without text. Measured on the drawing,
  the fur is at least 3.2:1 against what is behind it (WCAG asks 3:1 for
  graphics), the cheeks at least 5.9:1, and the eyes against the fur at least
  7.1:1. The fox is two thirds of the plate on the desktop, and on Android it
  stays inside the 66 dp safe zone of the adaptive icon, so that circle, squircle and
  rounded square masks show it whole.

One drawing serves every platform:

| Platform | Files | Notes |
| --- | --- | --- |
| Linux | [`shared/packaging/linux/dev.vfox.VfoxPlus.svg`](../../../shared/packaging/linux/dev.vfox.VfoxPlus.svg) | The master, on a squircle plate. `vfxtool build-deb` installs it as the scalable icon. The AppStream metadata next to it names the colours that software centres put behind the icon: `#FFDCC1` for the light and `#6B3B04` for the dark theme, the primary containers of the colour scheme of the vfox orange (the `--m3-primary-container` of the web interface). |
| Android launcher | `ic_launcher_background.xml`, `ic_launcher_foreground.xml` and `ic_launcher_monochrome.xml` in [`android/app/src/main/res/drawable`](../../../android/app/src/main/res/drawable) | The adaptive icon: the launcher masks the full-bleed plate and the fox. The monochrome layer, for the themed icons of Android 13, has the cheeks at half strength and the eyes and the nose cut out. |
| Android pairing screen | `ic_mark.xml` in the same directory | The master as a vector drawable. |
| Windows | `Assets/*.png` in the `.msix` | Drawn by `vfxtool build-msix` from the same paths ([`shared/tooling/internal/appicon`](../../../shared/tooling/internal/appicon)), because no image is committed. |
| Web interface | [`mygo/pair/favicon.svg`](../../../mygo/pair/favicon.svg) | The plate and the fox without the halo, and the fox larger, which still reads at 16 px. Every page of the pairing server inlines it as a data URI (`favicon.go`), so that the pages make no request for an icon; the web interface shows it as the brand mark of its navigation rail, and its web app manifest names it as the icon of the installed application (an SVG icon of the size `any` is one that Chromium accepts for installing). |

The SVG is the master: the vector drawables, `appicon` and the favicon repeat its
paths, and `go test` in `shared/tooling` (the first two) and in `mygo/pair` (the
favicon) fails when they differ.

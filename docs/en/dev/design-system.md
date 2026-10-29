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
  statistic tiles and avatars.
- Destructive actions use `error`.

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
| extra large            |  28 dp | Page panes, dialogs, statistic tiles             |
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
- **Statistic tiles** show a large shape partly hidden behind their edge.
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
titles, numbers in statistic tiles, list titles that need attention (the current
version) and card titles.

| Role                       | Baseline weight | Emphasized weight |
| -------------------------- | --------------: | ----------------: |
| Display, headline, title L |             400 |               500 |
| Title M, title S           |             500 |               700 |
| Body L, body M             |             400 |               500 |
| Label L, label M           |             500 |               700 |

mygo: `Emph` in `mygo/ui/kit/theme.go`.

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
| Search bar, cards, dialogs, menus, snack bars, switches, text fields | Material 3 | `SearchBar`, `Card`, `Menu`, dialogs and snack bars in `ui/kit`, themed in `ui/kit/theme.go` |

### Wavy progress indicator

- Track and active indicator: 4 dp thick, rounded ends; a 4 dp gap between them.
- Active indicator colour `primary`, track `secondaryContainer`.
- The wave has an amplitude of 3 dp and travels one wavelength per second. Its
  wavelength is 40 dp when determinate and 20 dp when indeterminate.
- Determinate: the wave flattens below 10 % and above 95 %, and a 2 dp stop
  indicator marks the end of the track.

### Loading indicator

- 48 dp by default; the active indicator is 38/48 of the size.
- Uncontained: `primary`. Contained: `onPrimaryContainer` on a
  `primaryContainer` circle.

## Icons

Icons are [Material Symbols](https://fonts.google.com/icons), **Rounded** style,
from the variable font, so that they can be animated along their axes: the
icon of the selected navigation destination is filled (`FILL` 1) and the others
are outlined (`FILL` 0). The icons and their meanings are listed in
[`shared/icons.json`](../../../shared/icons.json). The application ships only these
glyphs: `vfxtool iconfont` (`shared/tooling/internal/iconfont`) cuts the
same glyphs, with their ligatures, out of a pinned version of the font at
build time; pinning the weight, grade and optical size and keeping only the
fill axis gives a 9 KB WOFF2 font. mygo cannot set the fill axis of a font, so
it ships the subset twice — outlined and filled under another family name —
and cross-fades between the two (36 KB together).

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
| Web interface | [`mygo/pair/favicon.svg`](../../../mygo/pair/favicon.svg) | The plate and the fox without the halo, and the fox larger, which still reads at 16 px. Every page of the pairing server inlines it as a data URI (`favicon.go`), so that the pages make no request for an icon. |

The SVG is the master: the vector drawables, `appicon` and the favicon repeat its
paths, and `go test` in `shared/tooling` (the first two) and in `mygo/pair` (the
favicon) fails when they differ.

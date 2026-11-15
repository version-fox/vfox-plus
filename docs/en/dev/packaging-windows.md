# Windows packaging (MSIX)

> [简体中文](../../zh-CN/dev/packaging-windows.md)

The Windows packaging of vfox+ is a few commands of `vfxtool`. `vfxtool build-msix`
turns staged Windows executables into `.msix` packages (one per architecture) and
a `.msixbundle` for **Windows 11**. The release signs them in CI with Artifact
Signing, whose certificate Windows trusts ([Signing the release](#signing-the-release)):
`vfxtool rebundle` puts the signed packages into the bundle again, and
`vfxtool install-check --trusted` installs the bundle the way a user does.
`vfxtool sign-msix` signs with a certificate of its own, to test the signed path
(see [Signing the packages yourself](#signing-the-packages-yourself)).
`shared/packaging/windows/` holds only a pointer to this page.

The package identity, the manifest, the logos and the bundle layout are generated
by `vfxtool build-msix` (`shared/tooling/internal/buildmsix`); nothing binary is
committed here. The logos are the application icon, drawn at build time from the
paths of its SVG by `shared/tooling/internal/appicon` (see the
[design system](design-system.md#application-icon)).

## What is built

For each architecture (`x64` for AMD64, `arm64` for ARM64):

- `vfox-plus_<version>_<arch>.msix`: an OPC ZIP with `AppxManifest.xml`,
  `[Content_Types].xml`, `AppxBlockMap.xml`, the executable (for example
  `vfox-plus.exe`) and the `Assets/` logos.
- `vfox-plus_<version>.msixbundle`: the regular bundle containing the `.msix`
  files, with `AppxMetadata/AppxBundleManifest.xml`.

The identity defaults to `dev.vfox.VfoxPlus` (the same as the Linux desktop
entry). The publisher is set by `--publisher`: a release passes the subject of
its signing certificate. Without it the publisher is `CN=vfox-plus` with the
unsigned-package OID (`OID.2.25.311729368913984317654407730594956997722=1`),
which Windows 11 requires for `-AllowUnsigned` installs; CI builds these unsigned
packages to test them. The target device family is `Windows.Desktop` with
`MinVersion 10.0.22000.0` (Windows 11).

## Building

Cross-compile on any OS (mygo needs no C compiler), then package:

```sh
vfxtool build --goos windows --goarch amd64 --root mygo stage-x64/vfox-plus.exe
vfxtool build --goos windows --goarch arm64 --root mygo stage-arm64/vfox-plus.exe
SOURCE_DATE_EPOCH="$(git log -1 --format=%ct)" vfxtool build-msix \
  --package vfox-plus --version "$(tr -d '[:space:]' < mygo/VERSION)" \
  --stage-x64 stage-x64 --stage-arm64 stage-arm64 \
  --executable vfox-plus.exe dist
```

Only one `--stage-*` is needed for a single-architecture bundle. The output is
reproducible: ZIP entries are stored (no compression), sorted, with timestamps
from `SOURCE_DATE_EPOCH`. The exception is `[Content_Types].xml`, which is a
DEFLATE entry made of stored blocks (still not compressed, so no compressor
changes the bytes). The archives are ZIP64 archives, as MakeAppx writes them.
See [Signing notes](#signing-notes-for-maintainers) for why.

## Installing on Windows 11

Download `vfox-plus_<version>.msixbundle` from the GitHub release, then install it
in an elevated PowerShell:

```powershell
Add-AppxPackage -Path .\vfox-plus_<version>.msixbundle
```

The release bundle is signed with a certificate that Windows trusts, so it needs
no `-AllowUnsigned` and no step on the machine. The deployment platform picks the
slice matching the machine, and the package installs for all users. To remove it
again:

```powershell
Remove-AppxPackage -Package dev.vfox.VfoxPlus_<version>_neutral_~_<publisher-id>
```

(Get the full package name with `Get-AppxPackage -Name dev.vfox.VfoxPlus`.)

A version from before the signed release, unsigned or self-signed, has another
publisher, so it is another package and stays installed beside the signed one.
Remove it once with the command above.

The `vfox-plus-windows` artifact of a CI run holds the unsigned bundle next to the
staged executables and a Windows `vfxtool.exe`. Its packages are for tests: they
install with `Add-AppxPackage -AllowUnsigned` only.

To run from source without packaging, build on Windows and run the exe directly.

## Signing the release

The `sign-windows` job of the [release workflow](contributing.md#release) signs
the `.msix` files and the `.msixbundle` with
[Artifact Signing](https://learn.microsoft.com/azure/artifact-signing/), formerly
Trusted Signing, through
[`azure/artifact-signing-action`](https://github.com/Azure/artifact-signing-action).
The job logs in to Azure with OpenID Connect, so no certificate and no key is
stored in the repository or its secrets. It does these steps, in this order:

1. It signs each `.msix` with the certificate of the certificate profile.
2. It puts the signed packages into the bundle again (`vfxtool rebundle`): the
   signatures changed their sizes and offsets, which the bundle names.
3. It signs the bundle.
4. It installs the bundle with `vfxtool install-check --trusted`, without
   `-AllowUnsigned`, and removes it again. Windows accepts the install only when
   the signature and the publisher hold, and the certificates of the runner are
   left alone.

The packages are timestamped by the service, so their signatures stay valid after
the certificate expires; these certificates last only a few days.

The setup is done once, by a maintainer with the Azure subscription:

1. An Artifact Signing account, with a certificate profile of the public trust
   kind. The identity validation that the service asks for comes first, and the
   certificate names the identity that it validated.
2. An identity for the release in Microsoft Entra ID: an app registration, with a
   federated credential for GitHub Actions. Its organization is `version-fox`, its
   repository `vfox-plus`, its entity type **Environment** and its environment
   `release`. The subject is then `repo:version-fox/vfox-plus:environment:release`,
   which only the jobs of the `release` environment have.
3. The role **Artifact Signing Certificate Profile Signer** for that identity, on
   the account.
4. The secrets of the `release` environment are the client ID, the tenant ID and
   the subscription ID of that identity:

   ```sh
   gh secret set AZURE_CLIENT_ID --env release -R version-fox/vfox-plus
   gh secret set AZURE_TENANT_ID --env release -R version-fox/vfox-plus
   gh secret set AZURE_SUBSCRIPTION_ID --env release -R version-fox/vfox-plus
   ```

5. The variables of the repository (`gh variable set NAME -R version-fox/vfox-plus`):
   - `ARTIFACT_SIGNING_ENDPOINT`: the endpoint of the region of the account, for
     example `https://eus.codesigning.azure.net/`;
   - `ARTIFACT_SIGNING_ACCOUNT`: the name of the account;
   - `ARTIFACT_SIGNING_PROFILE`: the name of the certificate profile;
   - `WINDOWS_PUBLISHER`: the subject of the certificate, exactly as it is, in the
     form `CN=..., O=..., L=..., S=..., C=...`. The build fails while it is empty.
     The `sign-windows` job prints the subject of the signed bundle (step "Show the
     publisher of the signed bundle"): when the install then refuses the bundle,
     the log shows what to put here.

A release waits for two approvals of the `release` environment: before the
signing, and before the publishing. Start a **dry run** first, with the `dry-run`
input of Actions, Release, Run workflow: it builds, signs and installs the
bundle, and publishes nothing. Only a run without it pushes, tags and publishes.

Notes:

- The first signed release has another publisher than the packages before it, so
  it installs beside them: see [Installing on Windows 11](#installing-on-windows-11).
- A browser download of a new file gets its own SmartScreen check. A new release
  can meet a warning for a while, although its signature is valid.

## Signing the packages yourself

`vfxtool sign-msix` signs the bundle with a certificate that it makes and installs
it, for a machine where the bundle should install without `-AllowUnsigned`, and for
whoever tests the signed path. It takes the `vfox-plus-windows` artifact of a CI run
(`mygo.yml`): the unsigned packages, the staged executables in `stage-x64/` and
`stage-arm64/`, and a Windows `vfxtool.exe`. Run it in that folder:

```powershell
.\vfxtool.exe sign-msix
```

It does this, in this order:

1. It downloads osslsigncode 2.14 into `tools/`, pinned and checked by its SHA-256,
   so that no Windows SDK is needed. `--osslsigncode PATH` takes one that you have
   instead, and `--force` downloads again.
2. It rebuilds the packages into `signed/` with the publisher `CN=vfox-plus`
   (`--publisher-cn`) instead of the OID of unsigned packages, because the publisher
   of a signed package must equal the subject of the certificate.
3. It makes a self-signed code-signing certificate for that subject (RSA 2048,
   valid for five years) and signs every `.msix`, then puts the signed packages into
   the bundle again and signs the bundle, as SignTool does.
4. It trusts the certificate for the local machine, in its `TrustedPeople` store,
   which is what the sideloading of a signed package looks at, and takes out the
   certificate that an earlier run added.
5. It installs the bundle with `Add-AppxPackage` (`--skip-install` stops before).

The private key lives for one run: it goes to a temporary file for osslsigncode and
is deleted after the signing, never kept. The certificate that stays trusted on the
machine can therefore sign nothing any more, and there is no key to protect. The
price is that each run makes a new certificate, so a bundle that an earlier run
signed does not install any more: sign again and install the new one.

Trusting a certificate and installing a package need administrator rights. In a
shell that is not elevated the command asks Windows to start a copy of itself with
them: you answer the UAC prompt, and the copy works in a window of its own, which
waits for Enter at the end so that you can read what it did. In an elevated shell
it simply runs; `--no-relaunch` makes it fail instead of asking, for scripts. Other
systems have no store and no installation, so there the command only signs: pass
`--skip-install` and an `--osslsigncode` of that system.

To take it away again, remove the package (see above) and the certificate:

```powershell
Get-ChildItem Cert:\LocalMachine\TrustedPeople |
  Where-Object Subject -eq "CN=vfox-plus" | Remove-Item
```

The `validate-windows` job of [`mygo.yml`](../../../.github/workflows/mygo.yml) runs
the command on Windows Server 2025 (x64) and Windows 11 on ARM (arm64): it signs,
installs without `-AllowUnsigned`, checks the architecture, uninstalls, runs a
second time and checks that exactly one certificate is trusted. The elevated copy
is the one path that CI cannot take, because its runners are administrators with
UAC off; it needs a run on a machine with UAC after a change.

The `run-windows` job runs the application itself on the same two runners, with the
end-to-end build for Windows that the `build` job cross-compiles: the scenario
against the fake vfox, and the icon in the tray (see
[The tray](architecture.md#the-tray); the runner for ARM has a shell that takes no
icon, so the place of the icon is checked on Windows Server 2025 only).

## Signing notes (for maintainers)

- A signed package's publisher must exactly equal the certificate subject, with no
  OID. The release builds the packages with that publisher from the start
  (`--publisher`). `sign-msix` rebuilds them, since its packages carry the
  unsigned-package OID.
- The signature of a bundle covers only its manifest (the block map of a bundle
  lists nothing else), so the packages in it carry their own signatures, and the
  manifest names them by offset and size. A bundle of unsigned packages is refused
  even when it is signed itself. So the order is: sign the packages, make the bundle
  again from them (`vfxtool rebundle`, which refuses a package without a signature),
  then sign the bundle.
- osslsigncode rewrites `[Content_Types].xml` to add the signature part, and it always
  writes the new content deflated while keeping the method that the entry had. A
  stored entry therefore came out as a corrupt one (the method Store, a compressed
  size below the size), which its own `verify` accepts and the deployment platform of
  Windows does not: `Add-AppxPackage` fails with `0x8007000D`, "Opening the package
  ... failed". That is why `build-msix` writes that one part as a DEFLATE entry of
  stored blocks, like MakeAppx does, and why the test of `sign-msix` reads every entry
  of what osslsigncode signed to its end, which a signature check alone does not.
- Windows refuses a signed package that is a plain ZIP32 archive, although the same
  package installs unsigned: `Add-AppxPackage` fails with `0x8007000B`, "Opening the
  package ... failed". MakeAppx and SignTool write every package as a ZIP64 archive,
  however small it is: the classic end record holds only the sentinels `0xFFFF` and
  `0xFFFFFFFF`, the ZIP64 end record and its locator come before it, and the central
  directory says "made by 4.5" for every entry, which need only 2.0 and have no ZIP64
  extra field (a package with one is refused as well). `build-msix` writes that shape,
  `CheckZip64` checks it (`validate-msix` too), and osslsigncode keeps it. The shape
  comes from the signed test packages of
  [microsoft/msix-packaging](https://github.com/microsoft/msix-packaging), and from the
  measurements of [openappx](https://github.com/gianlucamazza/openappx) on an Xbox: a
  signed ZIP32 package is refused, a signed ZIP64 one installs.

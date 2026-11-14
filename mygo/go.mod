module github.com/version-fox/vfox-plus/mygo

go 1.27.4

require (
	github.com/ebitengine/purego v0.11.3
	github.com/egoist/mygo v0.9.0
	// Encodes the pairing link of the settings as a QR code for the Android
	// app to scan (MIT; see docs/en/dev/architecture.md#dependencies).
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
	// Cross-process task lock on Windows, where the standard syscall
	// package has no file locking (BSD; same section of the architecture).
	golang.org/x/sys v0.50.0
)

require (
	github.com/go-text/typesetting v0.3.5 // indirect
	golang.org/x/image v0.47.0 // indirect
	golang.org/x/net v0.62.0 // indirect
	golang.org/x/text v0.44.0 // indirect
)

module github.com/version-fox/vfox-plus/mygo

go 1.27.1

require (
	github.com/ebitengine/purego v0.11.2
	github.com/egoist/mygo v0.6.3
	// Prints the pairing link of `vfox-plus serve` as a terminal QR code
	// for the Android app (MIT; see docs/en/dev/architecture.md#dependencies).
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
	// Cross-process task lock on Windows, where the standard syscall
	// package has no file locking (BSD; same section of the architecture).
	golang.org/x/sys v0.49.0
)

require (
	github.com/go-text/typesetting v0.3.5 // indirect
	golang.org/x/image v0.47.0 // indirect
	golang.org/x/net v0.61.0 // indirect
	golang.org/x/text v0.43.0 // indirect
)

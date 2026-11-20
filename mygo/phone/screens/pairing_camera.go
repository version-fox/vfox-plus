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

package screens

import (
	"context"
	"errors"
	"os"
	"sync"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/camera"
	"github.com/egoist/mygo/plugins/imagepicker"
	"github.com/egoist/mygo/plugins/permissionflow"
	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/pairing"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// pairReading is what the pairing page keeps for reading a pairing code from a photo
// or from a picture. The page keeps it with ui.Local, so it lasts as long as the page.
// The goroutines of the reads set its results under mu, and the page takes them on
// the main thread at its next frame. The camera and its preview belong to the main
// thread alone.
type pairReading struct {
	mu sync.Mutex
	// busy is a read in progress: the page starts no other.
	busy bool
	// note is the key of the message about the last read, shown under the field, or "".
	note string
	// permission is the answer to the request for the camera, which the page takes
	// once: pairPermissionGranted or pairPermissionDenied.
	permission int
	// link is the pairing link that a read found, which the page takes once.
	link string

	// camOn says that the camera is shown; camera is its preview while it is, and nil
	// otherwise. Both are main-thread state.
	camOn  bool
	camera *camera.Camera
}

// The answers to the request for the camera.
const (
	pairPermissionNone = iota
	pairPermissionGranted
	pairPermissionDenied
)

// pairReadingOf returns the reads of the page that element draws, which the page keeps.
func pairReadingOf(element ui.Element) *pairReading {
	return *ui.Local(element, "pairReading", func() *pairReading { return &pairReading{} })
}

// start marks a read as running. It reports false, and changes nothing, when a read is
// running already.
func (p *pairReading) start() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.busy {
		return false
	}
	p.busy, p.note = true, ""
	return true
}

// finish ends a read with the link it found, or with the key of the message that says
// why it found none. The link, when there is one, is the note's reason to be empty.
func (p *pairReading) finish(link, note string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.busy = false
	if link != "" {
		p.link, p.note = link, ""
		return
	}
	p.note = note
}

// answer records the answer to the request for the camera. A failed request is a
// note, and any other answer lets the camera be opened: the camera asks for itself
// when the system must.
func (p *pairReading) answer(status permissionflow.Status, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	switch {
	case err != nil:
		p.note = "pairCameraUnavailable"
	case status == permissionflow.Denied:
		p.permission = pairPermissionDenied
	default:
		p.permission = pairPermissionGranted
	}
}

// take returns the answer to the request for the camera and the link that a read
// found, and forgets them: each reaches the page once.
func (p *pairReading) take() (int, string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	permission, link := p.permission, p.link
	p.permission, p.link = pairPermissionNone, ""
	return permission, link
}

// current returns whether a read is running, and the key of the note under the field.
func (p *pairReading) current() (bool, string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.busy, p.note
}

// setNote shows the message of key under the field. It is for the main thread.
func (p *pairReading) setNote(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.note = key
}

// pairStartCamera asks for the camera and opens its preview, when the system lets the
// phone use it. The request waits for the user, so it runs in the background; the
// preview is opened by the page at its next frame. It runs on the main thread.
func (r *Root) pairStartCamera(p *pairReading) {
	status, err := permissionflow.Check(permissionflow.Camera)
	switch {
	case errors.Is(err, errors.ErrUnsupported):
		// The system keeps no permission for apps on this system, as on a desktop.
		p.camOn = true
	case err != nil:
		p.setNote("pairCameraUnavailable")
	case status == permissionflow.Granted:
		p.camOn = true
	case status == permissionflow.Denied:
		r.pairCameraDenied()
	default:
		// The system has not asked yet, or cannot tell: the request asks it.
		r.Do(func(ctx context.Context) error {
			status, err := permissionflow.Request(ctx, permissionflow.Camera)
			p.answer(status, err)
			r.invalidate()
			return nil
		}, "")
	}
}

// pairCameraPage applies what the reads found since the last frame, shows the preview
// of the camera while it is on, and draws its shutter. It runs on the main thread,
// every frame.
func (r *Root) pairCameraPage(k kit.Ctx, p *pairReading) {
	permission, link := p.take()
	switch permission {
	case pairPermissionGranted:
		p.camOn = true
	case pairPermissionDenied:
		r.pairCameraDenied()
	}
	if link != "" {
		p.camOn = false
		r.pairConnectText(link)
	}
	if p.camOn && p.camera == nil {
		p.camera = pairOpenCamera(p)
		p.camOn = p.camera != nil
	}
	if !p.camOn && p.camera != nil {
		p.camera.Destroy()
		p.camera = nil
	}
	if p.camera == nil {
		return
	}
	busy, _ := p.current()
	preview := p.camera
	ui.Box(k.Context).Height(240).Shrink(0).FillWidth().Children(func() {
		ui.WebView(k.Context, preview).Fill()
		ui.Row(k.Context).Absolute().Left(8).Right(8).Bottom(8).Gap(8).Children(func() {
			kit.Button(k, kit.ButtonSpec{
				ID:       "shoot",
				Kind:     kit.FilledButton,
				Label:    i18n.Text("pairShoot"),
				Icon:     "qrCode",
				Disabled: busy,
				OnClick:  func() { r.pairTakePhoto(p) },
			}).Grow(1)
			kit.Button(k, kit.ButtonSpec{
				ID:      "stopCamera",
				Kind:    kit.TextButton,
				Label:   i18n.Text("cancel"),
				OnClick: func() { p.camOn = false },
			})
		})
	})
}

// pairOpenCamera makes the preview of the camera in the window of the app. It returns
// nil, and says so in the note, when the phone has no camera to use.
func pairOpenCamera(p *pairReading) *camera.Camera {
	windows := mygo.Windows()
	if len(windows) == 0 {
		p.setNote("pairCameraUnavailable")
		return nil
	}
	cam, err := camera.New(windows[0], camera.Options{})
	if err != nil {
		p.setNote("pairCameraUnavailable")
		return nil
	}
	return cam
}

// pairTakePhoto takes a photo with the camera, and reads the pairing code of it, in the
// background. It runs on the main thread.
func (r *Root) pairTakePhoto(p *pairReading) {
	cam := p.camera
	if cam == nil || !p.start() {
		return
	}
	r.Do(func(ctx context.Context) error {
		path, err := cam.TakePhoto(ctx)
		var link, note string
		if err != nil {
			note = "pairPhotoFailed"
		} else {
			link, note = pairReadCode(path)
		}
		p.finish(link, note)
		r.invalidate()
		return nil
	}, "")
}

// pairPickPicture chooses a picture from the library of the phone, and reads the
// pairing code of it, in the background. It runs on the main thread.
func (r *Root) pairPickPicture(p *pairReading) {
	if !p.start() {
		return
	}
	r.Do(func(ctx context.Context) error {
		paths, err := imagepicker.Pick(ctx, imagepicker.Options{Kind: imagepicker.Images, Limit: 1})
		var link, note string
		switch {
		case err != nil:
			note = "pairPickFailed"
		case len(paths) == 0:
			// The user went back, and chose nothing.
		default:
			link, note = pairReadCode(paths[0])
		}
		p.finish(link, note)
		r.invalidate()
		return nil
	}, "")
}

// pairReadCode returns the pairing link of the picture at path, and removes the copy of
// the picture: the camera and the picker keep their copies in the app's temporary
// directory. When there is no link, it returns the key of the message that says why.
func pairReadCode(path string) (string, string) {
	defer func() { _ = os.Remove(path) }()
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "pairPhotoFailed"
	}
	text, err := pairing.DecodeQR(data)
	switch {
	case err == nil:
		return text, ""
	case errors.Is(err, pairing.ErrNotPicture):
		return "", "pairNotPicture"
	case errors.Is(err, pairing.ErrPictureTooLarge):
		return "", "pairPictureTooLarge"
	}
	return "", "pairNoQRCode"
}

// pairCameraDenied asks the user to allow the camera in the system's settings, or to
// choose a picture of the code instead. The dialog opens the settings page of the
// system; it does not wait for the user to come back.
func (r *Root) pairCameraDenied() {
	r.env.ShowDialog("pairCameraDenied", func(c kit.Ctx, close func()) kit.DialogSpec {
		return kit.DialogSpec{
			Icon:      "warning",
			IconError: true,
			Title:     i18n.Text("pairCameraTitle"),
			Content: func() {
				kit.Label(c, i18n.Text("pairCameraBody"), kit.Type(kit.BodyMedium), c.Colors().OnSurfaceVariant)
			},
			Actions: []func(){
				func() {
					kit.Button(c, kit.ButtonSpec{ID: "cameraCancel", Kind: kit.TextButton, Label: i18n.Text("cancel"), OnClick: close})
				},
				func() {
					kit.Button(c, kit.ButtonSpec{
						ID:    "cameraSettings",
						Kind:  kit.FilledButton,
						Label: i18n.Text("openSettings"),
						Icon:  "settings",
						OnClick: func() {
							_ = permissionflow.OpenSettings(permissionflow.Camera)
							close()
						},
					})
				},
			},
		}
	})
}

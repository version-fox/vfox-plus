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

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// homeFindingsShown is how many findings of the health check the home tab lists.
const homeFindingsShown = 5

// The kinds of finding that have an action on the home tab: a SDK's receipt that is
// missing, which is adopted, and a SDK whose files differ from its receipt, which is
// installed again.
const (
	homeFindingNoReceipt = "sdk.no-receipt"
	homeFindingDrift     = "sdk.drift"
)

// homeHealth is the health of vfox and of the installed SDKs: the first findings of the
// check, and the action of the two kinds of finding that have one. A check that failed
// says so in red.
func (r *Root) homeHealth(k kit.Ctx) {
	cs := k.Colors()
	homeSection(k, i18n.Text("doctorTitle"))
	report := r.service.Doctor("", "")
	value, loaded := report.Value()
	switch {
	case report.Err() != nil:
		kit.Label(k, noticeText(app.KindOf(report.Err())), kit.Type(kit.BodyMedium), cs.Error)
		return
	case !loaded:
		kit.Label(k, i18n.Text("loading"), kit.Type(kit.BodyMedium), cs.OnSurfaceVariant)
		return
	case len(value.Findings) == 0:
		kit.Label(k, i18n.Text("doctorNoProblems"), kit.Type(kit.BodyMedium), cs.OnSurfaceVariant)
		return
	}
	findings := value.Findings
	if len(findings) > homeFindingsShown {
		findings = findings[:homeFindingsShown]
	}
	var items []kit.ListItemSpec
	for i, finding := range findings {
		items = append(items, r.homeFindingItem(k, i, finding))
	}
	kit.ListGroup(k, items...)
}

// homeFindingItem is one finding of the health check: what it is about, its severity,
// and the action that fixes it, when it has one.
func (r *Root) homeFindingItem(k kit.Ctx, index int, finding remote.DoctorFinding) kit.ListItemSpec {
	title := finding.Message
	if title == "" {
		title = finding.ID
	}
	subtitle := finding.Severity
	if subject := homeSubject(finding); subject != "" {
		subtitle = subject + " · " + finding.Severity
	}
	item := kit.ListItemSpec{
		ID:       index,
		Title:    title,
		Subtitle: subtitle,
	}
	switch finding.ID {
	case homeFindingNoReceipt:
		item.Trailing = func() {
			kit.Button(k, kit.ButtonSpec{
				ID:    "adopt",
				Kind:  kit.TonalButton,
				Label: i18n.Text("doctorAdopt"),
				OnClick: func() {
					r.homeStart(remote.TaskRequest{Operation: remote.RequestDoctorAdopt, Sdk: finding.Sdk, Version: finding.Version})
				},
			})
		}
	case homeFindingDrift:
		item.Trailing = func() {
			kit.Button(k, kit.ButtonSpec{
				ID:    "reinstall",
				Kind:  kit.TonalButton,
				Label: i18n.Text("versionReinstall"),
				OnClick: func() {
					r.homeStart(remote.TaskRequest{Operation: remote.RequestInstall, Sdk: finding.Sdk, Version: finding.Version, Reinstall: true})
				},
			})
		}
	}
	return item
}

// homeSubject names what a finding is about: the SDK, with its version after an @.
func homeSubject(finding remote.DoctorFinding) string {
	switch {
	case finding.Sdk != "" && finding.Version != "":
		return finding.Sdk + "@" + finding.Version
	case finding.Sdk != "":
		return finding.Sdk
	}
	return finding.Version
}

// homeStart queues a task that a finding asked for, in the background. The task is
// shown by the tasks tab.
func (r *Root) homeStart(request remote.TaskRequest) {
	r.Do(func(ctx context.Context) error {
		_, err := r.service.StartTask(ctx, request)
		return err
	}, "statusQueued")
}

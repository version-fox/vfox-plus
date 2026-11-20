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

package app

// The reads of the service. Each returns a snapshot of its resource, and starts the
// load of the resource when the resource is due (see load); the window calls them
// every frame and does not decide when to load. When the phone has no computer, a
// read returns what is there, which is nothing, and makes no request.
//
// Cache keys. A resource with no arguments is one value. The others are kept by
// their arguments: a task and its log by the task's ID; a doctor report by the SDK
// and the version it checks; a profile, the aliases and the plugin's information by
// the plugin's name; the versions by the name and the search. Each cache keeps a
// few entries and forgets the least recently read.

import (
	"context"
	"errors"

	"github.com/version-fox/vfox-plus/mygo/phone/pairing"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
)

// Inventory returns vfox, the installed SDKs, the plugin registry and the vfox
// configuration. A change of the configuration or of the installed SDKs, and an
// event of vfox, mark it out of date.
func (s *Service) Inventory() Resource[remote.Inventory] {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c := s.client; c != nil {
		load(s, c, s.inventory, c.Inventory)
	}
	return s.inventory.res
}

// Tasks returns the tasks, with the newest lines of their logs. Any task event,
// and every change that the phone makes to a task, marks it out of date.
func (s *Service) Tasks() Resource[[]remote.Task] {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c := s.client; c != nil {
		load(s, c, s.tasks, c.Tasks)
	}
	return s.tasks.res
}

// Task returns one task with the newest lines of its log. Its key is the ID of the
// task. A task event marks it out of date.
func (s *Service) Task(id int) Resource[remote.Task] {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.task.at(id)
	if c := s.client; c != nil {
		load(s, c, e, func(ctx context.Context) (remote.Task, error) {
			return c.Task(ctx, id)
		})
	}
	return e.res
}

// TaskLog returns the complete output of a failed task, which the computer keeps for
// a while. A missing output is a failed resource whose error is ErrNotFound. Its key
// is the ID of the task.
func (s *Service) TaskLog(id int) Resource[string] {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.logs.at(id)
	if c := s.client; c != nil {
		load(s, c, e, func(ctx context.Context) (string, error) {
			return c.TaskLog(ctx, id)
		})
	}
	return e.res
}

// Settings returns the settings of the desktop application.
func (s *Service) Settings() Resource[remote.AppSettings] {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c := s.client; c != nil {
		load(s, c, s.settings, c.Settings)
	}
	return s.settings.res
}

// Web returns how other devices connect to the computer. An event of the web marks
// it out of date.
func (s *Service) Web() Resource[remote.WebState] {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c := s.client; c != nil {
		load(s, c, s.web, c.Web)
	}
	return s.web.res
}

// Doctor returns the health of vfox and of the installed SDKs; empty sdk and version
// ask for all of them. Its key is the pair. A change of the installed SDKs or an
// event of vfox marks it out of date.
func (s *Service) Doctor(sdk, version string) Resource[remote.DoctorReport] {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.doctors.at(doctorKey{sdk: sdk, version: version})
	if c := s.client; c != nil {
		load(s, c, e, func(ctx context.Context) (remote.DoctorReport, error) {
			return c.Doctor(ctx, sdk, version)
		})
	}
	return e.res
}

// Profile returns the profile of the plugin name, which is nil for a plugin that has
// none. A change of the plugin's search or variables, or of the installed SDKs,
// marks it out of date.
func (s *Service) Profile(name string) Resource[*remote.Profile] {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.profiles.at(name)
	if c := s.client; c != nil {
		load(s, c, e, func(ctx context.Context) (*remote.Profile, error) {
			return c.Profile(ctx, name)
		})
	}
	return e.res
}

// Versions returns the versions that the plugin name can install, which search
// narrows for this call only; an empty search is the plugin's own list. A change of
// the plugin's search, and of the installed SDKs, marks it out of date.
func (s *Service) Versions(name, search string) Resource[[]remote.AvailableVersion] {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.versions.at(versionsKey{name: name, search: search})
	if c := s.client; c != nil {
		load(s, c, e, func(ctx context.Context) ([]remote.AvailableVersion, error) {
			return c.Versions(ctx, name, search)
		})
	}
	return e.res
}

// Aliases returns the aliases of the plugin name.
func (s *Service) Aliases(name string) Resource[[]remote.VersionAlias] {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.aliases.at(name)
	if c := s.client; c != nil {
		load(s, c, e, func(ctx context.Context) ([]remote.VersionAlias, error) {
			return c.Aliases(ctx, name)
		})
	}
	return e.res
}

// ForgetProbe drops the probe of the last link, so that the next read of Probe connects
// again. The dialog that asks the user to trust a computer calls it when it opens: the
// probe does not use the token of the link, so the link can be tried as often as the user
// likes, and a computer that did not answer a moment ago, because the network was not up
// yet or the user had not turned it on, may answer now, and may show another certificate.
// A probe that is still running is not waited for: its answer goes to an entry that nothing
// reads any more.
func (s *Service) ForgetProbe() {
	s.mu.Lock()
	s.probe, s.probeKey = nil, ""
	s.mu.Unlock()
}

// Info returns the metadata of the installed plugin name.
func (s *Service) Info(name string) Resource[remote.PluginInfo] {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.infos.at(name)
	if c := s.client; c != nil {
		load(s, c, e, func(ctx context.Context) (remote.PluginInfo, error) {
			return c.SdkInfo(ctx, name)
		})
	}
	return e.res
}

// Probe returns the certificate of the computer that link names, for the user to
// compare with the one the computer shows, before Pair. It needs no pairing: it
// reaches the address, and trusts nothing. The first read starts the probe; the
// reads of the same link share it, and a read of another link starts another. A
// link without an address is a failed resource. A probe that failed is not tried
// again by reading it: ForgetProbe makes the next read connect anew.
func (s *Service) Probe(link pairing.Link) Resource[Candidate] {
	s.mu.Lock()
	defer s.mu.Unlock()
	if link.Base == nil {
		var failed Resource[Candidate]
		failed.Fail(errors.New("the link names no computer"))
		return failed
	}
	key := link.Base.String() + " " + link.Token
	if s.probe == nil || s.probeKey != key {
		s.probe, s.probeKey = &entry[Candidate]{}, key
	}
	load(s, nil, s.probe, func(ctx context.Context) (Candidate, error) {
		found, err := remote.Probe(ctx, link.Base)
		if err != nil {
			return Candidate{}, err
		}
		return Candidate{Link: link, Fingerprint: found.Fingerprint, Certificate: found.Certificate, SystemTrusted: found.SystemTrusted}, nil
	})
	return s.probe.res
}

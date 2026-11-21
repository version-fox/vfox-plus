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

import "github.com/version-fox/vfox-plus/mygo/phone/remote"

// onEvent applies an event of the computer's stream to the resources it names. The
// events carry no data: the resources are marked out of date, and each is loaded
// again when it is read next. client is the connection of the stream; the events of
// a connection that was replaced are ignored.
//
//	task         the tasks, every task and every log
//	inventory    the inventory, the doctor, the versions, the profiles, the aliases
//	             and the plugins' information (installed SDKs change them all)
//	vfox         the inventory and the doctor
//	web          the web state
//	hello        everything (the stream has just opened, after a break or a pause),
//	             and the phone is paired
func (s *Service) onEvent(client *remote.Client, kind string) {
	s.mu.Lock()
	if s.client != client {
		s.mu.Unlock()
		return
	}
	switch kind {
	case remote.EventHello:
		s.markAllLocked()
		s.state, s.cause = Paired, nil
	case remote.EventTask:
		s.markTasksLocked(nil)
	case remote.EventInventory:
		s.inventory.dirty = true
		s.doctors.mark(nil)
		s.versions.mark(nil)
		s.profiles.mark(nil)
		s.aliases.mark(nil)
		s.infos.mark(nil)
	case remote.EventVfox:
		s.inventory.dirty = true
		s.doctors.mark(nil)
	case remote.EventWeb:
		s.web.dirty = true
	}
	s.mu.Unlock()
	s.notify()
}

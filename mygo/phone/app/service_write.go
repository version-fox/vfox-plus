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

// The calls that change the computer. Each waits for the network, so the window
// runs it with Do and never calls it on the main thread. When the call succeeds,
// the resources that it can change are marked out of date, and the next read of
// each loads it again; the call's own error is returned, and a failed call changes
// no resource. A call that fails because the computer does not accept the phone
// ends the connection (see observeLocked), and the record stays saved.
//
// What each call marks:
//
//	StartTask, AddPlugin      the list of tasks (the new task is in it)
//	CancelTask, RetryTask     the list, and that task and its log
//	ClearFinished             the list, and every task and log
//	SetSearch                 the plugin's versions (every search) and profile
//	SetVariable               the plugin's profile
//	PutConfig, DeleteConfig   the inventory (its configuration)
//	PutSettings               the settings are set to the answer, and the inventory
//	                          and the doctor are marked (vfox's path can change them)
//	PutWeb                    the web state is set to the answer
//	WebPairing, Quit          nothing: they change no resource the phone keeps
//
// The changes that the computer makes on its own, such as a task that finishes,
// come as events (see onEvent), not as the answer of a call.

import (
	"context"

	"github.com/version-fox/vfox-plus/mygo/phone/remote"
)

// StartTask queues a task on the computer and returns it.
func (s *Service) StartTask(ctx context.Context, request remote.TaskRequest) (remote.Task, error) {
	var task remote.Task
	err := s.call(func(c *remote.Client) (err error) {
		task, err = c.StartTask(ctx, request)
		return err
	})
	if err == nil {
		s.touch(func() { s.markTasksLocked(none) })
	}
	return task, err
}

// CancelTask cancels the task with id.
func (s *Service) CancelTask(ctx context.Context, id int) error {
	err := s.call(func(c *remote.Client) error {
		return c.CancelTask(ctx, id)
	})
	if err == nil {
		s.touch(func() { s.markTasksLocked(only(id)) })
	}
	return err
}

// RetryTask runs the task with id again.
func (s *Service) RetryTask(ctx context.Context, id int) error {
	err := s.call(func(c *remote.Client) error {
		return c.RetryTask(ctx, id)
	})
	if err == nil {
		s.touch(func() { s.markTasksLocked(only(id)) })
	}
	return err
}

// ClearFinished forgets the finished tasks, so that every task is marked.
func (s *Service) ClearFinished(ctx context.Context) error {
	err := s.call(func(c *remote.Client) error {
		return c.ClearFinished(ctx)
	})
	if err == nil {
		s.touch(func() { s.markTasksLocked(nil) })
	}
	return err
}

// AddPlugin adds a plugin from a source, and returns the task that adds it. Its
// alias is optional.
func (s *Service) AddPlugin(ctx context.Context, source, alias string) (remote.Task, error) {
	var task remote.Task
	err := s.call(func(c *remote.Client) (err error) {
		task, err = c.AddPlugin(ctx, source, alias)
		return err
	})
	if err == nil {
		s.touch(func() { s.markTasksLocked(none) })
	}
	return task, err
}

// SetSearch chooses what the plugin name lists. An empty value is the plugin's
// default.
func (s *Service) SetSearch(ctx context.Context, name, value string) error {
	err := s.call(func(c *remote.Client) error {
		return c.SetSearch(ctx, name, value)
	})
	if err == nil {
		s.touch(func() {
			s.versions.mark(func(key versionsKey) bool { return key.name == name })
			s.profiles.mark(func(key string) bool { return key == name })
		})
	}
	return err
}

// SetVariable sets the environment variable key of the plugin name. An empty value
// takes it away.
func (s *Service) SetVariable(ctx context.Context, name, key, value string) error {
	err := s.call(func(c *remote.Client) error {
		return c.SetVariable(ctx, name, key, value)
	})
	if err == nil {
		s.touch(func() {
			s.profiles.mark(func(k string) bool { return k == name })
		})
	}
	return err
}

// PutConfig sets a key of the vfox configuration.
func (s *Service) PutConfig(ctx context.Context, key, value string) error {
	err := s.call(func(c *remote.Client) error {
		return c.PutConfig(ctx, key, value)
	})
	if err == nil {
		s.touch(func() { s.inventory.dirty = true })
	}
	return err
}

// DeleteConfig resets a key of the vfox configuration to its default.
func (s *Service) DeleteConfig(ctx context.Context, key string) error {
	err := s.call(func(c *remote.Client) error {
		return c.DeleteConfig(ctx, key)
	})
	if err == nil {
		s.touch(func() { s.inventory.dirty = true })
	}
	return err
}

// PutSettings changes the settings that patch sets, and returns all of them.
func (s *Service) PutSettings(ctx context.Context, patch remote.SettingsPatch) (remote.AppSettings, error) {
	var settings remote.AppSettings
	err := s.call(func(c *remote.Client) (err error) {
		settings, err = c.PutSettings(ctx, patch)
		return err
	})
	if err == nil {
		s.touch(func() {
			set(s.settings, settings)
			s.inventory.dirty = true
			s.doctors.mark(nil)
		})
	}
	return settings, err
}

// PutWeb changes how other devices connect, and returns the new state.
func (s *Service) PutWeb(ctx context.Context, patch remote.WebPatch) (remote.WebState, error) {
	var state remote.WebState
	err := s.call(func(c *remote.Client) (err error) {
		state, err = c.PutWeb(ctx, patch)
		return err
	})
	if err == nil {
		s.touch(func() { set(s.web, state) })
	}
	return state, err
}

// WebPairing makes a pairing link and its QR code for another device. host names
// the address to use; empty lets the computer choose.
func (s *Service) WebPairing(ctx context.Context, host string) (remote.PairingCode, error) {
	var code remote.PairingCode
	err := s.call(func(c *remote.Client) (err error) {
		code, err = c.WebPairing(ctx, host)
		return err
	})
	return code, err
}

// Quit ends the desktop application. The stream of the computer then breaks, and
// the phone shows the computer as unreachable.
func (s *Service) Quit(ctx context.Context) error {
	return s.call(func(c *remote.Client) error {
		return c.Quit(ctx)
	})
}

// call runs fn with the connection of the phone, and then reads what the answer
// says about the connection. Without a paired computer it returns ErrUnpaired.
func (s *Service) call(fn func(c *remote.Client) error) error {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil {
		return remote.ErrUnpaired
	}
	err := fn(client)
	s.mu.Lock()
	s.observeLocked(client, err)
	s.mu.Unlock()
	s.notify()
	return err
}

// touch runs mark with s.mu held, and asks the window for a frame.
func (s *Service) touch(mark func()) {
	s.mu.Lock()
	mark()
	s.mu.Unlock()
	s.notify()
}

// markTasksLocked marks the list of tasks, and the tasks and logs whose ID match
// out of date. A nil match marks every one. The caller holds s.mu.
func (s *Service) markTasksLocked(match func(id int) bool) {
	s.tasks.dirty = true
	s.task.mark(match)
	s.logs.mark(match)
}

// none matches no task: the list alone is marked.
func none(int) bool { return false }

// only returns the match of the task with id.
func only(id int) func(int) bool {
	return func(key int) bool { return key == id }
}

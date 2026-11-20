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

import (
	"context"
	"errors"

	"github.com/version-fox/vfox-plus/mygo/phone/remote"
	"github.com/version-fox/vfox-plus/mygo/phone/store"
)

// Pair pairs the phone with the computer of candidate. The one-time link is
// exchanged for a session, which the computer accepts from then on, and the
// computer is saved with the fingerprint that the user confirmed, in place of the
// one saved before. The pairing is not made when the certificate of the computer is
// not the candidate's: the connection is refused, and nothing is saved.
func (s *Service) Pair(ctx context.Context, candidate Candidate) error {
	if candidate.Link.Base == nil || candidate.Fingerprint == "" {
		return errors.New("the candidate names no computer to pair with")
	}
	anonymous, err := remote.New(candidate.Link.Base, candidate.Fingerprint, "")
	if err != nil {
		return err
	}
	session, err := anonymous.Pair(ctx, candidate.Link.Token)
	if err != nil {
		return err
	}
	client, err := remote.New(candidate.Link.Base, candidate.Fingerprint, session)
	if err != nil {
		return err
	}
	record := store.Record{
		Name:        candidate.Link.Base.Hostname(),
		Base:        candidate.Link.Base.String(),
		Fingerprint: candidate.Fingerprint,
	}
	if err := s.servers.Save(record); err != nil {
		return err
	}
	if err := s.sessions.Save(session); err != nil {
		_ = s.servers.Forget()
		return err
	}
	s.mu.Lock()
	s.stopStreamLocked()
	s.server, s.client = record, client
	s.state, s.cause = Paired, nil
	s.resetCachesLocked()
	s.kickLocked()
	s.mu.Unlock()
	s.notify()
	return nil
}

// Disconnect forgets the computer and its session, and the phone shows the pairing
// page until it pairs again. The computer is not asked to end the session: the API
// has no such call.
func (s *Service) Disconnect() error {
	s.mu.Lock()
	s.stopStreamLocked()
	s.client, s.server = nil, store.Record{}
	s.state, s.cause = Unpaired, nil
	s.resetCachesLocked()
	s.kickLocked()
	s.mu.Unlock()
	s.notify()
	return errors.Join(s.sessions.Forget(), s.servers.Forget())
}

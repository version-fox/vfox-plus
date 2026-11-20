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
	"net/url"
	"sync"
	"time"

	"github.com/version-fox/vfox-plus/mygo/phone/pairing"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
	"github.com/version-fox/vfox-plus/mygo/phone/store"
)

// Service is the connection of the phone to its computer, and the data that the
// window shows from it.
//
// Threads. Every method may be called from any goroutine. The window calls the
// read methods on the main thread, every frame: they never wait for the network.
// A read that finds a resource out of date starts its load on a goroutine of its
// own, and the window is asked for a frame (SetNotify) when the load ends; the
// next frame reads the new value. The calls that change the computer (the methods
// that return an error) wait for the network, so they run on a goroutine: Do does
// that, and reports their errors to the window as Notices.
//
// Freshness. A resource is loaded when it is first read, and again when it is read
// after it was marked out of date: by an event of the computer, by a change that
// the phone made, or by Refresh. A failed load is not repeated by reads, which
// keeps a frame from making requests; Refresh, or an event, makes the next read
// load it again. The resource keeps its last value while it loads again.
//
// Lifecycle. Run keeps the connection while the app lives. Suspend and Resume
// follow the app going to the background and coming back: the event stream and
// the calls in flight end at Suspend, and Resume connects again and marks every
// resource out of date.
type Service struct {
	servers    store.ServerStore
	sessions   store.SessionStore
	retryDelay time.Duration

	// kick wakes the loop of Run, which looks again at the state of the connection.
	kick chan struct{}

	// mu guards every field below. No network call is made with mu held, and the
	// window is never called with it held (see notify).
	mu sync.Mutex
	// onChange asks the window for a frame; see SetNotify.
	onChange   func()
	state      Session
	cause      error
	server     store.Record
	client     *remote.Client
	foreground bool
	// reqCtx is the context of the calls and the loads. It ends when the app goes
	// to the background, which cancels them; reqCancel ends it.
	reqCtx    context.Context
	reqCancel context.CancelFunc
	// stopStream ends the event stream of the loop of Run, and is nil when there
	// is none.
	stopStream context.CancelFunc
	notices    []Notice

	settings  *entry[remote.AppSettings]
	web       *entry[remote.WebState]
	inventory *entry[remote.Inventory]
	tasks     *entry[[]remote.Task]
	probe     *entry[Candidate]
	probeKey  string
	task      cache[int, remote.Task]
	logs      cache[int, string]
	doctors   cache[doctorKey, remote.DoctorReport]
	profiles  cache[string, *remote.Profile]
	versions  cache[versionsKey, []remote.AvailableVersion]
	aliases   cache[string, []remote.VersionAlias]
	infos     cache[string, remote.PluginInfo]
}

// DefaultRetryDelay is the pause between two tries to reach the computer, after
// its event stream broke.
const DefaultRetryDelay = 3 * time.Second

// maxNotices bounds the notices that wait for the window; the oldest go first.
const maxNotices = 32

// Limits of the keyed caches: how many entries each keeps. A full log of a task is
// the largest answer of the API, so its cache is the smallest.
const (
	taskLimit     = 32
	logLimit      = 4
	doctorLimit   = 8
	profileLimit  = 32
	versionsLimit = 16
	aliasLimit    = 32
	infoLimit     = 32
)

// doctorKey and versionsKey are the keys of the caches that depend on more than
// one name.
type doctorKey struct{ sdk, version string }

type versionsKey struct{ name, search string }

// Options configures a Service.
type Options struct {
	// Servers keeps the saved computer.
	Servers store.ServerStore
	// Sessions keeps the session of the saved computer.
	Sessions store.SessionStore
	// RetryDelay is the pause between two tries to reach the computer; zero means
	// DefaultRetryDelay.
	RetryDelay time.Duration
}

// Status is the connection of the phone to its computer.
type Status struct {
	// Session is the state of the connection.
	Session Session
	// Server is the saved computer. It is the zero record when none is saved.
	Server store.Record
	// Cause is the error that the last try met, or nil. For a damaged saved
	// record it is the error of loading it.
	Cause error
}

// Candidate is a computer that the user is about to pair with: the link that
// names it, and the fingerprint of the certificate that it presents now. The user
// compares the fingerprint with the one that the computer shows, before Pair
// trusts it.
type Candidate struct {
	Link        pairing.Link
	Fingerprint string
	// SystemTrusted reports whether the system's roots verify the certificate for
	// the address. A desktop's own certificate is not, so the user's comparison of
	// the fingerprint is what decides.
	SystemTrusted bool
}

// Notice is the error of a call that Do ran in the background, for the window to
// show to the user.
type Notice struct {
	Kind Kind
	Err  error
}

// NewService returns the service of the saved computer. It reads the saved record
// and session, and decides the first state from them, without a network call: a
// saved computer with a session is Connecting, which Run checks; a saved computer
// without one needs pairing; nothing saved is Unpaired.
func NewService(opts Options) (*Service, error) {
	if opts.Servers == nil || opts.Sessions == nil {
		return nil, errors.New("the service needs a store of the computer and one of the session")
	}
	delay := opts.RetryDelay
	if delay <= 0 {
		delay = DefaultRetryDelay
	}
	s := &Service{
		servers:    opts.Servers,
		sessions:   opts.Sessions,
		retryDelay: delay,
		kick:       make(chan struct{}, 1),
		state:      Unpaired,
		foreground: true,
	}
	s.reqCtx, s.reqCancel = context.WithCancel(context.Background())
	s.resetCachesLocked()
	s.restore()
	return s, nil
}

// restore decides the first state from what is saved. It runs before the service
// is shared, so it does not take the lock.
func (s *Service) restore() {
	record, err := s.servers.Load()
	switch {
	case err == nil:
	case errors.Is(err, store.ErrNoRecord), errors.Is(err, store.ErrCorrupt):
		// A session without its computer is of no use: the next pairing saves a new
		// one. A damaged record is replaced by the next save, as an empty one is.
		_ = s.sessions.Forget()
		if errors.Is(err, store.ErrCorrupt) {
			s.cause = err
		}
		return
	default:
		s.cause = err
		return
	}
	s.server = record
	session, err := s.sessions.Load()
	if err != nil {
		// The computer is saved, so the pairing page can offer it, but no session is.
		s.state = NeedsPairing
		if !errors.Is(err, store.ErrNoSession) {
			s.cause = err
		}
		return
	}
	base, err := url.Parse(record.Base)
	var client *remote.Client
	if err == nil {
		client, err = remote.New(base, record.Fingerprint, session)
	}
	if err != nil {
		s.state, s.cause = NeedsPairing, err
		return
	}
	s.client, s.state = client, Connecting
}

// SetNotify sets the function that asks the window for a frame after the state
// changed. It is called from the goroutines that make the change, never with the
// lock of the service held, so it must be safe to call from any goroutine: the
// window's Invalidate is. A nil function sets none.
func (s *Service) SetNotify(notify func()) {
	s.mu.Lock()
	s.onChange = notify
	s.mu.Unlock()
}

// notify asks the window for a frame. The caller does not hold s.mu.
func (s *Service) notify() {
	s.mu.Lock()
	notify := s.onChange
	s.mu.Unlock()
	if notify != nil {
		notify()
	}
}

// Status returns the connection of the phone. The window reads it each frame.
func (s *Service) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Status{Session: s.state, Server: s.server, Cause: s.cause}
}

// Refresh marks every resource out of date: each is loaded again the next time it
// is read. The window uses it for its refresh buttons.
func (s *Service) Refresh() {
	s.mu.Lock()
	s.markAllLocked()
	s.mu.Unlock()
	s.notify()
}

// Do runs fn on a goroutine, with the request context of the service, and reports
// its error to the window as a Notice. It is for the calls that change the
// computer, which wait for the network. The error of a call that the app's
// suspension cancelled is not reported.
func (s *Service) Do(fn func(ctx context.Context) error) {
	s.mu.Lock()
	ctx := s.reqCtx
	s.mu.Unlock()
	go func() {
		err := fn(ctx)
		kind := KindOf(err)
		if err == nil || kind == KindCanceled {
			return
		}
		s.mu.Lock()
		s.notices = append(s.notices, Notice{Kind: kind, Err: err})
		if len(s.notices) > maxNotices {
			s.notices = s.notices[len(s.notices)-maxNotices:]
		}
		s.mu.Unlock()
		s.notify()
	}()
}

// TakeNotices returns the notices that Do made, oldest first, and forgets them.
// The window takes them on the main thread, once a frame.
func (s *Service) TakeNotices() []Notice {
	s.mu.Lock()
	defer s.mu.Unlock()
	notices := s.notices
	s.notices = nil
	return notices
}

// Run keeps the connection to the computer while ctx lasts. It checks a saved
// session once, follows the events of the computer, and when the stream breaks,
// tries again after the retry delay. It waits while the app is in the background
// or the phone has no computer, and it returns when ctx ends, which also ends the
// calls in flight. Run is called once, on a goroutine of its own.
func (s *Service) Run(ctx context.Context) {
	defer s.stop()
	for ctx.Err() == nil {
		client, stream, cancel := s.beginStream(ctx)
		if client == nil {
			s.wait(ctx, 0)
			continue
		}
		err := s.follow(client, stream)
		// The stream is judged before it is cancelled here: a stream that ended on its
		// own must not look as if it was stopped on purpose.
		retry := s.lost(client, err, stream)
		cancel()
		s.endStream()
		if retry {
			s.wait(ctx, s.retryDelay)
		}
	}
}

// beginStream returns the client to follow with the context of its stream, and the
// function that ends the stream. The stream ends with ctx, and when a suspension or
// a new pairing stops it (stopStreamLocked). It returns a nil client when there is
// nothing to follow: no computer, or the app is in the background.
func (s *Service) beginStream(ctx context.Context) (*remote.Client, context.Context, context.CancelFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client == nil || !s.foreground {
		return nil, nil, nil
	}
	stream, cancel := context.WithCancel(ctx)
	s.stopStream = cancel
	return s.client, stream, cancel
}

// endStream forgets the stream that ended, which the loop of Run started.
func (s *Service) endStream() {
	s.mu.Lock()
	s.stopStream = nil
	s.mu.Unlock()
}

// follow checks a saved computer that has not been checked yet, and then follows
// its events until the stream ends. It returns the error that ended the stream,
// nil when the computer closed it.
func (s *Service) follow(client *remote.Client, stream context.Context) error {
	s.mu.Lock()
	checking := s.state == Connecting && s.client == client
	s.mu.Unlock()
	if checking {
		settings, err := client.Settings(stream)
		if err != nil {
			return err
		}
		s.mu.Lock()
		if s.client == client {
			set(s.settings, settings)
			s.state, s.cause = Paired, nil
		}
		s.mu.Unlock()
		s.notify()
	}
	return client.Events(stream, func(kind string) { s.onEvent(client, kind) })
}

// lost reports whether the loop tries again after the stream of client ended with
// err. A stream that was stopped on purpose, by a suspension or a new pairing, broke
// nothing. A computer that refuses the session or changed its certificate is not
// tried again on a timer: its state says what is wrong. Any other end makes the
// phone Unreachable.
func (s *Service) lost(client *remote.Client, err error, stream context.Context) bool {
	s.mu.Lock()
	retry := false
	if stream.Err() == nil && s.client == client {
		s.observeLocked(client, err)
		if s.client == client {
			s.state, s.cause = Unreachable, err
			retry = true
		}
	}
	s.mu.Unlock()
	s.notify()
	return retry
}

// wait returns when ctx ends, when the loop is kicked, or after delay, which is no
// wait at all when it is zero.
func (s *Service) wait(ctx context.Context, delay time.Duration) {
	var later <-chan time.Time
	if delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		later = timer.C
	}
	select {
	case <-ctx.Done():
	case <-s.kick:
	case <-later:
	}
}

// stop ends the calls and the stream: the app is leaving.
func (s *Service) stop() {
	s.mu.Lock()
	s.reqCancel()
	s.stopStreamLocked()
	s.mu.Unlock()
}

// Suspend is called when the app goes to the background. The event stream ends and
// the calls in flight are cancelled; the loads that they were are read again after
// Resume, when the resources are read next.
func (s *Service) Suspend() {
	s.mu.Lock()
	s.foreground = false
	s.reqCancel()
	s.stopStreamLocked()
	s.kickLocked()
	s.mu.Unlock()
}

// Resume is called when the app comes back to the foreground. The connection is
// made again, and every resource is marked out of date, since events may have been
// missed while the app was away.
func (s *Service) Resume() {
	s.mu.Lock()
	if !s.foreground {
		s.foreground = true
		s.reqCtx, s.reqCancel = context.WithCancel(context.Background())
	}
	s.markAllLocked()
	s.kickLocked()
	s.mu.Unlock()
	s.notify()
}

// observeLocked ends the connection when the answer of the computer says that it
// no longer accepts the phone, or that its certificate is not the paired one. The
// record stays saved. client is the connection that made the call, and an answer
// about a connection that was replaced is no news. The caller holds s.mu.
func (s *Service) observeLocked(client *remote.Client, err error) {
	if err == nil || client == nil || client != s.client {
		return
	}
	switch {
	case errors.Is(err, remote.ErrUnpaired):
		s.dropLocked(NeedsPairing, err)
	case errors.Is(err, remote.ErrCertificateChanged):
		s.dropLocked(CertificateChanged, err)
	}
}

// dropLocked forgets the client and ends its stream. The caller holds s.mu.
func (s *Service) dropLocked(state Session, cause error) {
	s.stopStreamLocked()
	s.client = nil
	s.state, s.cause = state, cause
	s.kickLocked()
}

// stopStreamLocked ends the event stream, if there is one. The caller holds s.mu.
func (s *Service) stopStreamLocked() {
	if s.stopStream != nil {
		s.stopStream()
		s.stopStream = nil
	}
}

// kickLocked wakes the loop of Run, without waiting. The caller holds s.mu.
func (s *Service) kickLocked() {
	select {
	case s.kick <- struct{}{}:
	default:
	}
}

// resetCachesLocked forgets every resource. The resources of one computer are not
// those of another. The caller holds s.mu, or the service is not shared yet.
func (s *Service) resetCachesLocked() {
	s.settings = &entry[remote.AppSettings]{}
	s.web = &entry[remote.WebState]{}
	s.inventory = &entry[remote.Inventory]{}
	s.tasks = &entry[[]remote.Task]{}
	s.task = newCache[int, remote.Task](taskLimit)
	s.logs = newCache[int, string](logLimit)
	s.doctors = newCache[doctorKey, remote.DoctorReport](doctorLimit)
	s.profiles = newCache[string, *remote.Profile](profileLimit)
	s.versions = newCache[versionsKey, []remote.AvailableVersion](versionsLimit)
	s.aliases = newCache[string, []remote.VersionAlias](aliasLimit)
	s.infos = newCache[string, remote.PluginInfo](infoLimit)
}

// markAllLocked marks every resource out of date. The caller holds s.mu.
func (s *Service) markAllLocked() {
	s.settings.dirty = true
	s.web.dirty = true
	s.inventory.dirty = true
	s.tasks.dirty = true
	s.task.mark(nil)
	s.logs.mark(nil)
	s.doctors.mark(nil)
	s.profiles.mark(nil)
	s.versions.mark(nil)
	s.aliases.mark(nil)
	s.infos.mark(nil)
}

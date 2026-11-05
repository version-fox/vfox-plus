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

package gatewaycheck

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

// A gateway is run as its users run it: as a container of the Docker image, with a volume for what it
// remembers, or as the program itself. Either way it listens on port 8080 of its own, takes no arguments,
// prints how to pair a browser with it, and stops when it is asked to, the way docker stop asks it.

// gateway is a gateway that runs.
type gateway interface {
	// address is where a browser on this computer reaches it, host and port.
	address() string
	// output is what it has printed so far.
	output() string
	// stop asks it to stop as docker stop does, waits, and says how it ended and how long that took.
	stop() (took time.Duration, err error)
}

// program starts gateways that remember in the same place, so that one that is started again is paired still.
type program interface {
	start() (gateway, error)
	// hosts are the hosts at which a gateway may reach the application on this computer, the likeliest first:
	// addresses are the ones that the application names.
	hosts(addresses []string) []string
	// cleanup removes what the program made: the container, the volume.
	cleanup()
}

// bannerLink and bannerFingerprint find in what the gateway prints the link that pairs a browser and the
// fingerprint of its certificate.
var (
	bannerLink        = regexp.MustCompile(`https://localhost:\d+(/auth/connect/\S+)`)
	bannerFingerprint = regexp.MustCompile(`(?m)^  ((?:[0-9A-F]{2}:){31}[0-9A-F]{2})\s*$`)
)

// parseBanner takes the pairing path and the fingerprint out of what the gateway printed.
func parseBanner(text string) (path, fingerprint string, err error) {
	link := bannerLink.FindStringSubmatch(text)
	if link == nil {
		return "", "", errors.New("no pairing link in what the gateway printed")
	}
	found := bannerFingerprint.FindStringSubmatch(text)
	if found == nil {
		return "", "", errors.New("no fingerprint in what the gateway printed")
	}
	return link[1], found[1], nil
}

// waitForBanner waits until the gateway has printed how to pair a browser with it.
func waitForBanner(g gateway, timeout time.Duration) (path, fingerprint string, err error) {
	deadline := time.Now().Add(timeout)
	for {
		if path, fingerprint, err := parseBanner(g.output()); err == nil {
			return path, fingerprint, nil
		}
		if time.Now().After(deadline) {
			return "", "", fmt.Errorf("the gateway printed no link to pair a browser with in %v; it printed:\n%s", timeout, strings.TrimSpace(g.output()))
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// ---- the Docker image

// image is a gateway that runs as a container of an image, with a volume of its own.
type image struct {
	name   string
	volume string
	made   bool
}

func newImage(name string) *image {
	return &image{name: name, volume: fmt.Sprintf("vfox-plus-gateway-check-%d", os.Getpid())}
}

// hostGateway is a name that a container gets for the computer it runs on (docker run --add-host).
const hostGateway = "host.docker.internal"

// slirpHost is how the container of a Docker without privileges reaches the computer: slirp4netns names the
// loopback interface of the computer so.
const slirpHost = "10.0.2.2"

// hosts are the addresses that the application names (the computer is reached through the network of the
// container as any computer on it), and the names that a container has for the computer it runs on.
func (*image) hosts(addresses []string) []string {
	return append(append([]string{}, addresses...), hostGateway, slirpHost)
}

// docker runs the Docker command and returns what it printed, and for the logs of a container what it
// printed on both of its outputs: a container keeps them apart.
func docker(ctx context.Context, args ...string) (stdout, stderr string, err error) {
	command := exec.CommandContext(ctx, "docker", args...) //nolint:gosec // The tool of the system, with arguments of this program.
	var out, errOut bytes.Buffer
	command.Stdout, command.Stderr = &out, &errOut
	if err := command.Run(); err != nil {
		return out.String(), errOut.String(), fmt.Errorf("docker %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(errOut.String()))
	}
	return out.String(), errOut.String(), nil
}

func (i *image) start() (gateway, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if !i.made {
		if _, _, err := docker(ctx, "volume", "create", i.volume); err != nil {
			return nil, err
		}
		i.made = true
	}
	id, _, err := docker(ctx, "run", "--detach", "--publish", "127.0.0.1::8080", "--add-host", hostGateway+":host-gateway",
		"--volume", i.volume+":/home/vfoxplus/.config/vfox-plus-gateway", i.name)
	if err != nil {
		return nil, err
	}
	c := &container{id: strings.TrimSpace(id)}
	published, _, err := docker(ctx, "port", c.id, "8080/tcp")
	if err != nil {
		c.remove()
		return nil, err
	}
	// "127.0.0.1:49153", and a line for each family of addresses that it is published on.
	line, _, _ := strings.Cut(strings.TrimSpace(published), "\n")
	_, port, err := net.SplitHostPort(strings.TrimSpace(line))
	if err != nil {
		c.remove()
		return nil, fmt.Errorf("the port of the container (%q): %w", published, err)
	}
	c.host = net.JoinHostPort("127.0.0.1", port)
	return c, nil
}

func (i *image) cleanup() {
	if i.made {
		_, _, _ = docker(context.Background(), "volume", "rm", "--force", i.volume)
	}
}

// container is a running container of the image.
type container struct {
	id   string
	host string
}

func (c *container) address() string { return c.host }

func (c *container) output() string {
	out, errOut, _ := docker(context.Background(), "logs", c.id)
	return out + errOut
}

// stop is docker stop, and the code that the container ended with is 0 when the gateway ended as it should.
func (c *container) stop() (time.Duration, error) {
	start := time.Now()
	if _, _, err := docker(context.Background(), "stop", "--time", "10", c.id); err != nil {
		return time.Since(start), err
	}
	took := time.Since(start)
	code, _, err := docker(context.Background(), "inspect", "--format", "{{.State.ExitCode}}", c.id)
	c.remove()
	if err != nil {
		return took, err
	}
	if strings.TrimSpace(code) != "0" {
		return took, fmt.Errorf("the container ended with the code %s, want 0", strings.TrimSpace(code))
	}
	return took, nil
}

func (c *container) remove() { _, _, _ = docker(context.Background(), "rm", "--force", c.id) }

// ---- the program

// binary is a gateway that runs as the program itself, with a home directory of its own.
type binary struct {
	path string
	home string
}

// hosts are the addresses that the application names: the gateway is on the computer that it is on.
func (*binary) hosts(addresses []string) []string { return addresses }

func (b *binary) start() (gateway, error) {
	// The gateway listens on port 8080 and cannot be told another; see that it is free, to say so if it is not.
	listener, err := net.Listen("tcp", ":8080") //nolint:gosec // To see that the port of the gateway is free.
	if err != nil {
		return nil, fmt.Errorf("the gateway listens on port 8080 and cannot be told another: %w", err)
	}
	_ = listener.Close()
	if err := os.MkdirAll(b.home, 0o700); err != nil {
		return nil, err
	}
	log, err := os.CreateTemp(b.home, "gateway-*.log")
	if err != nil {
		return nil, err
	}
	command := exec.Command(b.path) //nolint:gosec // The program that this tool was given to check.
	command.Env = append(os.Environ(), "HOME="+b.home, "XDG_CONFIG_HOME="+filepath.Join(b.home, ".config"))
	command.Stdout, command.Stderr = log, log
	if err := command.Start(); err != nil {
		_ = log.Close()
		return nil, err
	}
	p := &process{command: command, log: log.Name(), ended: make(chan error, 1)}
	go func() {
		p.ended <- command.Wait()
		_ = log.Close()
	}()
	return p, nil
}

func (*binary) cleanup() {}

// process is a running gateway.
type process struct {
	command *exec.Cmd
	log     string
	ended   chan error
}

func (*process) address() string { return "127.0.0.1:8080" }

func (p *process) output() string {
	content, _ := os.ReadFile(p.log) //nolint:gosec // The log of the program that was started.
	return string(content)
}

func (p *process) stop() (time.Duration, error) {
	start := time.Now()
	_ = p.command.Process.Signal(syscall.SIGTERM)
	select {
	case err := <-p.ended:
		return time.Since(start), err
	case <-time.After(10 * time.Second):
		_ = p.command.Process.Kill()
		ended := <-p.ended
		return time.Since(start), fmt.Errorf("the gateway did not stop when it was asked to; it was killed: %w", ended)
	}
}

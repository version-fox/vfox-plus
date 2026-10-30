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

// Package sandbox is the host side of the Docker sandbox, replacing
// mygo/sandbox/sandbox.sh and shared/sandbox/sandbox-lib.sh.
//
// It builds, tests and packages an implementation in throw-away Docker
// containers, so that nothing on the host changes. The containers see the
// repository read-only and work on a copy of it; results are written to
// dist/sandbox/<implementation>/<command>/.
package sandbox

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const baseImageName = "vfox-plus-sandbox-base"

// cleanBaseImage is the image the installation test uses: a clean Ubuntu
// container that downloads the declared dependencies.
const cleanBaseImage = "ubuntu:26.04"

const usage = `Usage: sandbox [--impl IMPL] [--repo DIR] <command>

  check        gofmt, go vet, golangci-lint and the unit tests with the
               coverage gate
  e2e          the end-to-end scenarios, against the fixtures and against
               the real vfox
  package      release build with a smoke test, .deb and .tar.gz checked by
               lintian plus unsigned .msix/.msixbundle, then an installation
               into a clean Ubuntu container
  measure      the memory use on the Home page and with the versions of
               Node.js listed
  screenshots  screenshots of every page, against the fixtures
  all          check, e2e and package
  shell        an interactive shell in the sandbox
  image        builds the sandbox images
  clean        deletes the sandbox image, the cache volumes and the results

  Environment variables for slow connections to the default servers:
  SANDBOX_UBUNTU_MIRROR    Ubuntu archive mirror, for example
                           http://mirrors.aliyun.com/ubuntu/
  SANDBOX_GOPROXY          Go module proxy, for example
                           https://goproxy.cn,direct
  SANDBOX_NPM_REGISTRY     npm registry that the package of the icons comes
                           from, for example https://registry.npmmirror.com
`

// implConfig holds what sandbox-lib.sh expects the sourcing script to set:
// the docker build arguments of the image, the cache volumes and the docker
// run options of the containers of every task.
type implConfig struct {
	// imageArgs renders IMAGE_ARGS from the implementation's versions.env.
	imageArgs func(versions map[string]string) []string
	// cacheVolumes holds CACHE_VOLUMES, NAME:PATH pairs of the volumes that
	// keep downloaded dependencies between runs.
	cacheVolumes []string
	// runOptions holds RUN_OPTIONS, the docker run options of the containers
	// of every task and of the installation test.
	runOptions []string
}

// configs maps the directory name of an implementation to its sandbox setup.
// Only mygo exists yet; other implementations add their entry here, like
// they would add a sandbox.sh sourcing sandbox-lib.sh.
var configs = map[string]implConfig{
	"mygo": {
		imageArgs: func(versions map[string]string) []string {
			return []string{
				"--build-arg", "GO_VERSION=" + versions["GO_VERSION"],
				"--build-arg", "GO_SHA256=" + versions["GO_SHA256"],
				"--build-arg", "GOLANGCI_LINT_VERSION=" + versions["GOLANGCI_LINT_VERSION"],
				"--build-arg", "GOLANGCI_LINT_SHA256=" + versions["GOLANGCI_LINT_SHA256"],
			}
		},
		cacheVolumes: []string{
			"vfox-plus-sandbox-go-modules:/home/tester/go/pkg/mod",
			"vfox-plus-sandbox-go-build:/home/tester/.cache/go-build",
			// The npm package that the icons are cut out of (`vfxtool prepare`).
			"vfox-plus-sandbox-icons:/home/tester/.cache/vfox-plus",
		},
		// Software rendering and the embedded test server need more
		// shared memory than Docker gives containers by default.
		runOptions: []string{"--init", "--shm-size=1g"},
	},
}

// session holds what the sourcing script sets before calling sandbox_main:
// the implementation, its directory, the repository, the versions and the
// image arguments, volumes and run options above.
type session struct {
	impl       string
	implDir    string
	repo       string
	sandboxDir string
	results    string
	imageName  string
	versions   map[string]string
	config     implConfig
	epoch      string
}

// Run runs a sandbox command. args are the command line without the program
// name: sandbox [--impl IMPL] [--repo DIR] <command>.
func Run(args []string) error {
	impl := "mygo"
	repo := ""
	rest := args
	for len(rest) > 1 && strings.HasPrefix(rest[0], "--") {
		switch rest[0] {
		case "--impl":
			impl = rest[1]
		case "--repo":
			repo = rest[1]
		default:
			usageError()
			return fmt.Errorf("unknown option: %s", rest[0])
		}
		rest = rest[2:]
	}
	if len(rest) == 0 {
		usageError()
		return fmt.Errorf("no command given")
	}
	command, extra := rest[0], rest[1:]

	s, err := newSession(impl, repo)
	if err != nil {
		return err
	}

	switch command {
	case "check":
		if err := s.ensureImage(); err != nil {
			return err
		}
		return s.check()
	case "e2e":
		if err := s.ensureImage(); err != nil {
			return err
		}
		return s.e2e()
	case "package":
		if err := s.ensureImage(); err != nil {
			return err
		}
		return s.package_()
	case "all":
		if err := s.ensureImage(); err != nil {
			return err
		}
		return s.all()
	case "measure", "screenshots":
		if err := s.ensureImage(); err != nil {
			return err
		}
		if err := s.runTask("bridge", "deps", nil); err != nil {
			return err
		}
		return s.runTask("none", command, nil)
	case "shell":
		if err := s.ensureImage(); err != nil {
			return err
		}
		if err := s.runTask("bridge", "deps", nil); err != nil {
			return err
		}
		return s.runTask("bridge", "shell", []string{"--interactive", "--tty"})
	case "image":
		return s.ensureImage()
	case "image-name":
		image, err := s.implImage()
		if err != nil {
			return err
		}
		fmt.Println(image)
		return nil
	case "clean":
		return s.clean()
	case "-h", "--help", "help":
		fmt.Print(usage)
		return nil
	default:
		_ = extra
		usageError()
		return fmt.Errorf("unknown command: %s", command)
	}
}

func usageError() {
	fmt.Fprint(os.Stderr, usage)
}

// newSession resolves the implementation directory, the repository and the
// versions, like the top of sandbox.sh and sandbox-lib.sh do.
func newSession(impl, repo string) (*session, error) {
	config, ok := configs[impl]
	if !ok {
		return nil, fmt.Errorf("unknown implementation: %s", impl)
	}
	if repo == "" {
		repo = detectRepo()
	}
	abs, err := filepath.Abs(repo)
	if err != nil {
		return nil, err
	}
	repo = abs
	implDir := filepath.Join(repo, impl)
	versions, err := readVersions(filepath.Join(implDir, "versions.env"))
	if err != nil {
		return nil, err
	}
	for _, key := range []string{"VFOX_VERSION", "VFOX_SHA256"} {
		if versions[key] == "" {
			return nil, fmt.Errorf("%s is not set in %s/versions.env", key, impl)
		}
	}
	return &session{
		impl:       impl,
		implDir:    implDir,
		repo:       repo,
		sandboxDir: filepath.Join(implDir, "sandbox"),
		results:    filepath.Join(repo, "dist", "sandbox", impl),
		imageName:  "vfox-plus-sandbox-" + impl,
		versions:   versions,
		config:     config,
		epoch:      sourceEpoch(repo),
	}, nil
}

// detectRepo finds the repository root: the top level of the checkout, or
// the working directory when git cannot tell.
func detectRepo() string {
	if out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output(); err == nil {
		if dir := strings.TrimSpace(string(out)); dir != "" {
			return dir
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	return "."
}

// readVersions parses a versions.env file of KEY=VALUE lines.
func readVersions(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	versions := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		versions[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return versions, nil
}

// sourceEpoch returns SOURCE_DATE_EPOCH, defaulting to the last commit time
// of the repository and then to the current time, like sandbox-lib.sh.
func sourceEpoch(repo string) string {
	if epoch := strings.TrimSpace(os.Getenv("SOURCE_DATE_EPOCH")); epoch != "" {
		return epoch
	}
	if out, err := exec.Command("git", "-C", repo, "log", "-1", "--format=%ct").Output(); err == nil {
		if epoch := strings.TrimSpace(string(out)); epoch != "" {
			return epoch
		}
	}
	return strconv.FormatInt(time.Now().Unix(), 10)
}

// ShortHash hashes data with SHA-256 and returns the first 12 hex digits,
// like `sha256sum | cut -c1-12` in content_tag.
func ShortHash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])[:12]
}

// ContentTag hashes the concatenation of files, like content_tag in
// sandbox-lib.sh: an image is rebuilt exactly when its inputs change.
func ContentTag(files []string) (string, error) {
	h := sha256.New()
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", err
		}
		h.Write(data)
	}
	return hex.EncodeToString(h.Sum(nil))[:12], nil
}

// BaseImageName tags the shared base image by its Dockerfile, the vfxtool
// sources the tasks run through, and the vfox of the end-to-end tests.
func BaseImageName(vfoxVersion, tag string) string {
	return fmt.Sprintf("%s:%s-vfox-%s", baseImageName, tag, vfoxVersion)
}

// ImplImageName tags the image of an implementation by its Dockerfile, its
// pinned versions and the image it builds on. dockerfile and versionsEnv
// are the contents of those files; parent is the parent image reference,
// followed by a newline exactly like `echo "$parent" | cat ... -` in the
// script.
func ImplImageName(imageName, parent string, dockerfile, versionsEnv []byte) string {
	h := sha256.New()
	h.Write(dockerfile)
	h.Write(versionsEnv)
	h.Write([]byte(parent + "\n"))
	return imageName + ":" + hex.EncodeToString(h.Sum(nil))[:12]
}

// toolingFiles lists the sources the sandbox images embed the behavior of:
// the tooling module definition and every Go source under it, sorted. Every
// container runs tasks through `go run` of this module, so an image is rebuilt
// exactly when one of these changes.
func toolingFiles(repo string) ([]string, error) {
	var files []string
	root := filepath.Join(repo, "shared", "tooling")
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		return nil, err
	}
	files = append(files, filepath.Join(root, "go.mod"))
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(path, ".go") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

// baseImage returns the tag of the shared base image.
func (s *session) baseImage() (string, error) {
	files, err := toolingFiles(s.repo)
	if err != nil {
		return "", err
	}
	files = append([]string{filepath.Join(s.repo, "shared", "sandbox", "base.Dockerfile")}, files...)
	tag, err := ContentTag(files)
	if err != nil {
		return "", err
	}
	return BaseImageName(s.versions["VFOX_VERSION"], tag), nil
}

// parentImage returns the image the implementation's image is built on: the
// image of the last of PARENT_IMPLS, or the base image. Only mygo exists,
// which has no parents.
func (s *session) parentImage() (string, error) {
	return s.baseImage()
}

// implImage returns the tag of the implementation's image.
func (s *session) implImage() (string, error) {
	parent, err := s.parentImage()
	if err != nil {
		return "", err
	}
	dockerfile, err := os.ReadFile(filepath.Join(s.sandboxDir, "Dockerfile"))
	if err != nil {
		return "", err
	}
	versionsEnv, err := os.ReadFile(filepath.Join(s.implDir, "versions.env"))
	if err != nil {
		return "", err
	}
	return ImplImageName(s.imageName, parent, dockerfile, versionsEnv), nil
}

// mirrorArgs returns the Ubuntu mirror build argument, if one is set.
func mirrorArgs() []string {
	if mirror := os.Getenv("SANDBOX_UBUNTU_MIRROR"); mirror != "" {
		return []string{"--build-arg=UBUNTU_MIRROR=" + mirror}
	}
	return nil
}

// ensureImage builds the base image and the implementation's image unless
// they already exist.
func (s *session) ensureImage() error {
	image, err := s.baseImage()
	if err != nil {
		return err
	}
	if !imageExists(image) {
		args := []string{
			"build", "--tag", image,
			"--file", filepath.Join(s.repo, "shared", "sandbox", "base.Dockerfile"),
			"--build-arg", "VFOX_VERSION=" + s.versions["VFOX_VERSION"],
			"--build-arg", "VFOX_SHA256=" + s.versions["VFOX_SHA256"],
		}
		args = append(args, mirrorArgs()...)
		args = append(args, filepath.Join(s.repo, "shared"))
		if err := docker(args...); err != nil {
			return err
		}
	}
	image, err = s.implImage()
	if err != nil {
		return err
	}
	if imageExists(image) {
		return nil
	}
	parent, err := s.parentImage()
	if err != nil {
		return err
	}
	args := []string{"build", "--tag", image, "--build-arg", "BASE_IMAGE=" + parent}
	args = append(args, mirrorArgs()...)
	args = append(args, s.config.imageArgs(s.versions)...)
	args = append(args, s.sandboxDir)
	return docker(args...)
}

// ownerArgs tells a rootful Docker daemon whom the results belong to. Under
// rootless Docker, root in a container is the user on the host.
func ownerArgs() []string {
	out, err := dockerOutput("info", "--format", "{{.SecurityOptions}}")
	if err != nil || !strings.Contains(string(out), "rootless") {
		return []string{fmt.Sprintf("--env=OUT_OWNER=%d:%d", os.Getuid(), os.Getgid())}
	}
	return nil
}

// removeResults deletes a directory of results. Under rootless Docker the files
// that a task wrote belong to a subordinate user, whom the user of the host may not
// delete from (a directory that the task made has no write permission for others).
// Root in a container may, so when the host refuses, the directory is deleted by a
// throw-away container of the sandbox image.
func (s *session) removeResults(dir string) error {
	err := os.RemoveAll(dir)
	if err == nil || !errors.Is(err, fs.ErrPermission) {
		return err
	}
	image := s.imageToDeleteWith()
	if image == "" {
		return fmt.Errorf("%w\nThe files belong to a subordinate user of rootless Docker: delete the directory as root, or build the image of the sandbox, which can do it", err)
	}
	fmt.Fprintf(os.Stderr, "Deleting %s in a container: what a task wrote under rootless Docker is not the user's to delete.\n", dir)
	if err := docker("run", "--rm", "--network", "none",
		"--volume", filepath.Dir(dir)+":/parent", "--entrypoint", "rm",
		image, "-rf", "--", "/parent/"+filepath.Base(dir)); err != nil {
		return fmt.Errorf("delete %s in a container: %w", dir, err)
	}
	return os.RemoveAll(dir)
}

// imageToDeleteWith is an image that is there to run rm in: the image of the
// implementation, else the base image or the clean Ubuntu. It is "" if there is none.
func (s *session) imageToDeleteWith() string {
	var candidates []string
	if image, err := s.implImage(); err == nil {
		candidates = append(candidates, image)
	}
	if image, err := s.baseImage(); err == nil {
		candidates = append(candidates, image)
	}
	for _, image := range append(candidates, cleanBaseImage) {
		if imageExists(image) {
			return image
		}
	}
	return ""
}

// runTask runs a task of the implementation in a new container that sees the
// repository read-only. The results go to
// dist/sandbox/<implementation>/<task>, which is emptied first.
//
// The task runs through `go run` of the vfxtool in the mounted repository:
// the images bring Go, and the `deps` task downloads the modules of both
// the application and the tooling into the shared module cache, so the
// offline tasks need no network access.
func (s *session) runTask(network, task string, extra []string) error {
	opts := []string{
		"--rm", "--init",
		"--network", network,
		"--volume", s.repo + ":/src:ro",
		"--env", "SOURCE_DATE_EPOCH=" + s.epoch,
	}
	// The mirror settings (SANDBOX_NPM_REGISTRY, SANDBOX_GOPROXY, ...) reach the tasks.
	for _, env := range os.Environ() {
		if name, _, ok := strings.Cut(env, "="); ok && strings.HasPrefix(name, "SANDBOX_") {
			opts = append(opts, "--env", name)
		}
	}
	for _, volume := range s.config.cacheVolumes {
		opts = append(opts, "--volume", volume)
	}
	resultsTask := filepath.Join(s.results, task)
	if err := s.removeResults(resultsTask); err != nil {
		return err
	}
	if err := os.MkdirAll(resultsTask, 0o755); err != nil {
		return err
	}
	// The container runs as tester, whose uid maps to a subordinate uid on
	// the host under rootless Docker, so the host-owned directory would not
	// be writable to it. Open it up like the shared apt archives: the
	// results are disposable build outputs.
	if err := os.Chmod(resultsTask, 0o777); err != nil {
		return err
	}
	opts = append(opts, "--volume", resultsTask+":/out")
	// `go run` needs the module of the directory it runs in, and the caches
	// and the module cache belong to tester.
	opts = append(opts, "--workdir", "/src/shared/tooling")
	image, err := s.implImage()
	if err != nil {
		return err
	}
	args := append(opts, s.config.runOptions...)
	args = append(args, ownerArgs()...)
	args = append(args, extra...)
	args = append(args, image,
		"runuser", "-u", "tester", "--", "env", "HOME=/home/tester",
		"go", "run", "./cmd/vfxtool", "runtask", task)
	if err := docker(append([]string{"run"}, args...)...); err != nil {
		return err
	}
	// Keep only the result directories that received something.
	_ = os.Remove(resultsTask)
	return nil
}

// installTest installs the .deb of the package task into a clean Ubuntu
// container, which downloads the dependencies the package declares, and
// starts the application.
// installTest installs the .deb of the package task into a clean Ubuntu
// container, which downloads the dependencies the package declares, and
// starts the application. The clean image has no Go toolchain, so the test
// runs the static vfxtool the package task published next to the .deb.
func (s *session) installTest() error {
	mirror := os.Getenv("SANDBOX_UBUNTU_MIRROR")
	args := []string{
		"run", "--rm", "--init", "--network", "bridge",
		"--volume", filepath.Join(s.results, "package") + ":/pkg:ro",
		"--env", "UBUNTU_MIRROR=" + mirror,
	}
	args = append(args, s.config.runOptions...)
	args = append(args, cleanBaseImage, "/pkg/vfxtool", "runtask", "install-test")
	return docker(args...)
}

func (s *session) check() error {
	if err := s.runTask("bridge", "deps", nil); err != nil {
		return err
	}
	return s.runTask("none", "check", nil)
}

func (s *session) e2e() error {
	if err := s.runTask("bridge", "deps", nil); err != nil {
		return err
	}
	return s.runTask("none", "e2e", nil)
}

func (s *session) package_() error {
	if err := s.runTask("bridge", "deps", nil); err != nil {
		return err
	}
	if err := s.runTask("none", "package", nil); err != nil {
		return err
	}
	return s.installTest()
}

func (s *session) all() error {
	if err := s.runTask("bridge", "deps", nil); err != nil {
		return err
	}
	for _, task := range []string{"check", "e2e", "package"} {
		if err := s.runTask("none", task, nil); err != nil {
			return err
		}
	}
	return s.installTest()
}

func (s *session) clean() error {
	// The results go first: the image that can delete them under rootless Docker is
	// about to go.
	resultsErr := s.removeResults(s.results)
	if out, err := dockerOutput("image", "ls", "--format", "{{.Repository}}:{{.Tag}}", s.imageName); err == nil {
		var images []string
		for _, line := range strings.Split(string(out), "\n") {
			if line = strings.TrimSpace(line); line != "" {
				images = append(images, line)
			}
		}
		if len(images) > 0 {
			_ = docker(append([]string{"image", "rm"}, images...)...)
		}
	}
	for _, volume := range s.config.cacheVolumes {
		if name, _, ok := strings.Cut(volume, ":"); ok {
			_ = docker("volume", "rm", "--force", name)
		}
	}
	return resultsErr
}

// The commands of Docker that the sandbox runs. They are variables so that the
// tests need no Docker.
var (
	// imageExists reports whether a Docker image exists locally.
	imageExists = func(image string) bool {
		return exec.Command("docker", "image", "inspect", image).Run() == nil
	}
	// docker runs a docker command with the standard streams attached.
	docker = func(args ...string) error {
		cmd := exec.Command("docker", args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	// dockerOutput runs a docker command and returns what it printed.
	dockerOutput = func(args ...string) ([]byte, error) {
		return exec.Command("docker", args...).Output()
	}
)

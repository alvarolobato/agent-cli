package discovery

import (
	"io/fs"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestInstallDirScannerDetectsElasticAgentInstallAndConfigs(t *testing.T) {
	const root = "/opt/Elastic/Agent"

	fsys := newFakeInstallFS().
		addDir(root).
		addDir(filepath.Join(root, "inputs.d")).
		addDir(filepath.Join(root, "data")).
		addDir(filepath.Join(root, "data", "elastic-agent-9.2.4-abcd")).
		addFile(filepath.Join(root, ".flavor")).
		addFile(filepath.Join(root, ".installed")).
		addFile(filepath.Join(root, "elastic-agent")).
		addFile(filepath.Join(root, "elastic-agent.yml")).
		addFile(filepath.Join(root, "otel.yml")).
		addFile(filepath.Join(root, "inputs.d", "applications.yml")).
		addFile(filepath.Join(root, "inputs.d", "docker.yaml")).
		addFile(filepath.Join(root, "inputs.d", "disabled.yml_no")).
		addFile(filepath.Join(root, "data", "elastic-agent-9.2.4-abcd", "manifest.yaml"))

	scanner := NewInstallDirScannerWithFS([]string{root}, fsys)

	got, ok, err := scanner.scanInstallDir(root)
	if err != nil {
		t.Fatalf("scanInstallDir() error = %v", err)
	}
	if !ok {
		t.Fatalf("scanInstallDir() expected detected install dir")
	}
	if got.AgentType != agentTypeElastic {
		t.Fatalf("scanInstallDir() type = %q, want %q", got.AgentType, agentTypeElastic)
	}
	if got.InstallPath != root {
		t.Fatalf("scanInstallDir() install path = %q, want %q", got.InstallPath, root)
	}

	wantConfigs := []string{
		filepath.Join(root, "data", "elastic-agent-9.2.4-abcd", "manifest.yaml"),
		filepath.Join(root, "elastic-agent.yml"),
		filepath.Join(root, "inputs.d", "applications.yml"),
		filepath.Join(root, "inputs.d", "docker.yaml"),
		filepath.Join(root, "otel.yml"),
	}
	if !reflect.DeepEqual(got.ConfigPaths, wantConfigs) {
		t.Fatalf("scanInstallDir() config paths = %#v, want %#v", got.ConfigPaths, wantConfigs)
	}
}

func TestInstallDirScannerDiscoverAtPathFallsBackToConfigDirectory(t *testing.T) {
	const root = "/etc/otelcol"

	fsys := newFakeInstallFS().
		addDir(root).
		addFile(filepath.Join(root, "config.yaml"))

	scanner := NewInstallDirScannerWithFS([]string{root}, fsys)
	got, err := scanner.discoverAtPath(root)
	if err != nil {
		t.Fatalf("discoverAtPath() error = %v", err)
	}
	if got.AgentType != agentTypeOTel {
		t.Fatalf("discoverAtPath() type = %q, want %q", got.AgentType, agentTypeOTel)
	}
	if len(got.ConfigPaths) != 1 || got.ConfigPaths[0] != filepath.Join(root, "config.yaml") {
		t.Fatalf("discoverAtPath() config paths = %#v", got.ConfigPaths)
	}
}

func TestInstallDirScannerDiscoverAtPathInfersEDOTByDirectoryName(t *testing.T) {
	const root = "/etc/elastic-otel-collector"

	fsys := newFakeInstallFS().
		addDir(root).
		addFile(filepath.Join(root, "config.yml"))

	scanner := NewInstallDirScannerWithFS([]string{root}, fsys)
	got, err := scanner.discoverAtPath(root)
	if err != nil {
		t.Fatalf("discoverAtPath() error = %v", err)
	}
	if got.AgentType != agentTypeEDOT {
		t.Fatalf("discoverAtPath() type = %q, want %q", got.AgentType, agentTypeEDOT)
	}
}

func TestInstallDirScannerExtractsElasticMetadata(t *testing.T) {
	const root = "/opt/Elastic/Agent"
	fsys := newFakeInstallFS().
		addDir(root).
		addDir(filepath.Join(root, "data")).
		addDir(filepath.Join(root, "data", "elastic-agent-9.3.1-abcd")).
		addFile(filepath.Join(root, ".flavor")).
		addFileWithContents(filepath.Join(root, ".flavor"), []byte("servers\n")).
		addFileWithContents(filepath.Join(root, ".elastic-agent.active.commit"), []byte("abc123\n")).
		addFile(filepath.Join(root, ".installed")).
		addFile(filepath.Join(root, "elastic-agent")).
		addFile(filepath.Join(root, "elastic-agent.sock")).
		addFile(filepath.Join(root, "elastic-agent.yml")).
		addFileWithContents(filepath.Join(root, "data", "elastic-agent-9.3.1-abcd", "manifest.yaml"), []byte("version: 9.3.1\nbuild_hash: 1234abcd\n"))

	scanner := NewInstallDirScannerWithFS([]string{root}, fsys)
	got, ok, err := scanner.scanInstallDir(root)
	if err != nil {
		t.Fatalf("scanInstallDir() error = %v", err)
	}
	if !ok {
		t.Fatalf("scanInstallDir() expected detected install dir")
	}

	if got.Metadata["agent_flavor"] != "servers" {
		t.Fatalf("agent_flavor = %q, want servers", got.Metadata["agent_flavor"])
	}
	if got.Metadata["agent_version"] != "9.3.1" {
		t.Fatalf("agent_version = %q, want 9.3.1", got.Metadata["agent_version"])
	}
	if got.Metadata["build_hash"] != "1234abcd" {
		t.Fatalf("build_hash = %q, want 1234abcd", got.Metadata["build_hash"])
	}
	if got.Metadata["socket_path"] != filepath.Join(root, "elastic-agent.sock") {
		t.Fatalf("socket_path = %q", got.Metadata["socket_path"])
	}
}

type fakeInstallFS struct {
	dirs         map[string]map[string]bool
	files        map[string]struct{}
	fileContents map[string][]byte
}

func newFakeInstallFS() *fakeInstallFS {
	return &fakeInstallFS{
		dirs:         map[string]map[string]bool{},
		files:        map[string]struct{}{},
		fileContents: map[string][]byte{},
	}
}

func (f *fakeInstallFS) addDir(path string) *fakeInstallFS {
	path = filepath.Clean(path)
	if _, ok := f.dirs[path]; !ok {
		f.dirs[path] = map[string]bool{}
	}
	parent := filepath.Dir(path)
	if parent != path {
		f.addDir(parent)
		f.dirs[parent][filepath.Base(path)] = true
	}
	return f
}

func (f *fakeInstallFS) addFile(path string) *fakeInstallFS {
	return f.addFileWithContents(path, nil)
}

func (f *fakeInstallFS) addFileWithContents(path string, content []byte) *fakeInstallFS {
	path = filepath.Clean(path)
	f.files[path] = struct{}{}
	f.fileContents[path] = content
	parent := filepath.Dir(path)
	f.addDir(parent)
	f.dirs[parent][filepath.Base(path)] = false
	return f
}

func (f *fakeInstallFS) Stat(name string) (fs.FileInfo, error) {
	name = filepath.Clean(name)
	if _, ok := f.files[name]; ok {
		return fakeFileInfo{name: filepath.Base(name), dir: false}, nil
	}
	if _, ok := f.dirs[name]; ok {
		return fakeFileInfo{name: filepath.Base(name), dir: true}, nil
	}
	return nil, fs.ErrNotExist
}

func (f *fakeInstallFS) ReadDir(name string) ([]fs.DirEntry, error) {
	name = filepath.Clean(name)
	children, ok := f.dirs[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	out := make([]fs.DirEntry, 0, len(children))
	for childName, isDir := range children {
		out = append(out, fakeDirEntry{name: childName, dir: isDir})
	}
	return out, nil
}

func (f *fakeInstallFS) ReadFile(name string) ([]byte, error) {
	name = filepath.Clean(name)
	content, ok := f.fileContents[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return append([]byte(nil), content...), nil
}

type fakeFileInfo struct {
	name string
	dir  bool
}

func (f fakeFileInfo) Name() string       { return f.name }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() fs.FileMode  { return 0 }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return f.dir }
func (f fakeFileInfo) Sys() interface{}   { return nil }

type fakeDirEntry struct {
	name string
	dir  bool
}

func (f fakeDirEntry) Name() string               { return f.name }
func (f fakeDirEntry) IsDir() bool                { return f.dir }
func (f fakeDirEntry) Type() fs.FileMode          { return 0 }
func (f fakeDirEntry) Info() (fs.FileInfo, error) { return fakeFileInfo(f), nil }

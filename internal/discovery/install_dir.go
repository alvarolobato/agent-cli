package discovery

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/alvarolobato/agent-cli/internal/agent"
	"gopkg.in/yaml.v3"
)

const (
	agentTypeElastic = "elastic-agent"
	agentTypeEDOT    = "edot"
	agentTypeOTel    = "otel"
)

// DiscoveredAgent captures installation root and discovered configs.
type DiscoveredAgent struct {
	Type        string
	InstallPath string
	ConfigPaths []string
	Metadata    map[string]string
}

type installDirFS interface {
	Stat(name string) (fs.FileInfo, error)
	ReadDir(name string) ([]fs.DirEntry, error)
	ReadFile(name string) ([]byte, error)
}

type osInstallFS struct{}

func (osInstallFS) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(filepath.Clean(name))
}

func (osInstallFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(filepath.Clean(name))
}

func (osInstallFS) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(filepath.Clean(name))
}

type installDirScanner struct {
	fsys          installDirFS
	candidateDirs []string
}

// NewInstallDirScanner creates an installation-directory discovery strategy.
func NewInstallDirScanner() Strategy {
	return &installDirScanner{
		fsys:          osInstallFS{},
		candidateDirs: defaultInstallDirs(runtime.GOOS),
	}
}

// NewInstallDirScannerWithFS creates an install-dir scanner with test doubles.
func NewInstallDirScannerWithFS(candidateDirs []string, fsys installDirFS) *installDirScanner {
	if fsys == nil {
		fsys = osInstallFS{}
	}
	return &installDirScanner{
		fsys:          fsys,
		candidateDirs: candidateDirs,
	}
}

// DiscoverAgentAtPath scans a user-provided installation/config directory.
func DiscoverAgentAtPath(path string) (DiscoveredAgent, error) {
	scanner := NewInstallDirScannerWithFS([]string{path}, osInstallFS{})
	return scanner.discoverAtPath(path)
}

func (s *installDirScanner) Discover(ctx context.Context) ([]agent.Agent, error) {
	out := make([]agent.Agent, 0, len(s.candidateDirs))
	for _, root := range s.candidateDirs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		discovered, ok, err := s.scanInstallDir(root)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		out = append(out, stubAgent{
			id:   discovered.InstallPath,
			kind: discovered.Type,
		})
	}
	return out, nil
}

func (s *installDirScanner) discoverAtPath(path string) (DiscoveredAgent, error) {
	discovered, ok, err := s.scanInstallDir(path)
	if err != nil {
		return DiscoveredAgent{}, err
	}
	if ok {
		return discovered, nil
	}

	root := filepath.Clean(path)
	if !s.isDir(root) {
		return DiscoveredAgent{}, fmt.Errorf("path %q is not a readable directory", path)
	}

	configPaths, err := discoverManualConfigPaths(root, s.fsys)
	if err != nil {
		return DiscoveredAgent{}, err
	}
	if len(configPaths) == 0 {
		return DiscoveredAgent{}, fmt.Errorf("no supported config files found under %q", path)
	}

	agentType := inferAgentTypeFromPath(root, configPaths)
	if agentType == "" {
		return DiscoveredAgent{}, fmt.Errorf("unable to infer agent type from %q", path)
	}

	return DiscoveredAgent{
		Type:        agentType,
		InstallPath: root,
		ConfigPaths: configPaths,
		Metadata:    discoverInstallMetadata(root, configPaths, s.fsys),
	}, nil
}

func (s *installDirScanner) scanInstallDir(root string) (DiscoveredAgent, bool, error) {
	root = filepath.Clean(root)
	if !s.isDir(root) {
		return DiscoveredAgent{}, false, nil
	}

	agentType := detectAgentType(root, s.fsys)
	if agentType == "" {
		return DiscoveredAgent{}, false, nil
	}

	configPaths, err := discoverConfigPaths(root, s.fsys)
	if err != nil {
		return DiscoveredAgent{}, false, err
	}

	return DiscoveredAgent{
		Type:        agentType,
		InstallPath: root,
		ConfigPaths: configPaths,
		Metadata:    discoverInstallMetadata(root, configPaths, s.fsys),
	}, true, nil
}

func (s *installDirScanner) isDir(path string) bool {
	info, err := s.fsys.Stat(path)
	return err == nil && info.IsDir()
}

func defaultInstallDirs(goos string) []string {
	switch goos {
	case "darwin":
		return []string{
			"/Library/Elastic/Agent",
			"/etc/otelcol",
			"/etc/edot",
		}
	case "windows":
		return []string{
			`C:\Program Files\Elastic\Agent`,
			`C:\Program Files\otelcol`,
		}
	default:
		return []string{
			"/opt/Elastic/Agent",
			"/etc/otelcol",
			"/etc/otel",
			"/etc/edot",
			"/etc/elastic-otel-collector",
		}
	}
}

func detectAgentType(root string, fsys installDirFS) string {
	if fileExists(fsys, filepath.Join(root, ".flavor")) ||
		fileExists(fsys, filepath.Join(root, ".installed")) ||
		fileExists(fsys, filepath.Join(root, "elastic-agent")) {
		return agentTypeElastic
	}

	if fileExists(fsys, filepath.Join(root, "otelcol")) {
		lowerRoot := strings.ToLower(filepath.ToSlash(root))
		if strings.Contains(lowerRoot, "edot") || strings.Contains(lowerRoot, "elastic-otel-collector") {
			return agentTypeEDOT
		}
		return agentTypeOTel
	}

	return ""
}

func discoverConfigPaths(root string, fsys installDirFS) ([]string, error) {
	paths := make([]string, 0, 8)
	for _, name := range []string{"elastic-agent.yml", "otel.yml", "manifest.yaml"} {
		path := filepath.Join(root, name)
		if fileExists(fsys, path) {
			paths = append(paths, path)
		}
	}

	inputsDir := filepath.Join(root, "inputs.d")
	inputEntries, err := fsys.ReadDir(inputsDir)
	if err == nil {
		for _, entry := range inputEntries {
			if entry.IsDir() {
				continue
			}
			name := strings.ToLower(entry.Name())
			if !strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".yaml") {
				continue
			}
			paths = append(paths, filepath.Join(inputsDir, entry.Name()))
		}
	} else if !isNotExist(err) {
		return nil, err
	}

	dataDir := filepath.Join(root, "data")
	dataEntries, err := fsys.ReadDir(dataDir)
	if err == nil {
		for _, entry := range dataEntries {
			if !entry.IsDir() {
				continue
			}
			manifestPath := filepath.Join(dataDir, entry.Name(), "manifest.yaml")
			if fileExists(fsys, manifestPath) {
				paths = append(paths, manifestPath)
			}
		}
	} else if !isNotExist(err) {
		return nil, err
	}

	sort.Strings(paths)
	return paths, nil
}

func discoverManualConfigPaths(root string, fsys installDirFS) ([]string, error) {
	paths := make([]string, 0, 8)
	for _, name := range []string{"elastic-agent.yml", "elastic-agent.yaml", "otel.yml", "otel.yaml", "config.yml", "config.yaml"} {
		path := filepath.Join(root, name)
		if fileExists(fsys, path) {
			paths = append(paths, path)
		}
	}

	entries, err := fsys.ReadDir(root)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name())
		if strings.HasSuffix(name, ".yml_no") || strings.HasSuffix(name, ".yaml_no") {
			continue
		}
		if !strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".yaml") {
			continue
		}
		paths = append(paths, filepath.Join(root, entry.Name()))
	}

	sort.Strings(paths)
	return uniqueStrings(paths), nil
}

func inferAgentTypeFromPath(root string, configPaths []string) string {
	for _, configPath := range configPaths {
		name := strings.ToLower(filepath.Base(configPath))
		if name == "elastic-agent.yml" || name == "elastic-agent.yaml" {
			return agentTypeElastic
		}
	}

	lowerRoot := strings.ToLower(filepath.ToSlash(root))
	for _, configPath := range configPaths {
		name := strings.ToLower(filepath.Base(configPath))
		if name == "otel.yml" || name == "otel.yaml" || name == "config.yml" || name == "config.yaml" {
			if strings.Contains(lowerRoot, "edot") || strings.Contains(lowerRoot, "elastic-otel-collector") {
				return agentTypeEDOT
			}
			return agentTypeOTel
		}
	}

	return ""
}

func uniqueStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	last := ""
	for _, item := range items {
		if item == last {
			continue
		}
		out = append(out, item)
		last = item
	}
	return out
}

func fileExists(fsys installDirFS, path string) bool {
	info, err := fsys.Stat(path)
	return err == nil && !info.IsDir()
}

func isNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}

func discoverInstallMetadata(root string, configPaths []string, fsys installDirFS) map[string]string {
	metadata := map[string]string{}

	if flavorBytes, err := fsys.ReadFile(filepath.Join(root, ".flavor")); err == nil {
		if flavor := strings.TrimSpace(string(flavorBytes)); flavor != "" {
			metadata["agent_flavor"] = flavor
		}
	}

	if commitBytes, err := fsys.ReadFile(filepath.Join(root, ".elastic-agent.active.commit")); err == nil {
		if commit := strings.TrimSpace(string(commitBytes)); commit != "" {
			metadata["active_commit"] = commit
		}
	}

	socketPath := filepath.Join(root, "elastic-agent.sock")
	if fileExists(fsys, socketPath) {
		metadata["socket_path"] = socketPath
	}

	manifestPath := firstManifestPath(configPaths)
	if manifestPath != "" {
		if raw, err := fsys.ReadFile(manifestPath); err == nil {
			version, buildHash := parseManifestMetadata(raw)
			if version != "" {
				metadata["agent_version"] = version
			}
			if buildHash != "" {
				metadata["build_hash"] = buildHash
			}
		}
	}

	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

func firstManifestPath(paths []string) string {
	for _, path := range paths {
		if strings.EqualFold(filepath.Base(path), "manifest.yaml") {
			return path
		}
	}
	return ""
}

func parseManifestMetadata(raw []byte) (version string, buildHash string) {
	var payload map[string]interface{}
	if err := yaml.Unmarshal(raw, &payload); err != nil {
		return "", ""
	}
	version = firstManifestString(payload, "version", "build_version")
	buildHash = firstManifestString(payload, "commit", "build_hash")
	return version, buildHash
}

func firstManifestString(payload map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok {
			continue
		}
		asString, ok := value.(string)
		if !ok {
			continue
		}
		asString = strings.TrimSpace(asString)
		if asString != "" {
			return asString
		}
	}
	return ""
}

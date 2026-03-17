package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// InspectRunner executes elastic-agent inspect and returns stdout/stderr bytes.
type InspectRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

// InspectResult captures config parsing source information.
type InspectResult struct {
	Config *ElasticAgentConfig
	Source string
}

const (
	configSourceInspect = "inspect"
	configSourceFiles   = "files"
)

// ParseElasticAgentConfigWithInspect prefers elastic-agent inspect output and
// gracefully falls back to file-based parsing when inspect is unavailable.
func ParseElasticAgentConfigWithInspect(
	ctx context.Context,
	configPath string,
	runner InspectRunner,
) (*InspectResult, error) {
	if runner == nil {
		runner = defaultInspectRunner
	}

	if cfg, err := parseElasticConfigViaInspect(ctx, configPath, runner); err == nil && cfg != nil {
		return &InspectResult{
			Config: cfg,
			Source: configSourceInspect,
		}, nil
	}

	cfg, err := ParseElasticAgentConfig(configPath)
	if err != nil {
		return nil, err
	}
	return &InspectResult{
		Config: cfg,
		Source: configSourceFiles,
	}, nil
}

func parseElasticConfigViaInspect(ctx context.Context, configPath string, runner InspectRunner) (*ElasticAgentConfig, error) {
	configPath = strings.TrimSpace(configPath)
	if configPath == "" {
		return nil, errors.New("empty config path")
	}

	configDir := filepath.Dir(configPath)
	configFile := filepath.Base(configPath)
	bin := resolveElasticAgentBinaryPath(configDir)

	args := []string{
		"inspect",
		"--path.home", configDir,
		"--path.config", configDir,
		"-c", configFile,
	}
	out, err := runner(ctx, bin, args...)
	if err != nil {
		return nil, err
	}

	cfg, err := ParseElasticAgentConfigBytes(out)
	if err != nil {
		return nil, fmt.Errorf("parse inspect output: %w", err)
	}
	return cfg, nil
}

func resolveElasticAgentBinaryPath(configDir string) string {
	candidate := filepath.Join(configDir, "elastic-agent")
	info, err := os.Stat(candidate)
	if err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
		return candidate
	}
	if runtime.GOOS == "windows" {
		candidateExe := filepath.Join(configDir, "elastic-agent.exe")
		exeInfo, exeErr := os.Stat(candidateExe)
		if exeErr == nil && !exeInfo.IsDir() {
			return candidateExe
		}
		return "elastic-agent.exe"
	}
	return "elastic-agent"
}

func defaultInspectRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

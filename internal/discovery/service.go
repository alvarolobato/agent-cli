package discovery

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/alvarolobato/agent-cli/internal/agent"
)

const (
	defaultSystemdUnitPath = "/etc/systemd/system/elastic-agent.service"
	defaultLaunchdPlist    = "/Library/LaunchDaemons/co.elastic.elastic-agent.plist"
)

type serviceScanner struct {
	readFile       func(string) ([]byte, error)
	systemdUnit    string
	launchdPlist   string
	installScanner *installDirScanner
}

// NewServiceScanner creates a service-definition discovery strategy.
func NewServiceScanner() Strategy {
	return &serviceScanner{
		readFile:       os.ReadFile,
		systemdUnit:    defaultSystemdUnitPath,
		launchdPlist:   defaultLaunchdPlist,
		installScanner: NewInstallDirScannerWithFS(defaultInstallDirs(runtime.GOOS), osInstallFS{}),
	}
}

func (s *serviceScanner) Discover(ctx context.Context) ([]agent.Agent, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	path, err := s.extractInstallPath()
	if err != nil || strings.TrimSpace(path) == "" {
		return nil, nil
	}

	discovered, ok, err := s.installScanner.scanInstallDir(path)
	if err != nil || !ok {
		return nil, err
	}

	return []agent.Agent{
		stubAgent{id: discovered.InstallPath, kind: discovered.Type},
	}, nil
}

func (s *serviceScanner) extractInstallPath() (string, error) {
	if runtime.GOOS == "darwin" {
		raw, err := s.readFile(s.launchdPlist)
		if err != nil {
			return "", err
		}
		return parseLaunchdWorkingDirectory(raw)
	}

	raw, err := s.readFile(s.systemdUnit)
	if err != nil {
		return "", err
	}
	return parseSystemdWorkingDirectory(raw), nil
}

func parseSystemdWorkingDirectory(data []byte) string {
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";") {
			continue
		}
		key, value, ok := strings.Cut(trimmed, "=")
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(key), "WorkingDirectory") {
			return filepath.Clean(strings.Trim(strings.TrimSpace(value), `"'`))
		}
	}
	return ""
}

func parseLaunchdWorkingDirectory(data []byte) (string, error) {
	var parsed launchdPlist
	if err := xml.Unmarshal(data, &parsed); err != nil {
		return "", fmt.Errorf("parse launchd plist: %w", err)
	}

	items := parsed.Dict.Items
	for i := 0; i < len(items)-1; i++ {
		key, keyOK := items[i].(launchdKey)
		if !keyOK || strings.TrimSpace(key.Value) != "WorkingDirectory" {
			continue
		}
		value, valueOK := items[i+1].(launchdString)
		if !valueOK {
			continue
		}
		return filepath.Clean(strings.TrimSpace(value.Value)), nil
	}
	return "", nil
}

type launchdPlist struct {
	Dict launchdDict `xml:"dict"`
}

type launchdDict struct {
	Items []interface{} `xml:",any"`
}

type launchdKey struct {
	Value string `xml:",chardata"`
}

type launchdString struct {
	Value string `xml:",chardata"`
}

func (d *launchdDict) UnmarshalXML(dec *xml.Decoder, start xml.StartElement) error {
	for {
		tok, err := dec.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		switch element := tok.(type) {
		case xml.StartElement:
			switch element.Name.Local {
			case "key":
				var key launchdKey
				if err := dec.DecodeElement(&key, &element); err != nil {
					return err
				}
				d.Items = append(d.Items, key)
			case "string":
				var value launchdString
				if err := dec.DecodeElement(&value, &element); err != nil {
					return err
				}
				d.Items = append(d.Items, value)
			default:
				var skip struct{}
				if err := dec.DecodeElement(&skip, &element); err != nil {
					return err
				}
			}
		case xml.EndElement:
			if element.Name.Local == start.Name.Local {
				return nil
			}
		}
	}
}

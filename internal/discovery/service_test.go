package discovery

import "testing"

func TestParseSystemdWorkingDirectory(t *testing.T) {
	raw := []byte(`
[Unit]
Description=Elastic Agent

[Service]
WorkingDirectory=/opt/Elastic/Agent
ExecStart=/opt/Elastic/Agent/elastic-agent run
`)

	got := parseSystemdWorkingDirectory(raw)
	if got != "/opt/Elastic/Agent" {
		t.Fatalf("parseSystemdWorkingDirectory() = %q, want /opt/Elastic/Agent", got)
	}
}

func TestParseLaunchdWorkingDirectory(t *testing.T) {
	raw := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>Label</key>
    <string>co.elastic.elastic-agent</string>
    <key>WorkingDirectory</key>
    <string>/Library/Elastic/Agent</string>
  </dict>
</plist>`)

	got, err := parseLaunchdWorkingDirectory(raw)
	if err != nil {
		t.Fatalf("parseLaunchdWorkingDirectory() error = %v", err)
	}
	if got != "/Library/Elastic/Agent" {
		t.Fatalf("parseLaunchdWorkingDirectory() = %q, want /Library/Elastic/Agent", got)
	}
}

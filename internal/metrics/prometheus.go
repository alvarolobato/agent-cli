package metrics

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// OTelSnapshot is a normalized view of key OTel collector runtime metrics.
type OTelSnapshot struct {
	Receivers  map[string]OTelComponentMetrics
	Processors map[string]OTelComponentMetrics
	Exporters  map[string]OTelComponentMetrics
}

// OTelComponentMetrics contains the key counters used for health and throughput.
type OTelComponentMetrics struct {
	Accepted   float64
	Sent       float64
	Dropped    float64
	SendFailed float64
}

// CollectOTelPrometheus scrapes and parses OTel collector Prometheus metrics.
func (c *Collector) CollectOTelPrometheus(ctx context.Context, endpoint string) (*OTelSnapshot, error) {
	url := strings.TrimSpace(endpoint)
	if url == "" {
		return nil, fmt.Errorf("prometheus endpoint is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build prometheus request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request prometheus endpoint: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("prometheus endpoint returned %s", resp.Status)
	}

	snapshot := &OTelSnapshot{
		Receivers:  map[string]OTelComponentMetrics{},
		Processors: map[string]OTelComponentMetrics{},
		Exporters:  map[string]OTelComponentMetrics{},
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		metricName, labels, value, ok := parsePrometheusLine(line)
		if !ok {
			continue
		}

		switch {
		case strings.HasPrefix(metricName, "otelcol_receiver_accepted_"):
			receiver := strings.TrimSpace(labels["receiver"])
			if receiver == "" {
				continue
			}
			m := snapshot.Receivers[receiver]
			m.Accepted += value
			snapshot.Receivers[receiver] = m
		case strings.HasPrefix(metricName, "otelcol_exporter_sent_"):
			exporter := strings.TrimSpace(labels["exporter"])
			if exporter == "" {
				continue
			}
			m := snapshot.Exporters[exporter]
			m.Sent += value
			snapshot.Exporters[exporter] = m
		case strings.HasPrefix(metricName, "otelcol_processor_dropped_"):
			processor := strings.TrimSpace(labels["processor"])
			if processor == "" {
				continue
			}
			m := snapshot.Processors[processor]
			m.Dropped += value
			snapshot.Processors[processor] = m
		case strings.HasPrefix(metricName, "otelcol_exporter_send_failed_"):
			exporter := strings.TrimSpace(labels["exporter"])
			if exporter == "" {
				continue
			}
			m := snapshot.Exporters[exporter]
			m.SendFailed += value
			snapshot.Exporters[exporter] = m
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read prometheus response: %w", err)
	}

	return snapshot, nil
}

func parsePrometheusLine(line string) (string, map[string]string, float64, bool) {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return "", nil, 0, false
	}

	rawMetric := parts[0]
	valuePart := parts[len(parts)-1]
	value, err := strconv.ParseFloat(valuePart, 64)
	if err != nil {
		return "", nil, 0, false
	}

	metricName := rawMetric
	labels := map[string]string{}
	if open := strings.Index(rawMetric, "{"); open >= 0 {
		close := strings.LastIndex(rawMetric, "}")
		if close <= open {
			return "", nil, 0, false
		}
		metricName = rawMetric[:open]
		labels = parsePrometheusLabels(rawMetric[open+1 : close])
	}

	return metricName, labels, value, true
}

func parsePrometheusLabels(raw string) map[string]string {
	out := map[string]string{}
	for _, token := range splitLabels(raw) {
		pair := strings.SplitN(strings.TrimSpace(token), "=", 2)
		if len(pair) != 2 {
			continue
		}
		key := strings.TrimSpace(pair[0])
		if key == "" {
			continue
		}
		value := strings.Trim(strings.TrimSpace(pair[1]), `"`)
		out[key] = value
	}
	return out
}

func splitLabels(raw string) []string {
	parts := []string{}
	if strings.TrimSpace(raw) == "" {
		return parts
	}

	current := strings.Builder{}
	inQuotes := false
	for _, r := range raw {
		switch r {
		case '"':
			inQuotes = !inQuotes
			current.WriteRune(r)
		case ',':
			if inQuotes {
				current.WriteRune(r)
				continue
			}
			parts = append(parts, current.String())
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}
	parts = append(parts, current.String())
	return parts
}

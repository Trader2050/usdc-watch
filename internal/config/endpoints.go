package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

// Endpoint represents a JSON-RPC endpoint definition.
type Endpoint struct {
	Name string
	URL  string
}

// LoadEndpoints parses the [[rpc.endpoints]] blocks from a TOML-style configuration file.
func LoadEndpoints(path string) ([]Endpoint, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open endpoints file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var (
		endpoints    []Endpoint
		current      Endpoint
		inCollection bool
		lineNo       int
	)

	flush := func(force bool) error {
		if !inCollection {
			return nil
		}
		if strings.TrimSpace(current.URL) == "" {
			if force {
				return fmt.Errorf("endpoint missing url near line %d", lineNo)
			}
			current = Endpoint{}
			inCollection = false
			return nil
		}
		if strings.TrimSpace(current.Name) == "" {
			current.Name = fmt.Sprintf("endpoint-%d", len(endpoints)+1)
		}
		endpoints = append(endpoints, current)
		current = Endpoint{}
		inCollection = false
		return nil
	}

	for scanner.Scan() {
		lineNo++
		rawLine := scanner.Text()
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[[") && strings.HasSuffix(line, "]]") {
			if err := flush(true); err != nil {
				return nil, err
			}
			if line == "[[rpc.endpoints]]" {
				inCollection = true
				current = Endpoint{}
			} else {
				inCollection = false
			}
			continue
		}
		if !inCollection {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid line %d: %s", lineNo, rawLine)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}
		switch key {
		case "name":
			current.Name = value
		case "url":
			current.URL = value
		default:
			continue
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan endpoints file: %w", err)
	}
	if err := flush(true); err != nil {
		return nil, err
	}
	if len(endpoints) == 0 {
		return nil, errors.New("no endpoints found in configuration")
	}
	return endpoints, nil
}

package profile

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/metacubex/mihomo/common/convert"
	"go.yaml.in/yaml/v3"

	"ctun/internal/config"
)

const (
	maxBodyBytes  = 16 << 20
	maxImportSize = 8 << 20
)

var ErrNoProxies = errors.New("profile: no servers found")

func Parse(body []byte) (*config.Profile, error) {
	body, err := gunzipIfNeeded(body)
	if err != nil {
		return nil, err
	}
	body = bytes.TrimSpace(bytes.TrimPrefix(body, []byte("\xef\xbb\xbf")))

	var p *config.Profile
	if layer, ok := parseClash(body); ok {
		p = layer
	} else if looksLikeJSON(body) {
		p = &config.Profile{Proxies: parseJSON(body)}
	} else {
		// The converter only reports "format invalid": treat that as "no servers".
		proxies, err := convert.ConvertsV2Ray(decodeLinks(body))
		if err != nil {
			return nil, ErrNoProxies
		}
		p = &config.Profile{Proxies: proxies}
	}

	if len(p.Proxies) == 0 && len(p.ProxyProviders) == 0 {
		return nil, ErrNoProxies
	}
	dedupeNames(p.Proxies)
	return p, nil
}

func gunzipIfNeeded(body []byte) ([]byte, error) {
	if len(body) < 2 || body[0] != 0x1f || body[1] != 0x8b {
		return body, nil
	}
	r, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("profile: gzip: %w", err)
	}
	defer r.Close()
	out, err := io.ReadAll(io.LimitReader(r, maxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("profile: gzip: %w", err)
	}
	return out, nil
}

type clashDocument struct {
	Proxies        []map[string]any `yaml:"proxies"`
	ProxyGroups    []map[string]any `yaml:"proxy-groups"`
	ProxyProviders map[string]any   `yaml:"proxy-providers"`
	RuleProviders  map[string]any   `yaml:"rule-providers"`
	Rules          []string         `yaml:"rules"`
}

func parseClash(body []byte) (*config.Profile, bool) {
	var doc clashDocument
	if err := yaml.Unmarshal(body, &doc); err != nil {
		return nil, false
	}
	if len(doc.Proxies) == 0 && len(doc.ProxyProviders) == 0 {
		return nil, false
	}
	return &config.Profile{
		Proxies:        doc.Proxies,
		ProxyGroups:    doc.ProxyGroups,
		ProxyProviders: doc.ProxyProviders,
		RuleProviders:  doc.RuleProviders,
		Rules:          doc.Rules,
	}, true
}

func looksLikeJSON(body []byte) bool {
	return len(body) > 0 && (body[0] == '{' || body[0] == '[')
}

func decodeLinks(body []byte) []byte {
	if bytes.Contains(body, []byte("://")) {
		return body
	}
	if decoded, ok := decodeBase64(string(bytes.Join(bytes.Fields(body), nil))); ok {
		return decoded
	}
	return body
}

func decodeBase64(s string) ([]byte, bool) {
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		if decoded, err := enc.DecodeString(s); err == nil {
			return decoded, true
		}
	}
	return nil, false
}

func dedupeNames(proxies []map[string]any) {
	used := make(map[string]bool, len(proxies))
	for _, p := range proxies {
		name, _ := p["name"].(string)
		if name == "" {
			name = fmt.Sprintf("%v:%v", p["server"], p["port"])
		}
		unique := name
		for i := 2; used[unique]; i++ {
			unique = name + " " + strconv.Itoa(i)
		}
		used[unique] = true
		p["name"] = unique
	}
}

func Describe(proxy map[string]any) string {
	parts := []string{strings.ToUpper(str(proxy["type"]))}
	switch {
	case proxy["reality-opts"] != nil:
		parts = append(parts, "Reality")
	case proxy["tls"] == true:
		parts = append(parts, "TLS")
	}
	if network := str(proxy["network"]); network != "" && network != "tcp" {
		parts = append(parts, network)
	}
	return strings.Join(parts, " · ")
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func ReadConfig(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("profile: open config: %w", err)
	}
	defer f.Close()
	content, err := io.ReadAll(io.LimitReader(f, maxImportSize+1))
	if err != nil {
		return nil, fmt.Errorf("profile: read config: %w", err)
	}
	if len(content) > maxImportSize {
		return nil, fmt.Errorf("profile: file is larger than %d MB", maxImportSize>>20)
	}
	return content, nil
}

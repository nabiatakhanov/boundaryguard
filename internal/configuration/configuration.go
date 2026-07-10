// Package configuration parses and validates BoundaryGuard policies.
// A Config is immutable after Parse returns and can be shared by analyzers.
package configuration

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Layer assigns canonical import path patterns to a named architectural layer.
type Layer struct {
	Name     string   `json:"name"`
	Packages []string `json:"packages"`
}

// DependencyRule forbids a direct or transitive dependency between two layers.
type DependencyRule struct {
	ID        string `json:"id"`
	FromLayer string `json:"from_layer"`
	ToLayer   string `json:"to_layer"`
	Kind      string `json:"kind"`
}

// ForbiddenSymbol forbids references to a package-level function or variable.
type ForbiddenSymbol struct {
	ID       string   `json:"id"`
	Packages []string `json:"packages"`
	Symbol   string   `json:"symbol"`
}

type policy struct {
	Version          int               `json:"version"`
	Layers           []Layer           `json:"layers"`
	DependencyRules  []DependencyRule  `json:"dependency_rules"`
	ForbiddenSymbols []ForbiddenSymbol `json:"forbidden_symbols"`
}

// Config holds a validated, normalized policy. Its zero value is not valid;
// construct one with Parse or Load.
type Config struct {
	policy      policy
	fingerprint string
}

// Parse rejects unknown fields, duplicate object keys, trailing JSON values,
// unsupported patterns, overlapping layers and invalid rule references.
func Parse(data []byte) (*Config, error) {
	if err := checkJSON(data); err != nil {
		return nil, fmt.Errorf("configuration JSON: %w", err)
	}
	var p policy
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("configuration JSON: %w", err)
	}
	if err := validate(&p); err != nil {
		return nil, fmt.Errorf("configuration: %w", err)
	}
	normalize(&p)
	canonical, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("encode normalized configuration: %w", err)
	}
	sum := sha256.Sum256(canonical)
	return &Config{policy: p, fingerprint: hex.EncodeToString(sum[:])}, nil
}

// Load resolves a config filename relative to the invocation directory once.
// The resulting policy has no filesystem dependencies during analysis.
func Load(path string) (*Config, error) {
	if path == "" {
		return nil, fmt.Errorf("configuration: -config must name a JSON file")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("configuration path %q: %w", path, err)
	}
	data, err := os.ReadFile(absolute)
	if err != nil {
		return nil, fmt.Errorf("configuration file %q: %w", absolute, err)
	}
	c, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("configuration file %q: %w", absolute, err)
	}
	return c, nil
}

// Layers returns an independent copy sorted by layer name.
func (c *Config) Layers() []Layer {
	out := append([]Layer(nil), c.policy.Layers...)
	for i := range out {
		out[i].Packages = append([]string(nil), out[i].Packages...)
	}
	return out
}

// DependencyRules returns an independent copy sorted by rule ID.
func (c *Config) DependencyRules() []DependencyRule {
	return append([]DependencyRule(nil), c.policy.DependencyRules...)
}

// ForbiddenSymbols returns an independent copy sorted by rule ID.
func (c *Config) ForbiddenSymbols() []ForbiddenSymbol {
	out := append([]ForbiddenSymbol(nil), c.policy.ForbiddenSymbols...)
	for i := range out {
		out[i].Packages = append([]string(nil), out[i].Packages...)
	}
	return out
}

// LayerFor returns the unique layer name, or an empty string if unclassified.
func (c *Config) LayerFor(path string) string {
	for _, layer := range c.policy.Layers {
		for _, pattern := range layer.Packages {
			if Match(pattern, path) {
				return layer.Name
			}
		}
	}
	return ""
}

// Fingerprint identifies the complete normalized policy, including symbol rules.
func (c *Config) Fingerprint() string { return c.fingerprint }

// Match compares a validated exact pattern or /... pattern with an import path.
// A recursive pattern includes its root package and respects segment boundaries.
func Match(pattern, path string) bool {
	if strings.HasSuffix(pattern, "/...") {
		root := strings.TrimSuffix(pattern, "/...")
		return path == root || strings.HasPrefix(path, root+"/")
	}
	return path == pattern
}

func validate(p *policy) error {
	if p.Version != 1 {
		return fmt.Errorf("version must be 1, got %d", p.Version)
	}
	layers := make(map[string]bool, len(p.Layers))
	for _, layer := range p.Layers {
		if !validLabel(layer.Name) {
			return fmt.Errorf("invalid layer name %q", layer.Name)
		}
		if layers[layer.Name] {
			return fmt.Errorf("duplicate layer name %q", layer.Name)
		}
		layers[layer.Name] = true
		if err := validatePatterns(layer.Packages); err != nil {
			return fmt.Errorf("layer %q: %w", layer.Name, err)
		}
	}
	for i, a := range p.Layers {
		for _, b := range p.Layers[i+1:] {
			for _, ap := range a.Packages {
				for _, bp := range b.Packages {
					if overlap(ap, bp) {
						return fmt.Errorf("overlapping layers %q (%s) and %q (%s)", a.Name, ap, b.Name, bp)
					}
				}
			}
		}
	}
	ids := make(map[string]bool, len(p.DependencyRules)+len(p.ForbiddenSymbols))
	checkID := func(id string) error {
		if !validLabel(id) {
			return fmt.Errorf("invalid rule id %q", id)
		}
		if ids[id] {
			return fmt.Errorf("duplicate rule id %q", id)
		}
		ids[id] = true
		return nil
	}
	for _, rule := range p.DependencyRules {
		if err := checkID(rule.ID); err != nil {
			return err
		}
		if !layers[rule.FromLayer] || !layers[rule.ToLayer] {
			return fmt.Errorf("rule %q references missing layer (%q -> %q)", rule.ID, rule.FromLayer, rule.ToLayer)
		}
		if rule.Kind != "direct" && rule.Kind != "transitive" {
			return fmt.Errorf("rule %q kind must be direct or transitive", rule.ID)
		}
	}
	for _, rule := range p.ForbiddenSymbols {
		if err := checkID(rule.ID); err != nil {
			return err
		}
		if err := validatePatterns(rule.Packages); err != nil {
			return fmt.Errorf("rule %q: %w", rule.ID, err)
		}
		dot := strings.LastIndexByte(rule.Symbol, '.')
		if dot < 1 || !validPath(rule.Symbol[:dot]) || !token.IsIdentifier(rule.Symbol[dot+1:]) || rule.Symbol[dot+1:] == "_" {
			return fmt.Errorf("rule %q symbol must be package-path.Identifier, got %q", rule.ID, rule.Symbol)
		}
	}
	return nil
}

func validLabel(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.') {
			return false
		}
	}
	return true
}

func validatePatterns(patterns []string) error {
	if len(patterns) == 0 {
		return fmt.Errorf("packages must contain at least one import path pattern")
	}
	seen := make(map[string]bool, len(patterns))
	for _, pattern := range patterns {
		root := strings.TrimSuffix(pattern, "/...")
		if !validPath(root) || strings.Contains(root, "...") {
			return fmt.Errorf("unsupported package pattern %q: use an exact import path or path/...", pattern)
		}
		if seen[pattern] {
			return fmt.Errorf("duplicate package pattern %q", pattern)
		}
		seen[pattern] = true
	}
	return nil
}

func validPath(path string) bool {
	if path == "" || !utf8.ValidString(path) {
		return false
	}
	for _, segment := range strings.Split(path, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	for _, r := range path {
		if !unicode.IsGraphic(r) || unicode.IsSpace(r) || r == utf8.RuneError || strings.ContainsRune("!\"#$%&'()*,:;<=>?@[\\]^`{|}", r) {
			return false
		}
	}
	return true
}

func overlap(a, b string) bool {
	ar := strings.TrimSuffix(a, "/...")
	br := strings.TrimSuffix(b, "/...")
	return ar == br || strings.HasSuffix(a, "/...") && Match(a, br) || strings.HasSuffix(b, "/...") && Match(b, ar)
}

func normalize(p *policy) {
	for i := range p.Layers {
		sort.Strings(p.Layers[i].Packages)
	}
	for i := range p.ForbiddenSymbols {
		sort.Strings(p.ForbiddenSymbols[i].Packages)
	}
	sort.Slice(p.Layers, func(i, j int) bool { return p.Layers[i].Name < p.Layers[j].Name })
	sort.Slice(p.DependencyRules, func(i, j int) bool { return p.DependencyRules[i].ID < p.DependencyRules[j].ID })
	sort.Slice(p.ForbiddenSymbols, func(i, j int) bool { return p.ForbiddenSymbols[i].ID < p.ForbiddenSymbols[j].ID })
	// Normalize absent and empty optional arrays to the same representation.
	if p.Layers == nil {
		p.Layers = []Layer{}
	}
	if p.DependencyRules == nil {
		p.DependencyRules = []DependencyRule{}
	}
	if p.ForbiddenSymbols == nil {
		p.ForbiddenSymbols = []ForbiddenSymbol{}
	}
}

// checkJSON rejects duplicates before encoding/json can silently overwrite them.
// It also enforces exact field spelling (encoding/json otherwise folds case).
func checkJSON(data []byte) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err := jsonValue(d, "root", 0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return err
	}
	return nil
}

func jsonValue(d *json.Decoder, context string, depth int) error {
	if depth > 32 {
		return fmt.Errorf("configuration JSON nesting exceeds 32")
	}
	t, err := d.Token()
	if err != nil {
		return err
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	if delim == '{' {
		seen := map[string]bool{}
		for d.More() {
			keyToken, err := d.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("object key must be a string")
			}
			if seen[key] {
				return fmt.Errorf("duplicate JSON key %q", key)
			}
			seen[key] = true
			if !knownField(context, key) {
				return fmt.Errorf("unknown field %q in %s", key, context)
			}
			if err := jsonValue(d, key, depth+1); err != nil {
				return err
			}
		}
	} else if delim == '[' {
		for d.More() {
			if err := jsonValue(d, context, depth+1); err != nil {
				return err
			}
		}
	} else {
		return fmt.Errorf("unexpected delimiter %q", delim)
	}
	_, err = d.Token()
	return err
}

func knownField(context, key string) bool {
	var fields string
	switch context {
	case "root":
		fields = " version layers dependency_rules forbidden_symbols "
	case "layers":
		fields = " name packages "
	case "dependency_rules":
		fields = " id from_layer to_layer kind "
	case "forbidden_symbols":
		fields = " id packages symbol "
	default:
		return false
	}
	return strings.Contains(fields, " "+key+" ")
}

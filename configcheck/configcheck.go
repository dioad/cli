// Package configcheck reports configuration problems as findings, each with
// the path of the field it concerns, so a command can report every problem
// in a configuration at once instead of stopping at the first.
//
// It also holds the decode hook used to load configuration and a strict
// decoder built on it. The package depends only on the standard library,
// mapstructure and github.com/dioad/util, so libraries and plugins can
// validate configuration the same way the cli loader does without importing
// cobra or viper.
package configcheck

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// Finding is one configuration problem.
type Finding struct {
	// Path locates the field, for example "traffic.tunnels[2].subdomain".
	// It is empty for a problem with the configuration as a whole.
	Path string
	// Message describes the problem.
	Message string
}

// String returns the finding as "path: message", or just the message when
// Path is empty.
func (f Finding) String() string {
	if f.Path == "" {
		return f.Message
	}
	return f.Path + ": " + f.Message
}

// Error is an error made of one or more findings.
type Error struct {
	Findings []Finding
}

// Error returns the findings one per line.
func (e *Error) Error() string {
	lines := make([]string, len(e.Findings))
	for i, f := range e.Findings {
		lines[i] = f.String()
	}
	return strings.Join(lines, "\n")
}

// Collector accumulates findings. A collector derived with Field, Index or
// Key adds findings under a longer path but shares its findings with the
// collector it came from, so a check can be handed a collector for its own
// sub-tree and write paths relative to it.
//
// The zero value is an empty collector at the root path. Collectors are
// safe for concurrent use once created, but a zero-value collector must not
// be first used from several goroutines at once.
type Collector struct {
	path string
	list *findingList
}

type findingList struct {
	mu    sync.Mutex
	items []Finding
}

// NewCollector returns an empty collector at the root path.
func NewCollector() *Collector {
	return &Collector{list: &findingList{}}
}

func (c *Collector) shared() *findingList {
	if c.list == nil {
		c.list = &findingList{}
	}
	return c.list
}

func (c *Collector) derive(path string) *Collector {
	return &Collector{path: path, list: c.shared()}
}

// Path returns the path findings added to c are reported under.
func (c *Collector) Path() string {
	return c.path
}

// Field returns a collector for the struct field or map entry name, joined
// with a dot: "traffic" then "tunnels" gives "traffic.tunnels".
func (c *Collector) Field(name string) *Collector {
	return c.derive(JoinPath(c.path, name))
}

// Index returns a collector for element i of a list: "tunnels[2]".
func (c *Collector) Index(i int) *Collector {
	return c.derive(c.path + "[" + strconv.Itoa(i) + "]")
}

// Key returns a collector for the map entry k in bracket form, "routes[dmarc]",
// matching the paths Decode reports for map entries.
func (c *Collector) Key(k string) *Collector {
	return c.derive(c.path + "[" + k + "]")
}

// Errorf adds a finding at c's path.
func (c *Collector) Errorf(format string, args ...any) {
	c.append(Finding{Path: c.path, Message: fmt.Sprintf(format, args...)})
}

// Add adds err to c. An *Error, such as one returned by Decode, contributes
// each of its findings with c's path prefixed; any other error becomes one
// finding at c's path. A nil err adds nothing.
func (c *Collector) Add(err error) {
	if err == nil {
		return
	}
	var ce *Error
	if !errors.As(err, &ce) {
		c.append(Finding{Path: c.path, Message: err.Error()})
		return
	}
	for _, f := range ce.Findings {
		c.append(Finding{Path: JoinPath(c.path, f.Path), Message: f.Message})
	}
}

func (c *Collector) append(f Finding) {
	l := c.shared()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.items = append(l.items, f)
}

// Findings returns every finding added through c or any collector sharing
// its findings, in the order they were added.
func (c *Collector) Findings() []Finding {
	l := c.shared()
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]Finding(nil), l.items...)
}

// Err returns the findings as an *Error, or nil if there are none.
func (c *Collector) Err() error {
	findings := c.Findings()
	if len(findings) == 0 {
		return nil
	}
	return &Error{Findings: findings}
}

// JoinPath joins a path and a path relative to it. A relative path that
// starts with an index or key ("[2].name") is appended without a dot.
func JoinPath(base, rel string) string {
	switch {
	case base == "":
		return rel
	case rel == "":
		return base
	case strings.HasPrefix(rel, "["):
		return base + rel
	default:
		return base + "." + rel
	}
}

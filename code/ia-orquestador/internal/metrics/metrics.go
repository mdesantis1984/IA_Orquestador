// Package metrics provides in-process Prometheus-compatible metrics for the MCP orchestrator.
package metrics

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// Metrics holds all counters and gauges.
type Metrics struct {
	toolCallsTotal sync.Map // key: "skill|mode" → *int64
	sessionsTotal  int64    // atomic
	skillsLoaded   int64    // atomic
}

// New creates a new Metrics instance.
func New() *Metrics {
	return &Metrics{}
}

// IncrToolCall increments the counter for the (skill, mode) pair.
func (m *Metrics) IncrToolCall(skill, mode string) {
	key := skill + "|" + mode
	v, _ := m.toolCallsTotal.LoadOrStore(key, new(int64))
	atomic.AddInt64(v.(*int64), 1)
}

// IncrSession increments the total sessions counter.
func (m *Metrics) IncrSession() {
	atomic.AddInt64(&m.sessionsTotal, 1)
}

// SetSkillsLoaded sets the gauge for currently loaded skills.
func (m *Metrics) SetSkillsLoaded(n int64) {
	atomic.StoreInt64(&m.skillsLoaded, n)
}

// Handler returns an HTTP handler that emits Prometheus text format on GET /metrics.
func (m *Metrics) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		type kv struct {
			k string
			v int64
		}
		var lines []kv
		m.toolCallsTotal.Range(func(k, v interface{}) bool {
			lines = append(lines, kv{k: k.(string), v: atomic.LoadInt64(v.(*int64))})
			return true
		})
		sort.Slice(lines, func(i, j int) bool { return lines[i].k < lines[j].k })

		fmt.Fprintln(w, "# HELP mcp_tool_calls_total Total tool calls by skill and execution mode")
		fmt.Fprintln(w, "# TYPE mcp_tool_calls_total counter")
		for _, l := range lines {
			idx := strings.Index(l.k, "|")
			if idx < 0 {
				continue
			}
			skill := l.k[:idx]
			mode := l.k[idx+1:]
			fmt.Fprintf(w, "mcp_tool_calls_total{skill=%q,mode=%q} %d\n", skill, mode, l.v)
		}

		fmt.Fprintln(w, "# HELP mcp_sessions_total Total MCP sessions initialized")
		fmt.Fprintln(w, "# TYPE mcp_sessions_total counter")
		fmt.Fprintf(w, "mcp_sessions_total %d\n", atomic.LoadInt64(&m.sessionsTotal))

		fmt.Fprintln(w, "# HELP mcp_skills_loaded Number of currently loaded skills")
		fmt.Fprintln(w, "# TYPE mcp_skills_loaded gauge")
		fmt.Fprintf(w, "mcp_skills_loaded %d\n", atomic.LoadInt64(&m.skillsLoaded))
	}
}

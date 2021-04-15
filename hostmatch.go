package smux

import (
	"fmt"
	"regexp"
	"strings"
)

var hostSegmentRegex = regexp.MustCompile(`^([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])$`)

func verifyHostname(host string) bool {
	allowStar := true
	for _, s := range strings.Split(host, ".") {
		if s == "*" && !allowStar {
			return false
		}
		if s != "*" {
			if !hostSegmentRegex.MatchString(s) {
				return false
			}
			allowStar = false
		}
	}

	return true
}

func verifyHost(host string) bool {
	return host == "" || verifyHostname(host)
}

type hostentry struct {
	host         string
	segs         []string
	numwildcards int
	t            PathRouter
}

func (h1 hostentry) Intersects(h2 hostentry) bool {
	h1splt := strings.Split(h1.host, ".")
	h2splt := strings.Split(h2.host, ".")

	if len(h1splt) != len(h2splt) {
		return false
	}

	for i := range h1splt {
		hh1 := h1splt[i]
		hh2 := h2splt[i]

		if hh1 == hh2 || hh1 == "*" || hh2 == "*" {
			continue
		}

		return false
	}

	return true
}

// Compile Makes easier to find the correct host
func (h *hostentry) Compile() {
	h.segs = strings.Split(h.host, ".")
	h.numwildcards = 0
	for _, s := range h.segs {
		if s == "*" {
			h.numwildcards += 1
		}
	}
}

func (h hostentry) Match(hostname string, ctx *Context) bool {
	si := 0
	ctx.HostParams = make([]string, h.numwildcards)
	j := -1
	for i := range h.segs {
		j = strings.Index(hostname, ".")

		if j == -1 {
			j = len(hostname)
		}

		if h.segs[i] != "*" && h.segs[i] != hostname[:j] {
			return false
		}

		if h.segs[i] == "*" {
			ctx.HostParams[si] = hostname[:j]
			si += 1
		}

		if len(hostname) == j {
			if i < len(h.segs)-1 {
				return false
			} else {
				return true
			}
		}
		hostname = hostname[j+1:]
	}

	return false
}

type HostRouter struct {
	hosts   []hostentry
	allhost PathRouter
}

func NewHostRouter() *HostRouter {
	return &HostRouter{allhost: &trie{}}
}

func (h *HostRouter) AddHostname(hostname string) error {
	if !verifyHost(hostname) {
		return fmt.Errorf("invalid host")
	}

	hentry := hostentry{host: hostname, t: &trie{}}
	for i := range h.hosts {
		if hentry.Intersects(h.hosts[i]) {
			return fmt.Errorf("host in intersection with %v", h.hosts[i].host)
		}
	}
	h.hosts = append(h.hosts, hentry)
	h.hosts[len(h.hosts)-1].Compile()

	return nil
}

// Add Adds a new route to the HostRouter
// Accepts only
// {something} string not pointed
//   Example: {a}.local matches foo.local and bar.local, but not foo.bar.local
// {*} catch all on left
//   Example: {*}.local matches foo.local, bar.local and foo.bar.local
func (h *HostRouter) AddRoute(rt *Route) error {
	if rt == nil {
		return nil
	}
	if !verifyPath(rt.path) {
		return fmt.Errorf("invalid path")
	}

	if len(rt.methods) == 0 {
		return fmt.Errorf("no method on route")
	}

	// Empty host is considered 'All hosts'
	if rt.host == "" {
		return h.allhost.Add(rt)
	}

	for i := range h.hosts {
		if h.hosts[i].host == rt.host {
			return h.hosts[i].t.Add(rt)
		}
	}

	return fmt.Errorf("host not found")
}

func (h HostRouter) Get(hostname, path string, ctx *Context) MatchResult {
	for i := range h.hosts {
		if h.hosts[i].Match(hostname, ctx) {
			r := h.hosts[i].t.Get(path, ctx)
			if r != nil {
				return r
			}
		}
	}

	return h.allhost.Get(path, ctx)
}

// func (h HostRouter) GetAll(hostname, path string, ctx *Context) []MatchResult {
// 	for i := range h.hosts {
// 		if h.hosts[i].Match(hostname, ctx) {
// 			return h.hosts[i].t.GetAll(path, ctx)
// 		}
// 	}

// 	return h.allhost.GetAll(path, ctx)
// }

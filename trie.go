package smux

import (
	"fmt"
	"net/url"
	"strings"
)

func verifyPath(path string) bool {
	r1 := path
	if strings.HasSuffix(r1, "{*}") {
		r1 = strings.TrimRight(path, "{*}")
	}
	r1 = bracketSimplistic.ReplaceAllString(r1, "")
	u, err := url.ParseRequestURI(r1)
	if err != nil {
		return false
	}
	return u.EscapedPath() == r1
}

func ParsePath(path string) ([]segment, error) {
	if !verifyPath(path) {
		return nil, fmt.Errorf("invalid path")
	}

	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	ps := strings.Split(path, "/")

	segs := make([]segment, len(ps))

	for i := range ps {
		seg, err := createSegment(ps[i])
		if err != nil {
			return nil, err
		}

		if seg.CatchAll() && i != len(ps)-1 {
			return nil, fmt.Errorf("catchall must be the last segment")
		}

		segs[i] = seg
	}

	return segs, nil
}

type node struct {
	seg     segment
	methods map[string]*Route
	nodes   []*node
}

func (n node) Methods() map[string]*Route {
	return n.methods
}

type trie struct {
	node
	depth     int
	maxparams int
}

func (t *trie) Add(r *Route) error {
	if r == nil {
		return nil
	}

	path, err := ParsePath(r.path)

	if err != nil {
		return err
	}

	if len(r.methods) == 0 {
		return fmt.Errorf("no method on path")
	}

	n := &t.node
	// if n.nodes == nil {
	// 	n.nodes = make(map[string]*node)
	// }

	d := 0
	maxparams := 0

	l := len(path)
	for i, s := range path {
		inode := n

		// Tests if there is a there is a catch all to hide things
		for j := range inode.nodes {
			if inode.nodes[j].seg.CatchAll() {
				return fmt.Errorf("there is a catch all hiding that path")
			}
		}

		// Tests if there are other handlers and you're adding a catch all
		if len(inode.nodes) > 0 && s.CatchAll() {
			return fmt.Errorf("this catch all overlaps other paths")
		}

		// Search for existent node
		for j, nn := range inode.nodes {
			if nn.seg.Comparable() == s.Comparable() {
				n = n.nodes[j]
				break
			}
		}
		// if nn, found := inode.nodes[s.Comparable()]; found {
		// 	n = nn
		// }

		// Not found - Add new node
		if n == inode {
			n = &node{seg: s, methods: make(map[string]*Route)}
			// inode.nodes[s.Comparable()] = n
			inode.nodes = append(inode.nodes, n)
			// n = &inode.nodes[len(inode.nodes)-1]
		}

		//backtrack = append(backtrack, n)
		d += 1
		maxparams += n.seg.NumVars()

		// Must set value
		if i == l-1 {
			// Must not add route to already set method
			for m := range n.methods {
				if _, found := r.methods[m]; found {
					return fmt.Errorf("Path segment already handled")
				}
			}

			for m := range r.methods {
				n.methods[m] = r
			}
		}
	}

	if t.depth < d {
		t.depth = d
	}

	if t.maxparams < maxparams {
		t.maxparams = maxparams
	}

	return nil
}

// The fast case (no recursion)
func (t trie) Get(path string, ctx *Context) MatchResult {
	n := &t.node

	originalpath := path
	ior := 0

	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
		ior += 1
	}

	var match bool
	// var parms []PathParam
	ctx.pathParams = make([]PathParam, 0, t.maxparams)
	parms := &ctx.pathParams

	for {
		i := strings.IndexRune(path, '/')
		ii := i
		if i < 0 {
			ii = len(path)
		}

		inode := n

		// var parms []PathParam
		// Find node which matches p
		for j := range n.nodes {
			match = n.nodes[j].seg.Match(path[:ii], parms)
			if match {
				// ctx.AddPathParamsWithParams(parms)
				n = n.nodes[j]
				break
			}
		}

		// Not found
		if inode == n {
			return nil
		}

		// Capture all the rest of path
		if n.seg.CatchAll() {
			// ctx.AddPathParamsWithParams(parms)
			ctx.AddPathParam("*", path)
			// ctx.PathParams["*"] = path
			ctx.RoutePath = originalpath[:ior-1]
			return n
		}

		// Time to send an answer
		if i == -1 {
			ctx.RoutePath = originalpath
			// ctx.AddPathParamsWithParams(parms)
			return n
		}
		path = path[i+1:]
		ior += (i + 1)
	}
}

func (n node) GetAll(path string, ctx *Context) []*node {
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	var routes []*node = nil

	div := strings.IndexRune(path, '/')

	usedPath := path
	var retaindedPath string
	if div >= 0 {
		usedPath = path[:div]
		retaindedPath = path[div+1:]
	}

	divless1 := div == -1

	var match bool
	var parms []PathParam

	for i := range n.nodes {
		match = n.nodes[i].seg.Match(usedPath, &parms)
		if match || n.nodes[i].seg.CatchAll() {
			if divless1 {
				if len(n.nodes[i].methods) > 0 {
					routes = append(routes, n.nodes[i])
				}
			} else {
				routes = append(routes, n.nodes[i].GetAll(retaindedPath, ctx)...)
			}

			if !divless1 && n.nodes[i].seg.CatchAll() && len(n.nodes[i].methods) > 0 {
				routes = append(routes, n.nodes[i])
			}
		}
	}

	return routes
}

func (t trie) Print() string {
	builder := strings.Builder{}

	builder.WriteString("trie\n")

	i := 0
	for _, nn := range t.nodes {
		if i < len(t.nodes)-1 {
			nn.Print("", false, &builder)
		} else {
			nn.Print("", true, &builder)
		}
		i += 1
	}

	return builder.String()
}

func (n node) Print(prefix string, last bool, builder *strings.Builder) {
	builder.WriteString(prefix)
	if !last {
		builder.WriteString("├ /")
		prefix += "|  "
	} else {
		builder.WriteString("└ /")
		prefix += "   "
	}

	if n.seg != nil {
		builder.WriteString(n.seg.String())
	}

	if len(n.methods) > 0 {
		builder.WriteString(" (handler)")
	}
	builder.WriteString("\n")

	i := 0
	for _, nn := range n.nodes {
		if i < len(n.nodes)-1 {
			nn.Print(prefix, false, builder)
		} else {
			nn.Print(prefix, true, builder)
		}
		i += 1
	}
}

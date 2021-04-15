package smux

import (
	"net/http"
	"net/url"
	"testing"
)

func TestAddNoMathod(t *testing.T) {
	tr := trie{}
	r, _ := NewRoute().Handler(http.DefaultServeMux).Path("/users").Build()

	err := tr.Add(r)
	if err == nil {
		t.Fatal("Must have error")
	}

	t.Logf("Error: %v", err)
}

func TestTrieAdd(t *testing.T) {
	tr := trie{}

	paths := []string{"/a/sdf/dfdf", "/a/sdf/dfdf/ddd", "/a/sdf", "/a/sdf/df", "/b/asdb"}

	for _, path := range paths {
		r, _ := NewRoute().Handler(http.DefaultServeMux).Methods("get").Path(path).Build()

		if err := tr.Add(r); err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		t.Log(tr)
	}

	// Test if paths are created correctly

	// The basis must have size 2
	if len(tr.nodes) != 2 {
		t.Fatal("The size must be 2")
	}

	// The first node must have size 1
	if len(tr.nodes[0].nodes) != 1 {
		t.Fatal("The first node size must be 1")
	}

	if len(tr.nodes[1].nodes) != 1 {
		t.Fatal("The second node size must be 1")
	}

	if len(tr.nodes[0].methods) > 0 {
		t.Fatal("The sdf node must have value")
	}

	if len(tr.nodes[0].nodes[0].nodes) != 2 {
		t.Fatal("The sdf node size must be 2")
	}

	// Adding again must fail

	for _, path := range paths {
		r, _ := NewRoute().Handler(http.DefaultServeMux).Methods("get").Path(path).Build()

		err := tr.Add(r)
		if err == nil {
			t.Fatalf("Unexpected non error: %v", err)
		}

		t.Logf("Error found: %v", err)
	}
}

func TestTrieAddEmpty(t *testing.T) {
	tr := trie{}

	paths := []string{"/", "/a/"}

	for _, path := range paths {
		r, _ := NewRoute().Handler(http.DefaultServeMux).Methods("get").Path(path).Build()

		if err := tr.Add(r); err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		t.Log(tr)
	}
}

func TestTrieGet(t *testing.T) {
	ctx := &Context{}
	ctx.Reset()

	tr := trie{}

	paths := []string{"/a/sdf/dfdf", "/a/sdf/dfdf/ddd", "/a/sdf", "/a/sdf/df", "/b/asdb"}

	for _, path := range paths {

		r, _ := NewRoute().
			Handler(http.DefaultServeMux).
			Methods("GET", "POST").
			Path(path).
			Build()

		if err := tr.Add(r); err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		t.Log(tr)
	}

	if tr.Get("/a/sdf", ctx) == nil {
		t.Fatal("GET /a/sdf")
	}

	if tr.Get("/a/sdf/dfdf", ctx) == nil {
		t.Fatal("/a/sdf/dfdf")
	}

	if tr.Get("/a/sdf/dfdf", ctx) == nil {
		t.Fatal("/a/sdf/dfdf")
	}

	if tr.Get("/a/sdf/dfdf", ctx).Methods()["PUT"] != nil {
		t.Fatal("/a/sdf/dfdf")
	}

	if tr.Get("/outracoisa", ctx) != nil {
		t.Fatal("/outracoisa")
	}

	if tr.Get("", ctx) != nil {
		t.Fatal("Empty")
	}
}

func TestTrieCatchAllGetAll(t *testing.T) {
	tr := trie{}

	ctx := &Context{}
	ctx.Reset()

	paths := []string{"/a/b/{*}", "/a/b", "/c/{*}"}

	for _, path := range paths {
		r, _ := NewRoute().
			Handler(http.DefaultServeMux).
			Methods("GET", "POST").
			Path(path).
			Build()

		if err := tr.Add(r); err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	}

	t.Log("/a", tr.GetAll("/a", ctx), len(tr.GetAll("/a", ctx)))
	t.Log("/a/b", tr.GetAll("/a/b", ctx))
	t.Log("/a/b/c", tr.GetAll("/a/b/c", ctx))
	t.Log("/a/b/cccc", tr.GetAll("/a/b/cccc", ctx))
	t.Log("/c/b/cccc", tr.GetAll("/c/b/cccc", ctx))
}

func TestPrintTrie(t *testing.T) {
	tr := trie{}

	paths := []string{"/a/sdf/dfdf", "/a/sdf/dfdf/ddd", "/a/sdf", "/a/sdf/df", "/a/sdf/dfdddd", "/a/test/v1/{*}", "/a/test/v2/{*}", "/b/asdb", "/static/{*}", "/users/{uid}", "/slash/slc", "/slash/slc/"}

	for _, path := range paths {
		r, _ := NewRoute().
			Handler(http.DefaultServeMux).
			Methods("GET", "POST").
			Path(path).
			Build()

		if err := tr.Add(r); err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	}

	t.Logf("\n%s", tr.Print())
}

func TestTrieUsersGETandPOST(t *testing.T) {
	tr := trie{}

	r, _ := NewRoute().
		Handler(http.DefaultServeMux).
		Methods("GET").
		Path("/users").
		Build()

	if err := tr.Add(r); err != nil {
		t.Fatal("Erro adding GET")
	}

	r, _ = NewRoute().
		Handler(http.DefaultServeMux).
		Methods("POST").
		Path("/users").
		Build()

	if err := tr.Add(r); err != nil {
		t.Fatal("Erro adding POST")
	}
}

func TestCreateRegex(t *testing.T) {
	r, err := NewSegmentRegex("wait-{t:int}")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	t.Log(r)
	t.Log(r.(segmentregex).r.String())
	t.Log(r.(segmentregex).fields)
}

func TestCreateRegex2(t *testing.T) {
	r, err := NewSegmentRegex("from-{init:uint}-to-{end:uint}")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	t.Log(r)
	t.Log(r.(segmentregex).r.String())
	t.Log(r.(segmentregex).fields)
}

func TestCreateRegex3(t *testing.T) {
	r, err := NewSegmentRegex("App({id:uuidv4})")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	t.Log(r)
	t.Log(r.(segmentregex).r.String())
	t.Log(r.(segmentregex).fields)
}

func TestRegexesEqual(t *testing.T) {
	r1, _ := NewSegmentRegex(`wait-{t:uint}`)
	r2, _ := NewSegmentRegex(`wait-{tt:uint}`)

	if r1.Comparable() != r2.Comparable() {
		t.Fatal("Must be equal")
	}

	t.Log(r1, r2)
}

func TestMatchTime(t *testing.T) {
	tr := trie{}

	ctx := &Context{}
	ctx.Reset()

	route, _ := NewRoute().Handler(http.DefaultServeMux).Methods("get").Path("/time/wait-{t:uint}").Build()

	if err := tr.Add(route); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	result := tr.Get("/time/wait-200", ctx)
	if result == nil {
		t.Fatal("Must find a path")
	}

	t.Log(result)
	t.Log(ctx)
}

func TestMatchFromTo(t *testing.T) {
	ctx := &Context{}
	ctx.Reset()
	tr := trie{}
	route, _ := NewRoute().Handler(http.DefaultServeMux).Methods("get").Path("/distance/from-{from}-to-{to}").Build()

	if err := tr.Add(route); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	result := tr.Get("/distance/from-canoas-to-poa", ctx)
	if result == nil {
		t.Fatal("Must find a path")
	}

	t.Log(result)
	t.Log(ctx)
}

func TestCatchAllContextPath(t *testing.T) {
	ctx := &Context{}
	ctx.Reset()
	tr := trie{}

	route, _ := NewRoute().Handler(http.DefaultServeMux).Methods("get").Path("/static/{*}").Build()

	if err := tr.Add(route); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	result := tr.Get("/static/index.html", ctx)
	if result == nil {
		t.Fatal("Must find a path")
	}
	t.Log(result)
	t.Log(ctx)
	if ctx.PathParam("*") != "index.html" {
		t.Fatalf("Wrong path params: %v", ctx.PathParam("*"))
	}

	ctx.Reset()
	result = tr.Get("/static/assets/mylib.js", ctx)
	if result == nil {
		t.Fatal("Must find a path")
	}
	t.Log(result)
	t.Log(ctx)
	if ctx.PathParam("*") != "assets/mylib.js" {
		t.Fatalf("Wrong path params: %v", ctx.PathParam("*"))
	}

	ctx.Reset()
	result = tr.Get("/static/", ctx)
	if result == nil {
		t.Fatal("Must find a path")
	}
	t.Log(result)
	t.Log(ctx)
	if v := ctx.PathParam("*"); v != "" {
		t.Fatalf("Wrong path params: %v", ctx.PathParam("*"))
	}
}

func TestInvalidPath(t *testing.T) {
	tr := trie{}

	rt := &Route{
		path: "http://example.com/path",
	}

	err := tr.Add(rt)
	if err == nil {
		t.Fatal("Must have some error")
	}

	t.Logf("Err: %v", err)
}

func TestInvalidCatchAll(t *testing.T) {
	tr := trie{}

	_, err := NewRoute().Path("/path/{*}/error").Methods(http.MethodGet).Build()
	if err == nil {
		t.Fatal("Must have some error")
	}

	rt := &Route{
		path: "/path/{*}/error",
	}

	err = tr.Add(rt)
	if err == nil {
		t.Fatal("Must have some error")
	}

	t.Logf("Err: %v", err)
}

func TestCatchAllHidingPath(t *testing.T) {
	tr := trie{}

	rt, err := NewRoute().Path("/path/{*}").Handler(http.DefaultServeMux).Methods(http.MethodGet).Build()
	if err != nil {
		t.Fatalf("Must not have some error %v", err)
	}
	tr.Add(rt)

	rt, err = NewRoute().Path("/path/err").Handler(http.DefaultServeMux).Methods(http.MethodGet).Build()
	if err != nil {
		t.Fatalf("Must not have some error %v", err)
	}

	err = tr.Add(rt)
	if err == nil {
		t.Fatal("Must have some error")
	}

	t.Logf("Err: %v", err)
}

func TestCatchAllWillHidePath(t *testing.T) {
	tr := trie{}

	rt, _ := NewRoute().Path("/path/err").Handler(http.DefaultServeMux).Methods(http.MethodGet).Build()
	tr.Add(rt)

	rt, _ = NewRoute().Path("/path/{*}").Handler(http.DefaultServeMux).Methods(http.MethodGet).Build()
	err := tr.Add(rt)
	if err == nil {
		t.Fatal("Must have some error")
	}

	t.Logf("Err: %v", err)
}

func TestAddNilToTrie(t *testing.T) {
	tr := trie{}
	tr.Add(nil)
}

func TestUrlEscape(t *testing.T) {
	t.Log(url.PathEscape("<>"))
}

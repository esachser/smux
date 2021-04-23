package smux

import (
	"net/http"
	"testing"
)

func TestVerifyHostname(t *testing.T) {
	testCases := []struct {
		desc     string
		hostname string
	}{
		{
			desc:     "Hostname ok",
			hostname: "www.example.com",
		},
		{
			desc:     "Hostname ok 2",
			hostname: "*.example.com",
		},
		{
			desc:     "Hostname ok 4",
			hostname: "*.*.example.com",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			if !verifyHostname(tC.hostname) {
				t.Fatalf("hostname %v should be accepted", tC.hostname)
			}
		})
	}
}

func TestVerifyHostnameNotOk(t *testing.T) {
	testCases := []struct {
		desc     string
		hostname string
	}{
		{
			desc:     "Hostname ok",
			hostname: "www.{}.com",
		},
		{
			desc:     "Hostname ok 2",
			hostname: "www.example*.com",
		},
		{
			desc:     "Hostname ok",
			hostname: "www.**.com",
		},
		{
			desc:     "Hostname ok",
			hostname: "www.example.*",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			if verifyHostname(tC.hostname) {
				t.Fatalf("hostname %v should be not be accepted", tC.hostname)
			}
		})
	}
}

func TestEntriesNoIntersection(t *testing.T) {
	testCases := []struct {
		desc string
		h1   string
		h2   string
	}{
		{
			desc: "t1",
			h1:   "*.example",
			h2:   "*.local",
		},
		{
			desc: "t2",
			h1:   "*.example.com",
			h2:   "*.com",
		},
		{
			desc: "t3",
			h1:   "*.example.com",
			h2:   "*.*.example.com",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			he1 := hostentry{host: tC.h1}
			he2 := hostentry{host: tC.h2}
			if he1.Intersects(he2) {
				t.Fatal("Must not intersect")
			}
		})
	}
}

func TestEntriesIntersection(t *testing.T) {
	testCases := []struct {
		desc string
		h1   string
		h2   string
	}{
		{
			desc: "t1",
			h1:   "*.example",
			h2:   "*.*",
		},
		{
			desc: "t2",
			h1:   "*.example.com",
			h2:   "*.*.com",
		},
		{
			desc: "t2",
			h1:   "*.*.com",
			h2:   "*.example.com",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			he1 := hostentry{host: tC.h1}
			he2 := hostentry{host: tC.h2}
			if !he1.Intersects(he2) {
				t.Fatal("Must intersect")
			}
		})
	}
}

func TestMatch1(t *testing.T) {
	ctx := &Context{}
	h1 := hostentry{host: "*.example.com"}

	h1.Compile()

	for _, h := range []string{"www.example.com", "api.example.com"} {
		t.Run(h, func(t *testing.T) {
			if !h1.Match(h, ctx) {
				t.Fatal("Must match")
			}

			if len(ctx.HostParams) != 1 {
				t.Fatal("Must have 1 host param")
			}

			t.Log(ctx.HostParams)
		})
	}
}

func TestMatch2(t *testing.T) {
	ctx := &Context{}
	h1 := hostentry{host: "*.*.example.com"}

	h1.Compile()

	for _, h := range []string{"www.a.example.com", "opendata.api.example.com"} {
		t.Run(h, func(t *testing.T) {
			if !h1.Match(h, ctx) {
				t.Fatal("Must match")
			}

			if len(ctx.HostParams) != 2 {
				t.Fatal("Must have 2 host params")
			}

			t.Log(ctx.HostParams)
		})
	}
}

func TestAddHosts(t *testing.T) {
	hr := NewHostRouter()
	testCases := []struct {
		desc string
	}{
		{
			desc: "*.local",
		},
		{
			desc: "*.example.com",
		},
		{
			desc: "*.*.test",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			if err := hr.AddHostname(tC.desc); err != nil {
				t.Fatalf("Must not have error: %v", err)
			}
		})
	}
}

func TestInvalidHost(t *testing.T) {
	hr := NewHostRouter()

	err := hr.AddHostname("*.example.*.com")

	if err == nil {
		t.Fatal("Expected error")
	}

	t.Logf("Error returned: %v", err)
}

func TestAddHostWithError(t *testing.T) {
	hr := NewHostRouter()

	hr.AddHostname("*.example.test.com")

	err := hr.AddHostname("*.*.test.com")

	if err == nil {
		t.Fatal("Expected error")
	}

	t.Logf("Error returned: %v", err)
}

func TestAddHostWithError2(t *testing.T) {
	hr := NewHostRouter()

	hr.AddHostname("*.*.test.com")

	err := hr.AddHostname("*.example.test.com")

	if err == nil {
		t.Fatal("Expected error")
	}

	t.Logf("Error returned: %v", err)
}

func TestAddHostAndRouteWithGet(t *testing.T) {
	ctx := &Context{}
	ctx.Reset()

	hr := NewHostRouter()

	if err := hr.AddHostname("www.example.com"); err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	if err := hr.AddHostname("api.example.com"); err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	if err := hr.AddHostname("*.local.com"); err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	r1, err := NewRoute().Handler(http.DefaultServeMux).Host("www.example.com").Path("/").Methods(http.MethodGet).Build()
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}
	if err := hr.AddRoute(r1); err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	r2, err := NewRoute().Handler(http.DefaultServeMux).Host("www.example.com").Path("/static/{*}").Methods(http.MethodGet).Build()
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}
	if err := hr.AddRoute(r2); err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	r3, err := NewRoute().Handler(http.DefaultServeMux).Host("").Path("/").Methods(http.MethodGet).Build()
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}
	if err := hr.AddRoute(r3); err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	r4, err := NewRoute().Handler(http.DefaultServeMux).Host("*.local.com").Path("/local").Methods(http.MethodGet).Build()
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}
	if err := hr.AddRoute(r4); err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	r5, err := NewRoute().Handler(http.DefaultServeMux).Host("api.example.com").Path("/users").Methods(http.MethodGet).Build()
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}
	if err := hr.AddRoute(r5); err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	// Base route of www.example.com
	n1 := hr.Get("www.example.com", "/", ctx)
	if n1 == nil {
		t.Fatal("Must find")
	}
	if n1.Methods()["GET"] == nil {
		t.Fatal("Must find method")
	}
	if n1.Methods()["POST"] != nil {
		t.Fatal("Must not find method")
	}
	if n1.Methods()["GET"] != r1 {
		t.Fatal("Wrong route")
	}

	// Some static asset of www.example.com
	n2 := hr.Get("www.example.com", "/static/1.json", ctx)
	if n2 == nil {
		t.Fatal("Must find")
	}
	if n2.Methods()["GET"] == nil {
		t.Fatal("Must find method")
	}
	if n2.Methods()["POST"] != nil {
		t.Fatal("Must not find method")
	}
	if n2.Methods()["GET"] != r2 {
		t.Fatal("Wrong route")
	}

	// Some static asset of www.example.com
	n3 := hr.Get("www.example.com", "/static/img/1.png", ctx)
	if n3 == nil {
		t.Fatal("Must find")
	}
	if n3.Methods()["GET"] == nil {
		t.Fatal("Must find method")
	}
	if n3.Methods()["POST"] != nil {
		t.Fatal("Must not find method")
	}
	if n3.Methods()["GET"] != r2 {
		t.Fatal("Wrong route")
	}

	// Base route of "All hosts"
	n4 := hr.Get("www.test.com", "/", ctx)
	if n4 == nil {
		t.Fatal("Must find")
	}
	if n4.Methods()["GET"] == nil {
		t.Fatal("Must find method")
	}
	if n4.Methods()["POST"] != nil {
		t.Fatal("Must not find method")
	}
	if n4.Methods()["GET"] != r3 {
		t.Fatal("Wrong route")
	}

	// api.example.com has no base route
	// n5 := hr.Get("api.example.com", "/", ctx)
	// if n5 != nil {
	// 	t.Fatal("Must not find")
	// }

	// api.example.com has /users route
	n6 := hr.Get("api.example.com", "/users", ctx)
	if n6 == nil {
		t.Fatal("Must find")
	}
	if n6.Methods()["GET"] == nil {
		t.Fatal("Must find method")
	}
	if n6.Methods()["POST"] != nil {
		t.Fatal("Must not find method")
	}
	if n6.Methods()["GET"] != r5 {
		t.Fatal("Wrong route")
	}

	// api.example.com has no local
	n7 := hr.Get("api.example.com", "/local", ctx)
	if n7 != nil {
		t.Fatal("Must not find")
	}

	// *.local.com has /local and matches host api.local.com
	n8 := hr.Get("api.local.com", "/local", ctx)
	if n8 == nil {
		t.Fatal("Must find")
	}
	if n8.Methods()["GET"] == nil {
		t.Fatal("Must find method")
	}
	if n8.Methods()["POST"] != nil {
		t.Fatal("Must not find method")
	}
	if n8.Methods()["GET"] != r4 {
		t.Fatal("Wrong route")
	}

	// *.local.com has /local and matches host www.local.com
	n9 := hr.Get("www.local.com", "/local", ctx)
	if n9 == nil {
		t.Fatal("Must find")
	}
	if n9.Methods()["GET"] == nil {
		t.Fatal("Must find method")
	}
	if n9.Methods()["POST"] != nil {
		t.Fatal("Must not find method")
	}
	if n9.Methods()["GET"] != r4 {
		t.Fatal("Wrong route")
	}
}

func TestNoValidPath(t *testing.T) {
	hr := NewHostRouter()
	r1 := &Route{
		path: "www.example.com/",
	}

	err := hr.AddRoute(r1)
	if err == nil {
		t.Fatal("Must have some error")
	}

	t.Logf("Error: %v", err)
}

func TestNoMethods(t *testing.T) {
	hr := NewHostRouter()
	r1, _ := NewRoute().Handler(http.DefaultServeMux).Path("/foobar").Build()

	err := hr.AddRoute(r1)
	if err == nil {
		t.Fatal("Must have some error")
	}

	t.Logf("Error: %v", err)
}

func TestHostNotFound(t *testing.T) {
	hr := NewHostRouter()
	r1, _ := NewRoute().Handler(http.DefaultServeMux).Host("api.example.com").Path("/foobar").Methods(http.MethodGet).Build()

	err := hr.AddRoute(r1)
	if err == nil {
		t.Fatal("Must have some error")
	}

	t.Logf("Error: %v", err)
}

func TestAddNilToHR(t *testing.T) {
	hr := NewHostRouter()
	hr.AddRoute(nil)
}

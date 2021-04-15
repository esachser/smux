package smux

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAddGetUser(t *testing.T) {
	router := NewRouter()

	r1, err := NewRoute().Path("/users/{userid}").Methods("GET", "HEAD").HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		ctx := GetSmuxContext(r.Context())
		io.WriteString(rw, ctx.PathParam("userid"))
	}).Build()

	if err != nil {
		t.Fatalf("Found error: %v", err)
	}

	router.AddRoute(r1)

	if err := router.Compile(); err != nil {
		t.Fatalf("Error compiling: %v", err)
	}

	req := httptest.NewRequest("GET", "/users/1234", nil)
	rw := httptest.NewRecorder()

	router.ServeHTTP(rw, req)
	resp := rw.Result()
	if resp.StatusCode != 200 {
		t.Fatal("Must return 200")
	}
	defer resp.Body.Close()

	buf, _ := io.ReadAll(resp.Body)
	if string(buf) != "1234" {
		t.Fatal("Must receive userid 1234")
	}

	req = httptest.NewRequest("POST", "/users/1234", nil)
	rw = httptest.NewRecorder()

	router.ServeHTTP(rw, req)
	if rw.Code != http.StatusMethodNotAllowed {
		t.Fatal("Must be method not allowed")
	}
	if rw.Result().Header.Get("Allow") != "GET,HEAD" && rw.HeaderMap.Get("Allow") != "HEAD,GET" {
		t.Fatal("Expect GET,HEAD allowed")
	}
	if len(strings.Split(rw.Result().Header.Get("Allow"), ",")) != 2 {
		t.Fatal("Expected only 2 headers")
	}

	req = httptest.NewRequest("GET", "/users/", nil)
	rw = httptest.NewRecorder()

	router.ServeHTTP(rw, req)
	if rw.Code != http.StatusNotFound {
		t.Fatalf("Must be not found but was %v - %v", rw.Code, http.StatusText(rw.Code))
	}
}

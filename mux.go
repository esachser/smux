package smux

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
)

type RouteError struct {
	Type  string
	Msg   string
	Code  int
	Allow []string
}

func (r RouteError) Error() string {
	if r.Msg != "" {
		return r.Msg
	}
	return http.StatusText(r.Code)
}

type Router struct {
	routes          []*Route
	hosts           []string
	pool            *sync.Pool
	NotFoundHandler http.Handler
	hostRouter      *HostRouter
}

func NewRouter() *Router {
	r := &Router{pool: &sync.Pool{}, routes: []*Route{}}
	r.pool.New = func() interface{} {
		return &Context{}
	}
	return r
}

func (router *Router) AddRoute(route *Route) {
	router.routes = append(router.routes, route)
}

func (router *Router) SetRoutes(routes []*Route) {
	router.routes = routes
}

func (router Router) Routes() []*Route {
	return router.routes
}

func (router *Router) SetHostnames(hostnames []string) {
	router.hosts = hostnames
}

func (router *Router) Compile() error {
	router.hostRouter = NewHostRouter()
	for _, hn := range router.hosts {
		err := router.hostRouter.AddHostname(hn)
		if err != nil {
			return err
		}
	}
	for _, r := range router.routes {
		err := router.hostRouter.AddRoute(r)
		if err != nil {
			return err
		}
	}
	return nil
}

func (router *Router) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	ctx := router.pool.Get().(*Context)
	defer router.pool.Put(ctx)
	ctx.Reset()

	// Clean up the path following setted configuration

	if router.hostRouter == nil {
		if router.NotFoundHandler != nil {
			router.NotFoundHandler.ServeHTTP(rw, r)
		} else {
			http.NotFoundHandler().ServeHTTP(rw, r)
		}
		return
	}
	hostname := r.Host
	if strings.Contains(hostname, ":") {
		hostname, _, _ = net.SplitHostPort(r.Host)
	}

	routePath := ctx.RoutePath
	if routePath == "" {
		if r.URL.RawPath != "" {
			routePath = r.URL.RawPath
		} else {
			routePath = r.URL.Path
		}
	}

	n := router.hostRouter.Get(hostname, routePath, ctx)
	if n != nil {
		methodRouters := n.Methods()
		rt := methodRouters[r.Method]
		if rt == nil {
			// Method not allowed
			ms := make([]string, len(methodRouters))
			i := 0
			for m := range methodRouters {
				ms[i] = m
				i += 1
			}
			rw.Header().Set("Allow", strings.Join(ms, ","))
			rw.WriteHeader(http.StatusMethodNotAllowed)
			rw.Write([]byte(http.StatusText(http.StatusMethodNotAllowed)))
		} else {
			ctx.Route = rt
			ctx.handler = rt.handler
			ctx.RoutePath = routePath

			r = r.WithContext((*directContext)(ctx))
			ctx.handler.ServeHTTP(rw, r)
		}
	} else if router.NotFoundHandler != nil {
		router.NotFoundHandler.ServeHTTP(rw, r)
	} else {
		http.NotFoundHandler().ServeHTTP(rw, r)
	}
}

type Route struct {
	name    string
	host    string
	path    string
	methods map[string]struct{}
	handler http.Handler
}

func (r Route) Name() string {
	return r.name
}

func (r Route) Host() string {
	return r.host
}

func (r Route) Path() string {
	return r.path
}

func (r Route) Methods() []string {
	ms := make([]string, len(r.methods))
	i := 0
	for m := range r.methods {
		ms[i] = m
		i += 1
	}
	return ms
}

type RouteBuilder struct {
	name    string
	host    string
	path    string
	methods map[string]struct{}
	handler http.Handler
	err     error
}

func NewRoute() *RouteBuilder {
	return &RouteBuilder{methods: make(map[string]struct{})}
}

func (route *RouteBuilder) Name(n string) *RouteBuilder {
	if route.err != nil {
		return route
	}
	route.name = n
	return route
}

func (route RouteBuilder) mkname() string {
	if route.name != "" {
		return route.name
	}

	methods := make([]string, len(route.methods))
	i := 0
	for m := range route.methods {
		methods[i] = m
		i += 1
	}

	return fmt.Sprintf("[%s] %s%s", strings.Join(methods, " "), route.host, route.path)
}

func (route *RouteBuilder) Host(h string) *RouteBuilder {
	if route.err != nil {
		return route
	}
	if !verifyHost(h) {
		route.err = fmt.Errorf("invalid host")
		return route
	}
	route.host = h
	return route
}

func (route *RouteBuilder) Path(p string) *RouteBuilder {
	if route.err != nil {
		return route
	}
	if !verifyPath(p) {
		route.err = fmt.Errorf("invalid path")
		return route
	}
	route.path = p
	return route
}

func (route *RouteBuilder) Methods(ms ...string) *RouteBuilder {
	if route.err != nil {
		return route
	}

	var t struct{}
	route.methods = make(map[string]struct{})

	for i := range ms {
		m := strings.ToUpper(ms[i])
		route.methods[m] = t
	}

	return route
}

func (route *RouteBuilder) Handler(handler http.Handler) *RouteBuilder {
	if route.err != nil {
		return route
	}
	if handler == nil {
		route.err = fmt.Errorf("Handler must not be nil")
		return route
	}
	route.handler = handler
	return route
}

func (route *RouteBuilder) HandlerFunc(handler func(http.ResponseWriter, *http.Request)) *RouteBuilder {
	if route.err != nil {
		return route
	}
	if handler == nil {
		route.err = fmt.Errorf("Handler must not be nil")
		return route
	}
	route.handler = http.HandlerFunc(handler)
	return route
}

func (route RouteBuilder) GetError() error {
	return route.err
}

func (r RouteBuilder) Build() (*Route, error) {
	if r.err != nil {
		return nil, r.err
	}

	if r.handler == nil {
		return nil, fmt.Errorf("Handler must not be nil")
	}

	return &Route{
		name:    r.mkname(),
		host:    r.host,
		path:    r.path,
		methods: r.methods,
		handler: r.handler,
	}, nil
}

type MatchResult interface {
	Methods() map[string]*Route
}

type PathRouter interface {
	Add(*Route) error
	Get(string, *Context) MatchResult
}

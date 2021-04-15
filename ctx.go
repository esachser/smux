package smux

import (
	"context"
	"net/http"
	"time"
)

type contextParams struct {
	name string
}

func (c contextParams) String() string {
	return "smux ctx key " + c.name
}

var (
	ParamContext = &contextParams{"context"}
)

type PathParam struct {
	Key   string
	Value string
}

type Context struct {
	Route      *Route
	pathParams []PathParam
	HostParams []string
	RoutePath  string
	handler    http.Handler
	parentCtx  context.Context
}

func GetSmuxContext(ctx context.Context) *Context {
	cont, ok := ctx.Value(ParamContext).(*Context)
	if ok {
		return cont
	} else {
		ct := &Context{}
		ct.Reset()
		return ct
	}
}

func (ctx *Context) Reset() {
	ctx.pathParams = nil
	ctx.HostParams = nil
	ctx.RoutePath = ""
	ctx.Route = nil
	ctx.handler = nil
}

func (ctx Context) PathParam(p string) string {
	for i := range ctx.pathParams {
		if ctx.pathParams[i].Key == p {
			return string(ctx.pathParams[i].Value)
		}
	}
	return ""
}

func (ctx *Context) AddPathParam(key string, parm string) {
	ctx.pathParams = append(ctx.pathParams, PathParam{key, parm})
}

func (ctx *Context) SetPathParam(key, parm string) {
	for i := range ctx.pathParams {
		if ctx.pathParams[i].Key == key {
			ctx.pathParams[i].Value = string(parm)
		}
	}
	ctx.pathParams = append(ctx.pathParams, PathParam{key, string(parm)})
}

func (ctx *Context) AddPathParams(keys []string, params []string) {
	if len(keys) != len(params) {
		return
	}
	parms := make([]PathParam, len(keys))
	for i := range keys {
		parms[i] = PathParam{keys[i], params[i]}
	}
	ctx.pathParams = append(ctx.pathParams, parms...)
}

func (ctx *Context) AddPathParamsWithParams(parms []PathParam) {
	ctx.pathParams = append(ctx.pathParams, parms...)
}

// Copy from chi
type directContext Context

var _ context.Context = (*directContext)(nil)

func (d *directContext) Deadline() (deadline time.Time, ok bool) {
	return d.parentCtx.Deadline()
}

func (d *directContext) Done() <-chan struct{} {
	return d.parentCtx.Done()
}

func (d *directContext) Err() error {
	return d.parentCtx.Err()
}

func (d *directContext) Value(key interface{}) interface{} {
	if key == ParamContext {
		return (*Context)(d)
	}
	return d.parentCtx.Value(key)
}

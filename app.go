// MIT License
//
// Copyright (c) 2017 José Santos <henrique_1609@me.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package cloudy

import (
	"github.com/CloudyKit/cloudy/event"
	"github.com/CloudyKit/cloudy/link"
	"github.com/CloudyKit/cloudy/registry"
	"github.com/CloudyKit/router"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync"
)

var KernelType = reflect.TypeOf((*Kernel)(nil))

func GetKernel(c Registry) *Kernel {
	return c.LoadType(KernelType).(*Kernel)
}

var DefaultKernel = NewKernel()

func NewKernel() *Kernel {
	kernel := &Kernel{Registry: registry.New(), Router: router.New(), URLGen: make(MapURLGen), emitter: event.NewDispatcher()}

	// provide service URLGen as URLer
	kernel.Registry.WithTypeAndValue(link.URLGenType, kernel.URLGen)
	// provide the Router
	kernel.Registry.WithValues(kernel.Router)
	// provide the app
	kernel.Registry.WithTypeAndValue(KernelType, kernel)
	kernel.Registry.WithTypeAndValue(event.EmitterType, kernel.emitter)

	return kernel
}

func (kernel *Kernel) Container() Registry {
	return kernel.Registry
}

type MiddlewareBundle struct {
	handlers []Handler
}

// ClearMiddlewares will clear the registered middlewares for next route added
func (filterHandlers *MiddlewareBundle) ClearMiddlewares() {
	filterHandlers.handlers = nil
}

// AddMiddleware adds handlers to the request chain
func (filterHandlers *MiddlewareBundle) AddMiddleware(filters ...Handler) {
	filterHandlers.handlers = append(filterHandlers.handlers, filters...)
}

func (filterHandlers *MiddlewareBundle) AddMiddlewareFunc(filters ...HandlerFunc) {
	newLen := len(filters) + len(filterHandlers.handlers)
	newFilters := make([]Handler, newLen)
	copy(newFilters, filterHandlers.handlers)

	for i, j := len(filterHandlers.handlers), 0; i < newLen; i, j = i+1, j+1 {
		newFilters[i] = filters[j]
	}

	filterHandlers.handlers = newFilters
}

func (filterHandlers *MiddlewareBundle) reSlice(filters ...Handler) []Handler {
	newFilter := make([]Handler, 0, len(filterHandlers.handlers)+len(filters))
	newFilter = append(newFilter, filterHandlers.handlers...)
	newFilter = append(newFilter, filters...)
	return newFilter
}

type emitter interface {
	Subscribe(eventName string, handler interface{}) *event.Dispatcher
	Dispatch(registry Registry, eventName string, event event.Payload) (canceled bool, err error)
}

// Kernel app holds your top level data for you service
//
//	Router, Dispatcher, Scope
type Kernel struct {
	emitter emitter

	Registry Registry       // Kernel Registry dependency injection context
	Router   *router.Router // Router
	Prefix   string         // Prefix prefix for path added in this app
	URLGen   MapURLGen
	MiddlewareBundle
}

// Component represents a service component, a component need to implement
// a bootstrap method which is responsible to set up the component with the app,
// ex: register a type Providers, or add middleware handler
type Component interface {
	Bootstrap(app *Kernel)
}

// Root returns the root app
func (kernel *Kernel) Root() *Kernel {
	return GetKernel(kernel.Registry)
}

// Snapshot causes a sub app to be created and inserted in the scope
// calling app.Root will return the created sub app
func (kernel *Kernel) Snapshot() *Kernel {
	newKernel := *kernel

	newKernel.Registry = kernel.Registry.Fork()
	newKernel.Registry.WithTypeAndValue(KernelType, newKernel)

	return &newKernel
}

// ComponentFunc func implementing Component interface
type ComponentFunc func(*Kernel)

func (component ComponentFunc) Bootstrap(a *Kernel) {
	component(a)
}

// AddComponents bootstraps a list of components, a sub scope will be created, and a copy of the
// original app is used, in such form that modifying the app.Prefix will not reflect outside this
// call.
func (kernel *Kernel) AddComponents(b ...Component) {
	newApp := kernel.Fork()
	prefix := newApp.Prefix
	for i := 0; i < len(b); i++ {

		bv := reflect.ValueOf(b[i])
		if bv.Kind() == reflect.Ptr {
			bv = bv.Elem()
			if bv.Kind() == reflect.Struct {
				newApp.Registry.InjectValue(bv)
			}
		}

		b[i].Bootstrap(newApp)
		newApp.Prefix = prefix
	}
}

// Dispose Close same as app.registry.Close() invoke this func before exiting the app to cleanup
func (kernel *Kernel) Dispose() {
	kernel.Registry.Dispose()
}

// AddHandlerFunc register a func handler, see: Handler
func (kernel *Kernel) AddHandlerFunc(method, path string, fn HandlerFunc, filters ...Handler) {
	kernel.AddHandler(method, path, fn, filters...)
}

// AddHandler register a handler, see: Handler
func (kernel *Kernel) AddHandler(method, path string, handler Handler, filters ...Handler) {
	kernel.AddHandlerName("", method, path, handler, filters...)
}

// AddHandlerName register a named handler, see: Handler
func (kernel *Kernel) AddHandlerName(name, method, path string, handler Handler, filters ...Handler) {
	kernel.AddHandlerContextName(kernel.Registry, name, method, path, handler, filters...)
}

// AddHandlerContextName accepts a context, a ParamKey identifier, http method|methods, pattern path, handler and handlers
// ex: one handler app.AddHandlerContextName(myContext,"mySectionIdentifier","GET", "/public",fileServer,checkAuth)
//
//	multiples handles app.AddHandlerContextName(myContext,"mySectionIdentifier","GET|POST|SEARCH", "/products",productHandler,checkAuth)
func (kernel *Kernel) AddHandlerContextName(registry Registry, name, method, path string, handler Handler, filters ...Handler) {

	filters = append(kernel.reSlice(filters...), handler)

	if registry == nil {
		registry = kernel.Registry
	}

	for _, method := range strings.Split(method, "|") {
		kernel.Router.AddRoute(method, kernel.Prefix+path, func(rw http.ResponseWriter, r *http.Request, v router.Parameter) {
			c := newRequestContext()
			defer requestRecover(c)
			_ = DispatchNext(c, name, rw, r, v, registry.Fork(), filters)
		})
	}
}

// requestRecover finalizes and cleanup request allocated scope variables
func requestRecover(c *Context) {

	variables := c.Registry
	// resets request context
	*c = Context{}
	contextPool.Put(c)

	// we call scope EndForce, this requires that all children scopes Ended in this call if not
	// panic is raised
	variables.MustDispose()
}

func (kernel *Kernel) host(host string) (servein string) {
	// if host is empty set host apphost
	if host == "" {
		host = "apphost"
	}
	// check if host is an env variable containing a host string
	servein = os.Getenv(host)
	// if host is not an env variable than is a host string
	if servein == "" {
		servein = host
	}
	return
}

// RunServer runs the server with the specified host
// Calling this func will emit a "app.run" event in the app
func (kernel *Kernel) RunServer(host string) error {
	//e := &RunServerEvent{Host: host}
	//kernel.Dispatch("hub.run", e)
	return http.ListenAndServe(host, kernel.Router)
}

// RunServerTLS runs the server in tls mode
// Calling this func will emit a "app.run.tls" event in the app
func (kernel *Kernel) RunServerTLS(host, certfile, keyfile string) error {
	//e := &RunServerEventTLS{Host: host, CertFile: certfile, KeyFile: keyfile}
	// kernel.Dispatch("hub.run.tls", e)
	return http.ListenAndServeTLS(kernel.host(host), certfile, keyfile, kernel.Router)
}

func (kernel *Kernel) Subscribe(eventName string, handler interface{}) {
	kernel.emitter.Subscribe(eventName, handler)
}

func (kernel *Kernel) Dispatch(eventName string, payload event.Payload) {
	_, _ = kernel.emitter.Dispatch(kernel.Registry, eventName, payload)
}

// Fork create child app
func (kernel *Kernel) Fork() *Kernel {
	newApp := *kernel
	newApp.handlers = kernel.reSlice()
	//newApp.registry = app.registry.Fork()
	return &newApp
}

func (kernel *Kernel) MustDispose() {
	kernel.Registry.MustDispose()
}

var contextPool = sync.Pool{
	New: func() interface{} {
		return new(Context)
	},
}

func newRequestContext() *Context {
	return contextPool.Get().(*Context)
}

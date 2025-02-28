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
	"bytes"
	"encoding/json"
	"github.com/CloudyKit/cloudy/link"
	"github.com/CloudyKit/router"
	"io"

	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
)

var ContextType = reflect.TypeOf((*Context)(nil))

// GetContext gets a Context from the registry context
func GetContext(cdi Registry) *Context {
	return cdi.LoadType(ContextType).(*Context)
}

// Context holds context information about the incoming request
type Context struct {
	Name     string        // The name associated with the route
	Registry Registry      // Dependency injection context
	Request  *http.Request // Request data passed by the router
	Gen      *link.URLGen

	handlers []Handler

	Response  http.ResponseWriter // Response Writer passed by the router
	Params    router.Parameter    // Route Registry passed by the router
	body      io.ReadCloser
	bodyBytes []byte
	bodyReady bool
}

func (c *Context) SendJSONStatusCode(statusCode int, v any) error {

	c.Response.Header().Set("Content-Type", "application/json")
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	c.Response.WriteHeader(statusCode)
	return json.NewEncoder(c.Response).Encode(v)
}

func (c *Context) SendJSON(v any) error {
	return c.SendJSONStatusCode(http.StatusOK, v)
}

func (c *Context) GoContext() context.Context {
	return c.Request.Context()
}

// Next will continue with the request flow
func (c *Context) Next() error {

	if len(c.handlers) == 0 {
		return errors.New("request.Context: no available handlers to advance")
	}

	// todo: with this behavior we can allow retry, a func can advance multiple times
	// handlers := c.handlers
	// c.handlers = c.handlers[1:]
	// handlers[0].Handle(c)
	// c.handlers = handlers

	handler := c.handlers[0]
	c.handlers = c.handlers[1:]
	handler.Handle(c)
	return nil
}

// WriteString writes the string txt into the the response
func (c *Context) WriteString(txt string) (int, error) {
	return c.Response.Write([]byte(txt))
}

// Printf prints a formatted text to response writer
func (c *Context) Printf(format string, v ...interface{}) (int, error) {
	return fmt.Fprintf(c.Response, format, v...)
}

// Redirect redirects the request to the specified urlStr and send a http StatusFound code
func (c *Context) Redirect(urlStr string) {
	c.RedirectStatus(urlStr, http.StatusFound)
}

// RedirectStatus redirects the request to the specified urlStr and send the the status code specified by httpStatus
func (c *Context) RedirectStatus(urlStr string, httpStatus int) {
	http.Redirect(c.Response, c.Request, urlStr, httpStatus)
}

// GetURLParameter returns a parameter from the url route, GetURLParameter is shortcut for Context.Params.Get method
func (c *Context) GetURLParameter(name string) string {
	return c.Params.Get(name)
}

// GetPostValue  returns a form value from the request, GetPostValue is shortcut for Context.Request.Form.Get method
func (c *Context) GetPostValue(name string) string {
	if c.Request.PostForm == nil {
		_ = c.Request.ParseForm()
	}
	return c.Request.PostForm.Get(name)
}

// GetGetValue  returns a form value from the request, GetPostValue is shortcut for Context.Request.Form.Get method
func (c *Context) GetGetValue(name string) string {
	if c.Request.Form == nil {
		_ = c.Request.ParseForm()
	}
	return c.Request.Form.Get(name)
}

// GetCookieValue returns a cookie value from the request
func (c *Context) GetCookieValue(name string) (value string) {
	if cookie, _ := c.Request.Cookie(name); cookie != nil {
		value, _ = url.QueryUnescape(cookie.Value)
	}
	return
}

type contextBodyReader struct {
	c       *Context
	buffer  bytes.Reader
	started bool
}

func (c *Context) GetBodyBytes() ([]byte, error) {
	var err error
	if !c.bodyReady {
		c.bodyBytes, err = io.ReadAll(c.body)
		if err != nil {
			return nil, err
		}
		c.bodyReady = true
	}
	return c.bodyBytes, nil
}

func (c *contextBodyReader) Read(p []byte) (int, error) {
	if !c.started {
		bodyBytes, err := c.c.GetBodyBytes()
		if err != nil {
			return 0, err
		}
		c.started = true
		c.buffer.Reset(bodyBytes)
	}
	return c.buffer.Read(p)
}

func (c *contextBodyReader) Close() error {
	return c.c.body.Close()
}

// GetBodyReader returns bytes bodyReader
func (c *Context) GetBodyReader() io.ReadCloser {
	return &contextBodyReader{
		c: c,
	}
}

func (c *Context) SendTextWithStatus(statusCode int, content string) error {
	c.Response.Header().Set("Content-Type", "text/plain")
	c.Response.WriteHeader(statusCode)
	_, err := c.Response.Write([]byte(content))
	return err
}

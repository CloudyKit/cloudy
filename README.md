# Cloudy

Cloudy is a lightweight, flexible web framework for Go that emphasizes simplicity and productivity. It provides a clean architecture for building web applications with features like routing, sessions, middleware, and more.

## Features

- Simple and intuitive API
- Built-in session management
- Flash message support
- Middleware pipeline
- Controller-based architecture
- Easy routing with method mapping

## Installation

```bash
go get github.com/CloudyKit/cloudy
```

## Quick Start

Below is a basic example of how to set up a Cloudy web application:

### Directory Structure

```
myapp/
├── app/
│   └── app.go
├── cmd/
│   └── server.go
└── controllers/
    └── home.go
```

### Server Entry Point (server.go)

```go:cmd/server.go
package main

import "github.com/CloudyKit/cloudy-example/app"

func main() {
	app.Kernel.RunServer(":8888")
}
```

### Application Setup (app.go)

```go:app/app.go
package app

import (
	"github.com/CloudyKit/cloudy"
	"github.com/CloudyKit/cloudy-example/controllers"
	"github.com/CloudyKit/cloudy/flash"
	"github.com/CloudyKit/cloudy/session"
)

var Kernel = cloudy.NewKernel()

func init() {
	// Register components
	Kernel.AddComponents(
		&session.Component{
			Manager:       session.DefaultManager,
			CookieOptions: session.DefaultCookieOptions,
		},
		&flash.Component{},
	)

	// Register controllers
	Kernel.AddControllers(
		&controllers.Home{},
	)
	
	// Add middleware
	Kernel.AddMiddlewareFunc(func(ctx *cloudy.Context) {
		ctx.Response.Header().Set("X-Cloudy", "CloudyKit")
	})
}
```

### Controller (home.go)

```go:controllers/home.go
package controllers

import (
	"github.com/CloudyKit/cloudy"
	"github.com/CloudyKit/cloudy/session"
)

type Home struct {
	Context     *cloudy.Context
	SessionData *session.Session
}

// Mx maps HTTP methods to controller actions
func (h *Home) Mx(mx *cloudy.Mapper) {   
	mx.BindAction("GET", "/", "Index")
}

// Index handles the root path
func (h *Home) Index() {
	counter, _ := h.SessionData.Get("counter").(int)
	_, _ = h.Context.PrintfWriteStringfWriteString("Counter: %d", counter)
	counter++
	defer h.SessionData.Set("counter", counter)
}
```

## Core Concepts

### Kernel

The Kernel is the central component that manages the application lifecycle. It handles routing, middleware execution, and component initialization.

### Controllers

Controllers handle incoming requests and produce responses. In Cloudy, controllers are Go structs with methods that correspond to HTTP endpoints.

### Middleware

Middleware functions can be added to the request processing pipeline to handle cross-cutting concerns like authentication, logging, etc.

### Components

Components provide additional functionality like sessions and flash messages. They can be easily added to the application\'s kernel.

### Mapper

The Mapper binds HTTP methods and paths to controller methods, making routing simple and intuitive.

## Running the Example

To run the example application:

```bash
go run cmd/server.go
```

Then visit http://localhost:8888 in your browser. You should see a counter that increments on each page refresh, demonstrating the session functionality.

## License

[MIT License]

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.'
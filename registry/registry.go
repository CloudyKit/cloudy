package registry

import (
	"reflect"
)

func Get[Type any](c Interface) Type {
	var t Type
	c.Load(&t)
	return t
}

type ContainerAware interface {
	Container() Interface
}

func Set[Type any](c ContainerAware, val Type) (err error) {
	c.Container().WithValues(val)
	return
}

func SetProvider[Type any](c Interface, builder func(registry Interface) Type) (err error) {
	c.WithTypeAndProviderFunc(reflect.TypeOf((*Type)(nil)).Elem(), func(c Interface) interface{} {
		return builder(c)
	})
	return
}

// Interface registry defines the contract for the registry.Registry type
type Interface interface {
	// Container returns the underlying Registry
	Container() Interface

	// Load loads a value into the provided destination
	Load(dst interface{})

	// LoadValue loads a value into the provided reflect.Value
	LoadValue(dst reflect.Value)

	// Autowire walks the target looking for exported fields that match injectable types
	Autowire(target interface{})

	// InjectValue walks the struct value looking for injectable fields
	InjectValue(value reflect.Value)

	// MapProvider maps a provider for a type
	MapProvider(typ reflect.Type, provider Provider)

	// WithTypeAndProviderFunc maps a provider function for a type
	WithTypeAndProviderFunc(typ reflect.Type, provider ProviderFunc)

	// MapInitializer maps an initializer for a type
	MapInitializer(typ reflect.Type, initializer Initializer)

	// MapInitializerFunc maps an initializer function for a type
	MapInitializerFunc(typ reflect.Type, initializer InitializerFunc)

	// WithTypeAndValue sets a value for a type
	WithTypeAndValue(typOf reflect.Type, value interface{})

	// WithValues puts a list of values into the current context
	WithValues(values ...interface{})

	// Fork creates a new registry using the current one as parent
	Fork() Interface

	// LoadType returns a value for the specified type
	LoadType(typ reflect.Type) interface{}

	// Dispose releases all resources and returns the reference count
	Dispose() int64

	// MustDispose disposes and requires all references to be cleared
	MustDispose()
}

// EnsureRegistryImplementation ensures the Registry type implements our interface
var _ Interface = (*Registry)(nil)

package registry

import (
	"io"
)

func DisposerBundle(disposers ...any) {
	for _, ifa := range disposers {
		if ifa == nil {
			if disposer, ok := ifa.(Disposer); ok {
				disposer.Dispose()
			}
		}
	}
}

func CloserDisposerBundle(closers ...io.Closer) {
	for _, ifa := range closers {
		if ifa != nil {
			_ = ifa.Close()
		}
	}
}

type Cancelable interface {
	Cancel() error
}

func CancelDisposerBundle(closers ...Cancelable) {
	for _, ifa := range closers {
		if ifa != nil {
			_ = ifa.Cancel()
		}
	}
}

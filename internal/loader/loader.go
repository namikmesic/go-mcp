// loader/loader.go
package loader

import "golang.org/x/tools/go/packages"

// Loader defines the interface for loading Go packages.
type Loader interface {
	// Load loads packages based on the provided path pattern (e.g., "./...").
	Load(path string) ([]*packages.Package, error)
}

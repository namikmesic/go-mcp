// loader/gopackages.go
package loader

import (
	"fmt"
	"log"

	"golang.org/x/tools/go/packages"
)

// GoPackagesLoader implements the Loader interface using golang.org/x/tools/go/packages.
type GoPackagesLoader struct {
	// Config allows customizing the packages.Load behavior.
	Config packages.Config
}

// NewGoPackagesLoader creates a loader with default configuration for analysis.
func NewGoPackagesLoader() *GoPackagesLoader {
	return &GoPackagesLoader{
		Config: packages.Config{
			Mode: packages.NeedName |
				packages.NeedFiles |
				packages.NeedCompiledGoFiles |
				packages.NeedImports |
				packages.NeedDeps |
				packages.NeedExportFile |
				packages.NeedTypes |
				packages.NeedSyntax |
				packages.NeedTypesInfo |
				packages.NeedTypesSizes |
				packages.NeedModule |
				packages.NeedEmbedFiles |
				packages.NeedEmbedPatterns,
			Tests: true, // Include test files
			// Consider adding BuildFlags if needed, e.g., "-tags=yourtag"
		},
	}
}

func (l *GoPackagesLoader) Load(path string) ([]*packages.Package, error) {
	cfg := l.Config // Copy base config
	cfg.Dir = path  // Set the directory for the current load operation

	pkgs, err := packages.Load(&cfg, path) // Load using the path as pattern
	if err != nil {
		return nil, fmt.Errorf("loading packages from %s: %w", path, err)
	}

	// It's good practice to report errors but not necessarily fail entirely
	// if some packages loaded successfully. The caller can decide.
	if packages.PrintErrors(pkgs) > 0 {
		log.Printf("Warning: Encountered errors during package loading from %s, analysis might be incomplete.", path)
	}

	// Filter out packages that completely failed to load types (essential for analysis)
	var validPkgs []*packages.Package
	for _, pkg := range pkgs {
		if pkg.Types != nil || len(pkg.Errors) == 0 { // Keep packages with types or no errors
			validPkgs = append(validPkgs, pkg)
		} else {
			log.Printf("Skipping package %s due to critical loading errors (no types/syntax).", pkg.ID)
		}
	}

	if len(validPkgs) == 0 && len(pkgs) > 0 {
		return nil, fmt.Errorf("no valid packages could be loaded from %s", path)
	}

	return validPkgs, nil
}

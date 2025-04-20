// analyzer/analyzer.go
package analyzer

import (
	"go/token"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"

	"github.com/namikmesic/go-mcp/internal/datamodel" // Adjusted import path
)

// InterfaceAnalyzer extracts interface definitions from packages.
type InterfaceAnalyzer interface {
	// AnalyzeInterfaces analyzes packages and returns a map where the key is a unique identifier
	// (e.g., packagePath + "." + interfaceName) and the value is the Interface details.
	AnalyzeInterfaces(pkgs []*packages.Package) (map[string]*datamodel.Interface, error)
}

// ImplementationFinder finds implementations of interfaces across packages.
// It needs the interfaces found previously.
type ImplementationFinder interface {
	// FindImplementations searches through the packages to find types that implement the interfaces
	// provided in the 'interfaces' map. It modifies the Implementations field within the map's values.
	FindImplementations(
		pkgs []*packages.Package,
		interfaces map[string]*datamodel.Interface, // Pass in the interfaces to find impls for
		fset *token.FileSet, // FileSet needed for locating implementation types
	) error // Modifies the Implementations field in the passed interfaces map
}

// CallGraphAnalyzer extracts call site information using SSA.
type CallGraphAnalyzer interface {
	// AnalyzeCalls builds the SSA representation and extracts call sites.
	// It returns a map linking original packages to their call sites, the built SSA program,
	// and the FileSet used by SSA (crucial for consistent positioning).
	AnalyzeCalls(
		pkgs []*packages.Package,
	) (map[*packages.Package][]datamodel.CallSite, *ssa.Program, *token.FileSet, error)
}

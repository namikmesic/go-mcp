// datamodel/datamodel.go
package datamodel

import (
	"go/token"
	"go/types"
)

// Location represents a file:line:column position.
type Location struct {
	Filename string
	Line     int
	Column   int
}

// Parameter represents information about a function/method parameter.
type Parameter struct {
	Name      string
	Type      string
	IsPointer bool
	// Could add Location here if needed
}

// Method represents detailed information about an interface method.
type Method struct {
	Name        string
	Signature   string
	Parameters  []Parameter
	ReturnTypes []string
	DocComment  string
	Location    Location
}

// Implementation represents a concrete type that implements an interface.
type Implementation struct {
	TypeName    string
	PackagePath string
	PackageName string
	IsPointer   bool
	Location    Location // Location of the type definition
}

// Interface represents information about a found interface.
type Interface struct {
	Name            string
	PackageName     string // Package where the interface is defined
	PackagePath     string // Import path of the defining package
	Location        Location
	DocComment      string
	Methods         []Method
	Embeds          []string // Fully qualified names of embedded interfaces
	Implementations []Implementation
	// Keep underlying type info if needed for advanced analysis downstream
	UnderlyingType *types.Interface
}

// CallSite represents information about a single call site.
type CallSite struct {
	CallerFuncDesc string   // Description of the function/method containing the call
	CalleeDesc     string   // Description of the called function/method/interface method
	CallType       string   // Static, Interface, Go, Defer
	Location       Location // File:line:column of the call site
}

// ModuleInfo holds information about the Go module.
type ModuleInfo struct {
	Path    string
	Version string
	Dir     string
	GoMod   string
	IsMain  bool
}

// PackageAnalysis holds all analyzed information for a single Go package.
type PackageAnalysis struct {
	Name          string
	Path          string
	Files         []string
	Imports       []string // Import paths
	Module        *ModuleInfo
	EmbedFiles    []string
	EmbedPatterns []string
	Interfaces    []Interface
	Calls         []CallSite
	// Store original package and SSA for potential advanced use? Optional.
	// OriginalPackage *packages.Package
	// SsaPackage      *ssa.Package
}

// ProjectAnalysis holds the analysis results for all packages in the project.
type ProjectAnalysis struct {
	Packages []*PackageAnalysis
	// Could add cross-package analysis results here later
	// Could add the *ssa.Program here if needed globally
}

// Helper to create Location from token.Position
func NewLocation(pos token.Position) Location {
	return Location{
		Filename: pos.Filename,
		Line:     pos.Line,
		Column:   pos.Column,
	}
}

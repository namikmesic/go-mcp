// datamodel/datamodel.go
package datamodel

import (
	"encoding/json"
	"go/token"
	"go/types"
)

// Location represents a file:line:column position.
type Location struct {
	Filename string `json:"Filename"`
	Line     int    `json:"Line"`
	Column   int    `json:"-"` // Exclude from JSON output
}

// Parameter represents information about a function/method parameter.
type Parameter struct {
	Name      string `json:"Name"`
	Type      string `json:"Type"`
	IsPointer bool   `json:"IsPointer"`
	// Could add Location here if needed
}

// Method represents detailed information about an interface method.
type Method struct {
	Name        string      `json:"Name"`
	Signature   string      `json:"Signature"`
	Parameters  []Parameter `json:"Parameters"`
	ReturnTypes []string    `json:"ReturnTypes"`
	DocComment  string      `json:"DocComment"`
	Location    Location    `json:"Location"`
}

// Implementation represents a concrete type that implements an interface.
type Implementation struct {
	TypeName    string   `json:"TypeName"`
	PackagePath string   `json:"PackagePath"`
	PackageName string   `json:"PackageName"`
	IsPointer   bool     `json:"IsPointer"`
	Location    Location `json:"Location"` // Location of the type definition
}

// Interface represents information about a found interface.
type Interface struct {
	Name            string           `json:"Name"`
	PackageName     string           `json:"PackageName"` // Package where the interface is defined
	PackagePath     string           `json:"PackagePath"` // Import path of the defining package
	Location        Location         `json:"Location"`
	DocComment      string           `json:"DocComment"`
	Methods         []Method         `json:"Methods"`
	Embeds          []string         `json:"Embeds"` // Fully qualified names of embedded interfaces
	Implementations []Implementation `json:"Implementations"`
	// Keep underlying type info if needed for advanced analysis downstream
	UnderlyingType *types.Interface `json:"-"` // Exclude from direct JSON marshaling, we'll handle it in MarshalJSON
}

// MarshalJSON implements json.Marshaler for Interface to handle conditional inclusion of UnderlyingType
func (i Interface) MarshalJSON() ([]byte, error) {
	type InterfaceAlias Interface // Avoid recursion in MarshalJSON

	// Create a map representation of the struct without UnderlyingType
	m := map[string]interface{}{
		"Name":            i.Name,
		"PackageName":     i.PackageName,
		"PackagePath":     i.PackagePath,
		"Location":        i.Location,
		"DocComment":      i.DocComment,
		"Methods":         i.Methods,
		"Embeds":          i.Embeds,
		"Implementations": i.Implementations,
	}

	// We're omitting UnderlyingType completely as it's only used for internal analysis

	return json.Marshal(m)
}

// CallSite represents information about a single call site.
type CallSite struct {
	CallerFuncDesc string   `json:"CallerFuncDesc"` // Description of the function/method containing the call
	CalleeDesc     string   `json:"CalleeDesc"`     // Description of the called function/method/interface method
	CallType       string   `json:"CallType"`       // Static, Interface, Go, Defer
	Location       Location `json:"Location"`       // File:line:column of the call site
}

// ModuleInfo holds information about the Go module.
type ModuleInfo struct {
	Path    string `json:"Path"`
	Version string `json:"Version"`
	Dir     string `json:"Dir"`
	GoMod   string `json:"GoMod"`
	IsMain  bool   `json:"IsMain"`
}

// PackageAnalysis holds all analyzed information for a single Go package.
type PackageAnalysis struct {
	Name          string      `json:"Name"`
	Path          string      `json:"Path"`
	Files         []string    `json:"Files"`
	Imports       []string    `json:"Imports"` // Import paths
	EmbedFiles    []string    `json:"EmbedFiles,omitempty"`
	EmbedPatterns []string    `json:"EmbedPatterns,omitempty"`
	Interfaces    []Interface `json:"Interfaces"`
	Calls         []CallSite  `json:"Calls,omitempty"`
	// Store original package and SSA for potential advanced use? Optional.
	// OriginalPackage *packages.Package
	// SsaPackage      *ssa.Package
}

// ProjectAnalysis holds the analysis results for all packages in the project.
type ProjectAnalysis struct {
	// New top-level fields for module information
	ModulePath string             `json:"ModulePath"`
	ModuleDir  string             `json:"ModuleDir"`
	Packages   []*PackageAnalysis `json:"Packages"`
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

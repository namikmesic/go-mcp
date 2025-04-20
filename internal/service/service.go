// service/service.go
package service

import (
	"fmt"
	"go/token" // Import token needed by ImplementationFinder
	"log"
	"path/filepath"

	"github.com/namikmesic/go-mcp/internal/analyzer"  // Adjusted import path
	"github.com/namikmesic/go-mcp/internal/datamodel" // Adjusted import path
	"github.com/namikmesic/go-mcp/internal/loader"    // Adjusted import path
	"golang.org/x/tools/go/packages"                  // Import needed for map key type
)

// AnalysisService orchestrates the loading and analysis of Go projects.
type AnalysisService struct {
	loader               loader.Loader
	interfaceAnalyzer    analyzer.InterfaceAnalyzer
	implementationFinder analyzer.ImplementationFinder
	callGraphAnalyzer    analyzer.CallGraphAnalyzer
}

// NewAnalysisService creates a new service with the required components.
func NewAnalysisService(
	l loader.Loader,
	ia analyzer.InterfaceAnalyzer,
	idf analyzer.ImplementationFinder,
	cga analyzer.CallGraphAnalyzer,
) *AnalysisService {
	// Basic validation of inputs
	if l == nil || ia == nil || idf == nil || cga == nil {
		// In a real app, might return an error or panic
		log.Panicln("Error: Cannot create AnalysisService with nil components.")
	}
	return &AnalysisService{
		loader:               l,
		interfaceAnalyzer:    ia,
		implementationFinder: idf,
		callGraphAnalyzer:    cga,
	}
}

// AnalyzeProject loads and analyzes the Go project at the given path.
func (s *AnalysisService) AnalyzeProject(path string) (*datamodel.ProjectAnalysis, error) {
	log.Printf("Loading packages from directory: %s", path)
	pkgs, err := s.loader.Load(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load packages: %w", err)
	}
	if len(pkgs) == 0 {
		// Check if the loader itself returned an error previously
		// If not, it means Load succeeded but found nothing valid.
		return nil, fmt.Errorf("no valid Go packages found or loaded from %s", path)
	}
	log.Printf("Successfully loaded %d package(s) for analysis.", len(pkgs))

	// Determine module information - use the first package with a non-nil module
	var moduleInfo *datamodel.ModuleInfo
	var moduleDir string
	var modulePath string
	for _, pkg := range pkgs {
		if pkg != nil && pkg.Module != nil {
			moduleInfo = &datamodel.ModuleInfo{
				Path:    pkg.Module.Path,
				Version: pkg.Module.Version,
				Dir:     pkg.Module.Dir,
				GoMod:   pkg.Module.GoMod,
				IsMain:  pkg.Module.Main,
			}
			moduleDir = pkg.Module.Dir
			modulePath = pkg.Module.Path
			break
		}
	}
	if moduleInfo == nil {
		log.Println("Warning: No module information found for any package.")
	} else {
		log.Printf("Using module: path=%s, dir=%s", modulePath, moduleDir)
	}

	log.Println("Analyzing interfaces...")
	// interfacesMap key: packagePath + "." + interfaceName
	interfacesMap, err := s.interfaceAnalyzer.AnalyzeInterfaces(pkgs)
	if err != nil {
		// Depending on severity, might log and continue or return error
		log.Printf("Warning: Interface analysis failed: %v. Proceeding without interface data.", err)
		interfacesMap = make(map[string]*datamodel.Interface) // Ensure map is non-nil
	} else {
		log.Printf("Found %d unique interface definitions.", len(interfacesMap))
	}

	log.Println("Analyzing calls (building SSA)...")
	// callsByPackage key: *packages.Package
	var callsByPackage map[*packages.Package][]datamodel.CallSite
	var ssaFset *token.FileSet // FileSet from SSA is crucial for consistent positions

	callsByPackage, _, ssaFset, err = s.callGraphAnalyzer.AnalyzeCalls(pkgs)
	if err != nil {
		// Call graph analysis is often critical. Log details and fail.
		log.Printf("Error: Call graph analysis failed: %v", err)
		return nil, fmt.Errorf("failed during call graph analysis: %w", err)
	}
	callCount := 0
	for _, calls := range callsByPackage {
		callCount += len(calls)
	}
	log.Printf("Found %d call sites across %d packages.", callCount, len(callsByPackage))
	if ssaFset == nil {
		// This should ideally be caught by AnalyzeCalls, but double-check
		log.Println("Error: Call graph analysis succeeded but returned a nil FileSet. Location data will be inconsistent.")
		return nil, fmt.Errorf("call graph analysis returned nil FileSet")
	}

	log.Println("Finding implementations...")
	// Pass the FileSet obtained from SSA to the implementation finder
	err = s.implementationFinder.FindImplementations(pkgs, interfacesMap, ssaFset)
	if err != nil {
		// Implementation finding might be less critical than calls for some use cases.
		log.Printf("Warning: Implementation finding failed: %v. Proceeding without implementation data.", err)
		// If continuing, ensure Implementations slices are empty, not nil
		for _, iface := range interfacesMap {
			if iface.Implementations == nil {
				iface.Implementations = []datamodel.Implementation{}
			}
		}
	} else {
		implCount := 0
		for _, iface := range interfacesMap {
			implCount += len(iface.Implementations)
		}
		log.Printf("Found %d implementation relationships.", implCount)
	}

	// --- Assemble the final result ---
	log.Println("Assembling final analysis results...")
	projectAnalysis := &datamodel.ProjectAnalysis{
		ModulePath: modulePath,
		ModuleDir:  moduleDir,
		Packages:   make([]*datamodel.PackageAnalysis, 0, len(pkgs)),
	}

	// Create a map for quick lookup of interfaces belonging to a package path
	interfacesByPkgPath := make(map[string][]datamodel.Interface)
	for _, iface := range interfacesMap {
		// Make location filenames relative to module directory
		if moduleDir != "" && filepath.IsAbs(iface.Location.Filename) {
			relPath, err := filepath.Rel(moduleDir, iface.Location.Filename)
			if err == nil {
				iface.Location.Filename = relPath
			}
		}

		// Make method location filenames relative
		for i := range iface.Methods {
			if moduleDir != "" && filepath.IsAbs(iface.Methods[i].Location.Filename) {
				relPath, err := filepath.Rel(moduleDir, iface.Methods[i].Location.Filename)
				if err == nil {
					iface.Methods[i].Location.Filename = relPath
				}
			}
		}

		// Make implementation location filenames relative
		for i := range iface.Implementations {
			if moduleDir != "" && filepath.IsAbs(iface.Implementations[i].Location.Filename) {
				relPath, err := filepath.Rel(moduleDir, iface.Implementations[i].Location.Filename)
				if err == nil {
					iface.Implementations[i].Location.Filename = relPath
				}
			}
		}

		// Ensure the slice exists before appending
		if _, ok := interfacesByPkgPath[iface.PackagePath]; !ok {
			interfacesByPkgPath[iface.PackagePath] = []datamodel.Interface{}
		}
		interfacesByPkgPath[iface.PackagePath] = append(interfacesByPkgPath[iface.PackagePath], *iface)
	}

	// Make call site location filenames relative
	for pkg, calls := range callsByPackage {
		for i := range calls {
			if moduleDir != "" && filepath.IsAbs(calls[i].Location.Filename) {
				relPath, err := filepath.Rel(moduleDir, calls[i].Location.Filename)
				if err == nil {
					calls[i].Location.Filename = relPath
				}
			}
		}
		callsByPackage[pkg] = calls
	}

	// Populate PackageAnalysis for each loaded package
	for _, pkg := range pkgs {
		// Basic check if pkg is valid
		if pkg == nil || pkg.PkgPath == "" {
			log.Printf("Warning: Skipping assembly for a nil or invalid package.")
			continue
		}

		// Make file paths relative to module directory if possible
		relativeFiles := make([]string, 0, len(pkg.GoFiles))
		for _, file := range pkg.GoFiles {
			// Only make paths relative if we have module information
			if moduleDir != "" && filepath.IsAbs(file) {
				relPath, err := filepath.Rel(moduleDir, file)
				if err == nil {
					relativeFiles = append(relativeFiles, relPath)
				} else {
					// If we can't make it relative, keep the original path
					relativeFiles = append(relativeFiles, file)
				}
			} else {
				relativeFiles = append(relativeFiles, file)
			}
		}

		pkgAnalysis := &datamodel.PackageAnalysis{
			Name:          pkg.Name,
			Path:          pkg.PkgPath,
			Files:         relativeFiles, // Now using relative file paths
			Imports:       make([]string, 0, len(pkg.Imports)),
			EmbedFiles:    pkg.EmbedFiles,                   // Relative to package dir
			EmbedPatterns: pkg.EmbedPatterns,                // Relative to package dir
			Interfaces:    interfacesByPkgPath[pkg.PkgPath], // Get interfaces for this package path
			Calls:         callsByPackage[pkg],              // Get calls for this package (*packages.Package key)
		}

		// Ensure slices are non-nil for JSON marshalling
		if pkgAnalysis.Files == nil {
			pkgAnalysis.Files = []string{}
		}
		if pkgAnalysis.Imports == nil {
			pkgAnalysis.Imports = []string{}
		}
		if pkgAnalysis.EmbedFiles == nil {
			pkgAnalysis.EmbedFiles = []string{}
		}
		if pkgAnalysis.EmbedPatterns == nil {
			pkgAnalysis.EmbedPatterns = []string{}
		}
		if pkgAnalysis.Interfaces == nil {
			pkgAnalysis.Interfaces = []datamodel.Interface{}
		}
		if pkgAnalysis.Calls == nil {
			pkgAnalysis.Calls = []datamodel.CallSite{}
		}

		// Populate import paths
		for path := range pkg.Imports {
			pkgAnalysis.Imports = append(pkgAnalysis.Imports, path)
		}

		projectAnalysis.Packages = append(projectAnalysis.Packages, pkgAnalysis)
	}
	log.Printf("Assembled results for %d packages.", len(projectAnalysis.Packages))
	log.Println("Analysis complete.")

	return projectAnalysis, nil
}

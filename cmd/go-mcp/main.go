package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath" // Import filepath for absolute paths

	// Adjust import paths according to your project structure and module name
	"github.com/namikmesic/go-mcp/internal/analyzer/ast"
	"github.com/namikmesic/go-mcp/internal/analyzer/ssa"
	"github.com/namikmesic/go-mcp/internal/analyzer/typesystem"
	"github.com/namikmesic/go-mcp/internal/loader"
	"github.com/namikmesic/go-mcp/internal/service"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <path-to-go-project-or-package>")
		fmt.Println("  Example: go run main.go .")
		fmt.Println("  Example: go run main.go ./...") // Usually handled by loader now
		fmt.Println("  Example: go run main.go /path/to/your/project")
		os.Exit(1)
	}
	// The argument should be the directory containing the code (or where go.mod resides)
	targetPathArg := os.Args[1]

	// Ensure the path is absolute for consistency, especially for the loader's Dir config.
	targetPath, err := filepath.Abs(targetPathArg)
	if err != nil {
		log.Fatalf("Error converting path %s to absolute path: %v", targetPathArg, err)
	}

	// Check if the target path exists and is a directory
	info, err := os.Stat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatalf("Error: Target path does not exist: %s", targetPath)
		}
		log.Fatalf("Error accessing target path %s: %v", targetPath, err)
	}
	if !info.IsDir() {
		log.Fatalf("Error: Target path must be a directory: %s", targetPath)
	}

	log.Printf("Starting analysis for directory: %s", targetPath)

	// --- Dependency Injection ---
	// Create concrete instances of our components
	pkgLoader := loader.NewGoPackagesLoader()
	ifAnalyzer := ast.NewASTInterfaceAnalyzer()
	implFinder := typesystem.NewTypeBasedImplementationFinder()
	callAnalyzer := ssa.NewSSACallGraphAnalyzer()

	// Create the analysis service, injecting the components
	analysisService := service.NewAnalysisService(
		pkgLoader,
		ifAnalyzer,
		implFinder,
		callAnalyzer,
	)
	// --- End Dependency Injection ---

	// Run the analysis using the absolute path
	projectAnalysis, err := analysisService.AnalyzeProject(targetPath)
	if err != nil {
		log.Fatalf("Analysis failed: %v", err)
	}

	// --- Output ---
	// Output the results as JSON to standard output
	fmt.Println("\n===== ANALYSIS RESULTS (JSON) =====")
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ") // Pretty print JSON
	if err := encoder.Encode(projectAnalysis); err != nil {
		log.Fatalf("Failed to encode results to JSON: %v", err)
	}

	// Optional: Print summary after JSON output
	fmt.Fprintf(os.Stderr, "\n===== ANALYSIS SUMMARY =====\n") // Print summary to Stderr
	if projectAnalysis != nil {
		totalPackages := len(projectAnalysis.Packages)
		fmt.Fprintf(os.Stderr, "Analyzed %d packages.\n", totalPackages)
		totalInterfaces := 0
		totalCalls := 0
		totalImpls := 0
		for _, pkg := range projectAnalysis.Packages {
			if pkg == nil {
				continue
			}
			totalInterfaces += len(pkg.Interfaces)
			totalCalls += len(pkg.Calls)
			for _, iface := range pkg.Interfaces {
				totalImpls += len(iface.Implementations)
			}
		}
		fmt.Fprintf(os.Stderr, "Found %d interface definitions.\n", totalInterfaces)
		fmt.Fprintf(os.Stderr, "Found %d implementation relationships.\n", totalImpls)
		fmt.Fprintf(os.Stderr, "Found %d call sites.\n", totalCalls)
	} else {
		fmt.Fprintln(os.Stderr, "Project analysis result was nil.")
	}
	fmt.Fprintln(os.Stderr, "============================")
}

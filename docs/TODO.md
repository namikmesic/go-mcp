# Go-MCP Improvement Tasks

This document outlines specific tasks for improving the Go Module Call Path Analyzer (go-mcp) project. Each task includes detailed instructions and code examples where appropriate.

## Command Line Interface

### Task 1: Improve Error Handling in main.go

Replace generic `log.Fatalf` calls with more specific error handling that provides clearer messages to the user:

```go
// FIND:
if err != nil {
    log.Fatalf("Error accessing target path %s: %v", targetPath, err)
}

// REPLACE WITH:
if err != nil {
    if os.IsNotExist(err) {
        fmt.Fprintf(os.Stderr, "Error: Target path does not exist: %s\n", targetPath)
    } else {
        fmt.Fprintf(os.Stderr, "Error accessing target path %s: %v\n", targetPath, err)
    }
    os.Exit(1)
}
```

**Acceptance Criteria:**
- All `log.Fatalf` calls in main.go are replaced with specific error handling
- Error messages are written to stderr using `fmt.Fprintf(os.Stderr, ...)`
- Program exits with a non-zero status code on errors

### Task 2: Add Command-Line Flags for Configuration

Add command-line flags to make the tool more configurable:

```go
// Add at the top of main.go:
var (
    verbose      = flag.Bool("verbose", false, "Enable verbose logging")
    outputFormat = flag.String("format", "json", "Output format: json or summary")
    skipCalls    = flag.Bool("skip-calls", false, "Skip call graph analysis (faster)")
    skipImpls    = flag.Bool("skip-impls", false, "Skip implementation finding (faster)")
    neo4jUri     = flag.String("neo4j-uri", "", "Neo4j URI for storing results (optional)")
    neo4jUser    = flag.String("neo4j-user", "", "Neo4j username")
    neo4jPass    = flag.String("neo4j-pass", "", "Neo4j password")
)

// Update in main() function:
flag.Parse()
if flag.NArg() < 1 {
    fmt.Println("Usage: go-mcp [flags] <path-to-go-project-or-package>")
    fmt.Println("Example: go-mcp .")
    fmt.Println("Example: go-mcp ./...")
    fmt.Println("Example: go-mcp /path/to/your/project")
    flag.PrintDefaults()
    os.Exit(1)
}
targetPathArg := flag.Arg(0)
```

**Acceptance Criteria:**
- Command-line flags are added for key configuration options
- Usage information is displayed when no arguments are provided
- Flags are properly parsed and used throughout the application

### Task 3: Add Progress Indicators for Long-Running Operations

Implement progress indicators to provide feedback during long-running operations:

```go
// Add to main.go:
func printProgress(operation string, current, total int) {
    if total > 0 {
        percentage := float64(current) / float64(total) * 100
        fmt.Fprintf(os.Stderr, "\r%s: %d/%d (%.1f%%)...", operation, current, total, percentage)
    } else {
        fmt.Fprintf(os.Stderr, "\r%s: %d...", operation, current)
    }
}

// Example usage:
fmt.Fprintln(os.Stderr, "Analyzing interfaces...")
totalPackages := len(pkgs)
for i, pkg := range pkgs {
    if *verbose {
        printProgress("Analyzing interfaces", i+1, totalPackages)
    }
    // analysis code
}
if *verbose {
    fmt.Fprintln(os.Stderr, "") // Empty line after progress indicator
}
```

**Acceptance Criteria:**
- Progress indicators are added for long-running operations (package loading, interface analysis, etc.)
- Progress is only shown when verbose mode is enabled
- Progress indicators do not interfere with regular output

## Loader Package

### Task 4: Centralize Path Handling Logic

Extract duplicated path handling logic into a common utility function:

```go
// Add to internal/analyzer/utils/path_utils.go:
package utils

import (
    "path/filepath"
    "strings"
)

// NormalizePath standardizes a path pattern for analysis.
// It handles recursive patterns (those ending with /...) and
// returns both the normalized path and the directory to use.
func NormalizePath(path string) (pattern string, dir string) {
    // Remove trailing separator if present
    normalizedPath := path
    if len(normalizedPath) > 0 && normalizedPath[len(normalizedPath)-1] == filepath.Separator {
        normalizedPath = normalizedPath[:len(normalizedPath)-1]
    }

    // Set directory equal to normalized path by default
    dir = normalizedPath
    pattern = normalizedPath

    // Special handling for /... suffix
    recursiveSuffix := string(filepath.Separator) + "..."
    if strings.HasSuffix(pattern, recursiveSuffix) {
        // Remove the /... suffix to get the actual directory
        dir = pattern[:len(pattern)-len(recursiveSuffix)]
        // Use "." + separator + "..." as the standard recursive pattern
        pattern = "." + recursiveSuffix
    }

    return pattern, dir
}
```

**Acceptance Criteria:**
- Path handling logic is extracted to a common utility function
- The utility function is used in both main.go and loader.go
- Path handling is consistent across the application

### Task 5: Implement Package Caching

Add caching to the loader to avoid redundant package loading:

```go
// Update the GoPackagesLoader struct:
type GoPackagesLoader struct {
    Config packages.Config
    cache  map[string][]*packages.Package
}

// Update the NewGoPackagesLoader function:
func NewGoPackagesLoader() *GoPackagesLoader {
    return &GoPackagesLoader{
        Config: packages.Config{
            // ... existing config ...
        },
        cache: make(map[string][]*packages.Package),
    }
}

// Update the Load method:
func (l *GoPackagesLoader) Load(path string) ([]*packages.Package, error) {
    // Check cache first
    if cachedPkgs, ok := l.cache[path]; ok {
        log.Printf("Using cached packages for path: %s", path)
        return cachedPkgs, nil
    }

    // ... existing loading logic ...

    // Cache the result
    l.cache[path] = validPkgs
    return validPkgs, nil
}
```

**Acceptance Criteria:**
- Package caching is implemented in the loader
- Cache is checked before loading packages
- Performance is improved for repeated analyses of the same packages

## Interface Analyzer

### Task 6: Refactor AST Traversal for Better Maintainability

Break down the complex AST traversal function into smaller, more focused functions:

```go
// Refactor the AnalyzeInterfaces method:
func (a *ASTInterfaceAnalyzer) AnalyzeInterfaces(pkgs []*packages.Package) (map[string]*datamodel.Interface, error) {
    interfaces := make(map[string]*datamodel.Interface)

    for _, pkg := range pkgs {
        if !a.isPackageAnalyzable(pkg) {
            continue
        }

        for _, file := range pkg.Syntax {
            if file == nil {
                continue
            }
            
            ast.Inspect(file, func(n ast.Node) bool {
                typeSpec, ok := n.(*ast.TypeSpec)
                if !ok || typeSpec.Name == nil {
                    return true
                }

                interfaceType, ok := typeSpec.Type.(*ast.InterfaceType)
                if !ok {
                    return true
                }

                // Validate interface definition
                if !a.isValidInterfaceDefinition(typeSpec, pkg) {
                    return true
                }

                // Create interface model
                iface := a.createInterfaceModel(typeSpec, interfaceType, pkg)
                
                // Store interface if not duplicate
                mapKey := pkg.PkgPath + "." + iface.Name
                if _, exists := interfaces[mapKey]; !exists {
                    interfaces[mapKey] = iface
                } else {
                    log.Printf("Warning: Duplicate interface definition encountered for %s. Keeping first.", mapKey)
                }
                
                return true
            })
        }
    }
    
    return interfaces, nil
}

// Helper functions
func (a *ASTInterfaceAnalyzer) isPackageAnalyzable(pkg *packages.Package) bool {
    if pkg.Types == nil || pkg.Fset == nil || len(pkg.Syntax) == 0 || pkg.TypesInfo == nil {
        log.Printf("Skipping package %s for interface analysis: missing types, fileset, syntax trees, or types info.", pkg.ID)
        return false
    }
    return true
}

func (a *ASTInterfaceAnalyzer) isValidInterfaceDefinition(typeSpec *ast.TypeSpec, pkg *packages.Package) bool {
    // Check if definition exists in TypesInfo
    obj := pkg.TypesInfo.Defs[typeSpec.Name]
    if obj == nil {
        log.Printf("Warning: No type definition object found for %s in package %s using TypesInfo.Defs, skipping.", typeSpec.Name.Name, pkg.PkgPath)
        return false
    }
    
    // Check if it's an interface type
    _, ok := obj.Type().Underlying().(*types.Interface)
    return ok
}

func (a *ASTInterfaceAnalyzer) createInterfaceModel(typeSpec *ast.TypeSpec, interfaceType *ast.InterfaceType, pkg *packages.Package) *datamodel.Interface {
    defPos := pkg.Fset.Position(typeSpec.Name.Pos())
    iface := &datamodel.Interface{
        Name:            typeSpec.Name.Name,
        PackageName:     pkg.Name,
        PackagePath:     pkg.PkgPath,
        Location:        datamodel.NewLocation(defPos),
        Methods:         []datamodel.Method{},
        Embeds:          []string{},
        Implementations: []datamodel.Implementation{},
    }

    if typeSpec.Doc != nil {
        iface.DocComment = strings.TrimSpace(typeSpec.Doc.Text())
    }

    if interfaceType.Methods != nil {
        a.extractMethodsAndEmbeds(interfaceType, iface, pkg)
    }

    return iface
}

func (a *ASTInterfaceAnalyzer) extractMethodsAndEmbeds(interfaceType *ast.InterfaceType, iface *datamodel.Interface, pkg *packages.Package) {
    for _, field := range interfaceType.Methods.List {
        if field == nil {
            continue
        }

        // Embedded interface
        if len(field.Names) == 0 && field.Type != nil {
            embedName := utils.ExprToString(field.Type, pkg)
            if embedName != "" && embedName != "?" {
                iface.Embeds = append(iface.Embeds, embedName)
            }
            continue
        }

        // Regular method
        if len(field.Names) > 0 && field.Names[0] != nil && field.Type != nil {
            methodName := field.Names[0].Name
            methodPos := pkg.Fset.Position(field.Pos())
            methodInfo := a.createMethodModel(methodName, field, methodPos, pkg)
            iface.Methods = append(iface.Methods, methodInfo)
        }
    }
}

func (a *ASTInterfaceAnalyzer) createMethodModel(methodName string, field *ast.Field, methodPos token.Position, pkg *packages.Package) datamodel.Method {
    methodInfo := datamodel.Method{
        Name:        methodName,
        Location:    datamodel.NewLocation(methodPos),
        Parameters:  []datamodel.Parameter{},
        ReturnTypes: []string{},
    }

    if field.Doc != nil {
        methodInfo.DocComment = strings.TrimSpace(field.Doc.Text())
    }

    if funcType, ok := field.Type.(*ast.FuncType); ok {
        methodInfo.Signature = utils.FormatMethodSignature(methodName, funcType, pkg)
        methodInfo.Parameters = utils.ExtractParameters(funcType, pkg)
        methodInfo.ReturnTypes = utils.ExtractReturnTypes(funcType, pkg)
    } else {
        log.Printf("Warning: Method '%s' in interface '%s' (pkg: %s) has non-function type %T", methodName, field.Names[0].Name, pkg.PkgPath, field.Type)
        methodInfo.Signature = methodName + "(...) // Analysis Error: Non-FuncType"
    }

    return methodInfo
}
```

**Acceptance Criteria:**
- The single large function is broken down into multiple smaller functions
- Each function has a single responsibility
- Code is more maintainable and easier to understand
- Functionality remains identical

## Call Graph Analyzer

### Task 7: Improve SSA Package Mapping

Replace the index-based mapping with a more reliable path-based mapping:

```go
// FIND in AnalyzeCalls method:
ssaToOrigMap := make(map[*ssa.Package]*packages.Package)
for i, ssaPkg := range ssaPkgs {
    // Ensure index is valid and ssaPkg is not nil
    if i < len(pkgs) && ssaPkg != nil {
        // Check if the original package exists and is valid
        if pkgs[i] != nil {
            ssaToOrigMap[ssaPkg] = pkgs[i]
        } else {
            log.Printf("Warning: SSA package %s corresponds to a nil original package at index %d.", ssaPkg.Pkg.Path(), i)
        }
    } else if ssaPkg != nil {
        // This might happen if ssautil includes packages not in the original input list
    }
}

// REPLACE WITH:
ssaToOrigMap := make(map[*ssa.Package]*packages.Package)
// Create a lookup map by package path
pkgPathMap := make(map[string]*packages.Package)
for _, pkg := range pkgs {
    if pkg != nil {
        pkgPathMap[pkg.PkgPath] = pkg
    }
}

// Map SSA packages to original packages by path
for _, ssaPkg := range ssaPkgs {
    if ssaPkg == nil || ssaPkg.Pkg == nil {
        continue
    }
    
    ssaPkgPath := ssaPkg.Pkg.Path()
    if origPkg, ok := pkgPathMap[ssaPkgPath]; ok {
        ssaToOrigMap[ssaPkg] = origPkg
    } else {
        log.Printf("Warning: Could not map SSA package '%s' back to original package. It may be a dependency.", ssaPkgPath)
    }
}
```

**Acceptance Criteria:**
- The mapping from SSA packages to original packages uses package paths instead of array indices
- The mapping is more reliable, especially when package ordering differs
- Warning messages are clear about packages that couldn't be mapped

### Task 8: Extract Call Information Processing

Refactor the call processing logic into a separate function:

```go
// Add to call_analyzer.go:
func (a *SSACallGraphAnalyzer) processCallInstruction(call ssa.CallInstruction, callerName string) (string, string) {
    callType := "Unknown"
    calleeDesc := "Unknown Call"
    
    common := call.Common()
    if common == nil {
        return callType, calleeDesc
    }
    
    switch c := call.(type) {
    case *ssa.Call:
        if common.IsInvoke() {
            callType = "Interface"
            if common.Method != nil && common.Value != nil && common.Value.Type() != nil {
                calleeDesc = fmt.Sprintf("Interface method %s on %s", common.Method.Name(), types.TypeString(common.Value.Type(), nil))
            } else {
                calleeDesc = "Unknown Interface Call (nil method/value/type)"
            }
        } else {
            callee := common.StaticCallee()
            if callee != nil {
                callType = "Static"
                calleeDesc = callee.String()
            } else if common.Value != nil && common.Value.Type() != nil {
                callType = "Dynamic"
                name := common.Value.Name()
                if name == "" {
                    name = "anonymous_func_value"
                }
                calleeDesc = fmt.Sprintf("Dynamic via %s (%s)", name, types.TypeString(common.Value.Type(), nil))
            } else {
                calleeDesc = "Unknown Static/Dynamic Call"
            }
        }
    case *ssa.Go:
        callType = "Go"
        // Similar logic as above for the callee description
        // ...
    case *ssa.Defer:
        callType = "Defer"
        // Similar logic as above for the callee description
        // ...
    default:
        callType = fmt.Sprintf("Unknown-%T", c)
    }
    
    return callType, calleeDesc
}

// Then use this function in AnalyzeCalls:
// ... within the instruction processing loop ...
if call, ok := instr.(ssa.CallInstruction); ok {
    callType, calleeDesc := a.processCallInstruction(call, callerName)
    
    callInfo = &datamodel.CallSite{
        CallerFuncDesc: callerName,
        CalleeDesc:     calleeDesc,
        CallType:       callType,
        Location:       location,
    }
}
```

**Acceptance Criteria:**
- Call processing logic is extracted to a separate function
- The function is well-documented and handles all call types
- The original functionality is preserved

## Implementation Finder

### Task 9: Extract Location Finding Logic

Refactor the complex location finding logic into a dedicated function:

```go
// Add to implementation_finder.go:
// findTypeLocation attempts to find the location of a type definition in a package.
// It returns the location and a boolean indicating success.
func findTypeLocation(typeName *types.TypeName, pkg *packages.Package, fset *token.FileSet) (datamodel.Location, bool) {
    // Try to find the AST node first (most accurate)
    for _, syntaxFile := range pkg.Syntax {
        if syntaxFile == nil {
            continue
        }
        
        var foundNode ast.Node
        var pos token.Pos
        
        ast.Inspect(syntaxFile, func(n ast.Node) bool {
            if spec, ok := n.(*ast.TypeSpec); ok && spec.Name != nil && spec.Name.Name == typeName.Name() {
                // Verify it's the right object using TypesInfo
                if pkg.TypesInfo != nil && pkg.TypesInfo.Defs[spec.Name] == typeName {
                    foundNode = n
                    pos = spec.Name.Pos()
                    return false // Stop searching
                }
            }
            return foundNode == nil // Continue if not found yet
        })
        
        if foundNode != nil && pos.IsValid() {
            position := fset.Position(pos)
            if position.IsValid() {
                return datamodel.NewLocation(position), true
            }
        }
    }
    
    // Fallback 1: Try using typeName.Pos() with the provided fset
    if typeName.Pos().IsValid() {
        position := fset.Position(typeName.Pos())
        if position.IsValid() {
            log.Printf("Warning: Using fallback position for %s.%s", pkg.PkgPath, typeName.Name())
            return datamodel.NewLocation(position), true
        }
    }
    
    // Fallback 2: Try using pkg.Fset if provided fset failed
    if fset != pkg.Fset && pkg.Fset != nil && typeName.Pos().IsValid() {
        position := pkg.Fset.Position(typeName.Pos())
        if position.IsValid() {
            log.Printf("Warning: Using pkg.Fset fallback position for %s.%s", pkg.PkgPath, typeName.Name())
            return datamodel.NewLocation(position), true
        }
    }
    
    // Could not find a valid location
    return datamodel.Location{}, false
}

// Use this function in addImplementation:
func addImplementation(iface *datamodel.Interface, typeName *types.TypeName, pkg *packages.Package, isPointer bool, fset *token.FileSet) {
    location, found := findTypeLocation(typeName, pkg, fset)
    if !found {
        log.Printf("Skipping implementation %s.%s (pointer: %v) for interface %s due to missing location information.", 
                  pkg.PkgPath, typeName.Name(), isPointer, iface.Name)
        return
    }
    
    // Check for duplicates (existing logic)
    // ...
    
    // Add the implementation
    iface.Implementations = append(iface.Implementations, datamodel.Implementation{
        TypeName:    typeName.Name(),
        PackagePath: pkg.PkgPath,
        PackageName: pkg.Name,
        IsPointer:   isPointer,
        Location:    location,
    })
}
```

**Acceptance Criteria:**
- Location finding logic is extracted to a separate function
- The function handles all the fallback scenarios in a clear way
- Error messages are helpful for debugging
- The original functionality is preserved

### Task 10: Optimize Type Checking with Caching

Implement type caching to avoid redundant compatibility checks:

```go
// Add to implementation_finder.go:
type typeCompatibility struct {
    implementsAsValue   bool
    implementsAsPointer bool
}

type TypeBasedImplementationFinder struct {
    // Cache for type compatibility checks to avoid redundant work
    compatibilityCache map[typePair]typeCompatibility
}

type typePair struct {
    implType types.Type
    ifaceType *types.Interface
}

func NewTypeBasedImplementationFinder() *TypeBasedImplementationFinder {
    return &TypeBasedImplementationFinder{
        compatibilityCache: make(map[typePair]typeCompatibility),
    }
}

// Update the FindImplementations method:
func (f *TypeBasedImplementationFinder) FindImplementations(
    pkgs []*packages.Package,
    interfaces map[string]*datamodel.Interface,
    fset *token.FileSet,
) error {
    // ... existing setup code ...
    
    for _, pkg := range pkgs {
        // ... existing setup code ...
        
        for _, name := range scope.Names() {
            // ... existing type lookup code ...
            
            implementingType := typeName.Type()
            if implementingType == nil {
                continue
            }
            
            for typeInterface, ifaceData := range typeToInterfaceMap {
                pair := typePair{implType: implementingType, ifaceType: typeInterface}
                
                var compat typeCompatibility
                var found bool
                
                // Check cache first
                if compat, found = f.compatibilityCache[pair]; !found {
                    // Not in cache, perform the checks
                    compat.implementsAsValue = types.Implements(implementingType, typeInterface)
                    
                    // Create pointer type for pointer receiver check
                    ptrType := types.NewPointer(implementingType)
                    compat.implementsAsPointer = types.Implements(ptrType, typeInterface)
                    
                    // Store in cache
                    f.compatibilityCache[pair] = compat
                }
                
                // Use cached results
                if compat.implementsAsValue {
                    addImplementation(ifaceData, typeName, pkg, false, fset)
                }
                
                if compat.implementsAsPointer {
                    addImplementation(ifaceData, typeName, pkg, true, fset)
                }
            }
        }
    }
    
    return nil
}
```

**Acceptance Criteria:**
- Type compatibility checking is cached to avoid redundant checks
- Cache keys properly distinguish between different type-interface pairs
- Performance is improved, especially for large codebases with many types

## Data Model

### Task 11: Add Validation Methods to Data Model

Add validation methods to ensure data integrity:

```go
// Add to datamodel.go:
// Validate checks if the Interface has valid and consistent data.
func (i *Interface) Validate() error {
    if i.Name == "" {
        return fmt.Errorf("interface has empty name")
    }
    
    if i.PackagePath == "" {
        return fmt.Errorf("interface %s has empty package path", i.Name)
    }
    
    if i.Methods == nil {
        i.Methods = []Method{}
    }
    
    if i.Embeds == nil {
        i.Embeds = []string{}
    }
    
    if i.Implementations == nil {
        i.Implementations = []Implementation{}
    }
    
    // Validate methods
    for i, method := range i.Methods {
        if method.Name == "" {
            return fmt.Errorf("method at index %d has empty name", i)
        }
        
        if method.Parameters == nil {
            method.Parameters = []Parameter{}
        }
        
        if method.ReturnTypes == nil {
            method.ReturnTypes = []string{}
        }
    }
    
    return nil
}

// Similar validation methods for other types...

// ValidateProjectAnalysis validates the entire project analysis for consistency
func ValidateProjectAnalysis(pa *datamodel.ProjectAnalysis) error {
    if pa == nil {
        return fmt.Errorf("project analysis is nil")
    }
    
    if pa.Packages == nil {
        pa.Packages = []*PackageAnalysis{}
    }
    
    for i, pkg := range pa.Packages {
        if pkg == nil {
            continue
        }
        
        if pkg.Name == "" {
            return fmt.Errorf("package at index %d has empty name", i)
        }
        
        if pkg.Path == "" {
            return fmt.Errorf("package %s has empty path", pkg.Name)
        }
        
        // Validate interfaces
        for j, iface := range pkg.Interfaces {
            if err := iface.Validate(); err != nil {
                return fmt.Errorf("invalid interface %s in package %s: %w", 
                                 iface.Name, pkg.Path, err)
            }
            
            // Check consistency between interface and package
            if iface.PackagePath != pkg.Path {
                return fmt.Errorf("inconsistent package path for interface %s: %s vs %s",
                                 iface.Name, iface.PackagePath, pkg.Path)
            }
        }
    }
    
    return nil
}
```

**Acceptance Criteria:**
- Validation methods are added for all major data model types
- Validation ensures data consistency and completeness
- Methods initialize nil slices to empty slices to avoid nil JSON issues
- The validation is called at appropriate points in the codebase

### Task 12: Add Versioning to Data Model

Implement versioning for the data model to support future changes:

```go
// Update datamodel.go:
// Version information for the data model
const (
    DataModelVersion = "1.0.0"
)

// Add to ProjectAnalysis struct:
type ProjectAnalysis struct {
    // Existing fields...
    Version string `json:"version"` // Schema version
}

// Update service.go to set the version:
projectAnalysis := &datamodel.ProjectAnalysis{
    ModulePath: modulePath,
    ModuleDir:  moduleDir,
    Packages:   make([]*datamodel.PackageAnalysis, 0, len(pkgs)),
    Version:    datamodel.DataModelVersion,
}
```

**Acceptance Criteria:**
- Version information is added to the data model
- Version is included in serialized output
- Documentation explains the versioning scheme

## Neo4j Store

### Task 13: Implement Neo4j Storage Logic

Complete the Neo4j storage implementation:

```go
// Update neo4jstore.go:
// StoreAnalysis persists the analysis results to Neo4j.
func (s *Neo4jStore) StoreAnalysis(ctx context.Context, analysis *datamodel.ProjectAnalysis) error {
    if analysis == nil {
        return fmt.Errorf("cannot store nil analysis")
    }
    
    session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
    defer session.Close(ctx)
    
    // Start with constraints to ensure uniqueness
    if err := s.ensureConstraints(ctx, session); err != nil {
        return fmt.Errorf("failed to ensure constraints: %w", err)
    }
    
    // Store project info
    if err := s.storeProjectInfo(ctx, session, analysis); err != nil {
        return fmt.Errorf("failed to store project info: %w", err)
    }
    
    // Store packages
    if err := s.storePackages(ctx, session, analysis); err != nil {
        return fmt.Errorf("failed to store packages: %w", err)
    }
    
    // Store interfaces and their relationships
    if err := s.storeInterfaces(ctx, session, analysis); err != nil {
        return fmt.Errorf("failed to store interfaces: %w", err)
    }
    
    // Store call sites
    if err := s.storeCallSites(ctx, session, analysis); err != nil {
        return fmt.Errorf("failed to store call sites: %w", err)
    }
    
    return nil
}

// Ensure Neo4j constraints exist
func (s *Neo4jStore) ensureConstraints(ctx context.Context, session neo4j.SessionWithContext) error {
    constraints := []string{
        "CREATE CONSTRAINT IF NOT EXISTS FOR (p:Package) REQUIRE p.path IS UNIQUE",
        "CREATE CONSTRAINT IF NOT EXISTS FOR (i:Interface) REQUIRE (i.name, i.packagePath) IS NODE KEY",
        "CREATE CONSTRAINT IF NOT EXISTS FOR (m:Method) REQUIRE (m.name, m.interfaceName, m.packagePath) IS NODE KEY",
        "CREATE CONSTRAINT IF NOT EXISTS FOR (t:Type) REQUIRE (t.name, t.packagePath) IS NODE KEY",
    }
    
    for _, constraint := range constraints {
        _, err := session.Run(ctx, constraint, nil)
        if err != nil {
            return fmt.Errorf("failed to create constraint: %w", err)
        }
    }
    
    return nil
}

// Store project information
func (s *Neo4jStore) storeProjectInfo(ctx context.Context, session neo4j.SessionWithContext, analysis *datamodel.ProjectAnalysis) error {
    query := `
    MERGE (p:Project {path: $modulePath})
    SET p.dir = $moduleDir, p.version = $version
    `
    
    params := map[string]interface{}{
        "modulePath": analysis.ModulePath,
        "moduleDir":  analysis.ModuleDir,
        "version":    analysis.Version,
    }
    
    _, err := session.Run(ctx, query, params)
    return err
}

// Store packages
func (s *Neo4jStore) storePackages(ctx context.Context, session neo4j.SessionWithContext, analysis *datamodel.ProjectAnalysis) error {
    // Implementation details...
    return nil
}

// Store interfaces and their relationships
func (s *Neo4jStore) storeInterfaces(ctx context.Context, session neo4j.SessionWithContext, analysis *datamodel.ProjectAnalysis) error {
    // Implementation details...
    return nil
}

// Store call sites
func (s *Neo4jStore) storeCallSites(ctx context.Context, session neo4j.SessionWithContext, analysis *datamodel.ProjectAnalysis) error {
    // Implementation details...
    return nil
}
```

**Acceptance Criteria:**
- Neo4j storage implementation is complete with proper transactions
- Constraints ensure data integrity
- The schema efficiently represents the analysis data
- Error handling is robust

### Task 14: Implement Batch Operations for Neo4j

Add batch operations to improve performance when storing large analyses:

```go
// Add to neo4jstore.go:
// batchOperation executes a Neo4j operation in batches for better performance.
func (s *Neo4jStore) batchOperation(ctx context.Context, session neo4j.SessionWithContext, 
                                   items []interface{}, batchSize int,
                                   createParams func(batch []interface{}) (string, map[string]interface{})) error {
    if len(items) == 0 {
        return nil
    }
    
    for i := 0; i < len(items); i += batchSize {
        end := i + batchSize
        if end > len(items) {
            end = len(items)
        }
        
        batch := items[i:end]
        query, params := createParams(batch)
        
        if _, err := session.Run(ctx, query, params); err != nil {
            return fmt.Errorf("batch operation failed: %w", err)
        }
    }
    
    return nil
}

// Example usage in storeInterfaces:
func (s *Neo4jStore) storeInterfaces(ctx context.Context, session neo4j.SessionWithContext, analysis *datamodel.ProjectAnalysis) error {
    // Collect all interfaces
    var allInterfaces []interface{}
    for _, pkg := range analysis.Packages {
        if pkg == nil {
            continue
        }
        
        for i := range pkg.Interfaces {
            allInterfaces = append(allInterfaces, &pkg.Interfaces[i])
        }
    }
    
    // Store interfaces in batches
    return s.batchOperation(ctx, session, allInterfaces, 100, func(batch []interface{}) (string, map[string]interface{}) {
        // Build a parameterized query for this batch
        query := `
        UNWIND $interfaces AS iface
        MERGE (i:Interface {name: iface.name, packagePath: iface.packagePath})
        SET i.docComment = iface.docComment,
            i.fileName = iface.location.fileName,
            i.line = iface.location.line
        WITH i, iface
        MATCH (p:Package {path: iface.packagePath})
        MERGE (p)-[:DEFINES]->(i)
        `
        
        // Convert batch to params
        params := map[string]interface{}{
            "interfaces": convertInterfacesToParams(batch),
        }
        
        return query, params
    })
}

// Helper to convert interface structs to params
func convertInterfacesToParams(batch []interface{}) []map[string]interface{} {
    result := make([]map[string]interface{}, len(batch))
    
    for i, item := range batch {
        iface := item.(*datamodel.Interface)
        result[i] = map[string]interface{}{
            "name":        iface.Name,
            "packagePath": iface.PackagePath,
            "docComment":  iface.DocComment,
            "location": map[string]interface{}{
                "fileName": iface.Location.Filename,
                "line":     iface.Location.Line,
            },
        }
    }
    
    return result
}
```

**Acceptance Criteria:**
- Batch operations are implemented for all Neo4j operations
- Performance is improved for large datasets
- Batch size is configurable
- Error handling provides context on which batch failed

## Testing

### Task 15: Add Unit Tests for Core Components

Create unit tests for key components:

```go
// Example test for interface_analyzer.go in internal/analyzer/ast/interface_analyzer_test.go:
package ast

import (
    "go/token"
    "testing"

    "github.com/namikmesic/go-mcp/internal/datamodel"
    "golang.org/x/tools/go/packages"
    "golang.org/x/tools/go/packages/packagestest"
)

func TestAnalyzeInterfaces(t *testing.T) {
    // Test data
    modules := []packagestest.Module{
        {
            Name: "example.com/testpkg",
            Files: map[string]interface{}{
                "interfaces.go": `
                package testpkg

                // TestInterface is a test interface
                type TestInterface interface {
                    // TestMethod does something
                    TestMethod(arg string) error
                    
                    // AnotherMethod does something else
                    AnotherMethod() (int, bool)
                }
                
                // EmptyInterface is empty
                type EmptyInterface interface {}
                `,
            },
        },
    }
    
    // Create a temporary environment with the test modules
    exported := packagestest.Export(t, packagestest.Modules, modules)
    defer exported.Cleanup()
    
    // Load the test packages
    config := packages.Config{
        Mode: packages.NeedName |
              packages.NeedFiles |
              packages.NeedCompiledGoFiles |
              packages.NeedImports |
              packages.NeedTypes |
              packages.NeedSyntax |
              packages.NeedTypesInfo,
        Fset: token.NewFileSet(),
        Dir:  exported.Config.Dir,
    }
    
    pkgs, err := packages.Load(&config, "example.com/testpkg")
    if err != nil {
        t.Fatalf("Failed to load packages: %v", err)
    }
    
    // Run the analyzer
    analyzer := NewASTInterfaceAnalyzer()
    interfaces, err := analyzer.AnalyzeInterfaces(pkgs)
    
    // Check results
    if err != nil {
        t.Fatalf("AnalyzeInterfaces failed: %v", err)
    }
    
    if len(interfaces) != 2 {
        t.Errorf("Expected 2 interfaces, got %d", len(interfaces))
    }
    
    // Check TestInterface
    testIfKey := "example.com/testpkg.TestInterface"
    testIf, found := interfaces[testIfKey]
    if !found {
        t.Fatalf("TestInterface not found in results")
    }
    
    if testIf.Name != "TestInterface" {
        t.Errorf("Expected name TestInterface, got %s", testIf.Name)
    }
    
    if testIf.DocComment != "TestInterface is a test interface" {
        t.Errorf("Unexpected doc comment: %s", testIf.DocComment)
    }
    
    if len(testIf.Methods) != 2 {
        t.Errorf("Expected 2 methods, got %d", len(testIf.Methods))
    }
    
    // Check EmptyInterface
    emptyIfKey := "example.com/testpkg.EmptyInterface"
    emptyIf, found := interfaces[emptyIfKey]
    if !found {
        t.Fatalf("EmptyInterface not found in results")
    }
    
    if len(emptyIf.Methods) != 0 {
        t.Errorf("Expected 0 methods for EmptyInterface, got %d", len(emptyIf.Methods))
    }
}
```

**Acceptance Criteria:**
- Unit tests are added for each key component
- Tests use mocks or test data instead of live dependencies
- Tests cover both success and error cases
- Tests are documented with clear assertions

### Task 16: Add Integration Tests

Create integration tests for the end-to-end analysis:

```go
// Example integration test in internal/service/service_test.go:
package service_test

import (
    "path/filepath"
    "testing"

    "github.com/namikmesic/go-mcp/internal/analyzer/ast"
    "github.com/namikmesic/go-mcp/internal/analyzer/ssa"
    "github.com/namikmesic/go-mcp/internal/analyzer/typesystem"
    "github.com/namikmesic/go-mcp/internal/loader"
    "github.com/namikmesic/go-mcp/internal/service"
)

func TestAnalyzeProject(t *testing.T) {
    // Skip if not running in integration test mode
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    // Use the examples directory as test data
    examplesDir, err := filepath.Abs("../../examples")
    if err != nil {
        t.Fatalf("Failed to get absolute path to examples: %v", err)
    }
    
    // Create all dependencies
    pkgLoader := loader.NewGoPackagesLoader()
    ifAnalyzer := ast.NewASTInterfaceAnalyzer()
    implFinder := typesystem.NewTypeBasedImplementationFinder()
    callAnalyzer := ssa.NewSSACallGraphAnalyzer()
    
    // Create the service
    analysisService := service.NewAnalysisService(
        pkgLoader,
        ifAnalyzer,
        implFinder,
        callAnalyzer,
    )
    
    // Run analysis
    result, err := analysisService.AnalyzeProject(examplesDir)
    if err != nil {
        t.Fatalf("AnalyzeProject failed: %v", err)
    }
    
    // Verify result contains expected data
    if result == nil {
        t.Fatal("Result is nil")
    }
    
    if len(result.Packages) == 0 {
        t.Error("No packages found in result")
    }
    
    // Verify demo package was analyzed
    var demoPackage *datamodel.PackageAnalysis
    for _, pkg := range result.Packages {
        if pkg.Name == "demo" {
            demoPackage = pkg
            break
        }
    }
    
    if demoPackage == nil {
        t.Fatal("Demo package not found in results")
    }
    
    // Verify interfaces in demo package
    if len(demoPackage.Interfaces) == 0 {
        t.Error("No interfaces found in demo package")
    }
    
    // Check for specific interfaces
    animalFound := false
    vehicleFound := false
    
    for _, iface := range demoPackage.Interfaces {
        if iface.Name == "Animal" {
            animalFound = true
            
            if len(iface.Methods) != 3 {
                t.Errorf("Expected Animal to have 3 methods, got %d", len(iface.Methods))
            }
        }
        
        if iface.Name == "Vehicle" {
            vehicleFound = true
            
            if len(iface.Methods) != 4 {
                t.Errorf("Expected Vehicle to have 4 methods, got %d", len(iface.Methods))
            }
        }
    }
    
    if !animalFound {
        t.Error("Animal interface not found")
    }
    
    if !vehicleFound {
        t.Error("Vehicle interface not found")
    }
}
```

**Acceptance Criteria:**
- Integration tests verify the end-to-end functionality
- Tests use real project examples as test data
- Tests verify key aspects of the analysis results
- Tests can be skipped in short mode for CI/CD

### Task 17: Add Benchmarks for Performance Testing

Create benchmarks to measure performance of critical paths:

```go
// Example benchmark in internal/analyzer/ast/interface_analyzer_bench_test.go:
package ast

import (
    "testing"

    "github.com/namikmesic/go-mcp/internal/loader"
)

func BenchmarkAnalyzeInterfaces(b *testing.B) {
    // Skip setup time from benchmark
    pkgLoader := loader.NewGoPackagesLoader()
    pkgs, err := pkgLoader.Load("../../examples")
    if err != nil {
        b.Fatalf("Failed to load packages: %v", err)
    }
    
    analyzer := NewASTInterfaceAnalyzer()
    
    // Reset timer before the actual benchmark
    b.ResetTimer()
    
    // Run the benchmark
    for i := 0; i < b.N; i++ {
        interfaces, err := analyzer.AnalyzeInterfaces(pkgs)
        if err != nil {
            b.Fatalf("AnalyzeInterfaces failed: %v", err)
        }
        
        if len(interfaces) == 0 {
            b.Fatal("No interfaces found")
        }
    }
}

// Additional benchmarks for other components
func BenchmarkFindImplementations(b *testing.B) {
    // Similar setup...
}
```

**Acceptance Criteria:**
- Benchmarks are added for performance-critical components
- Benchmarks use realistic input data
- Benchmarks properly exclude setup time
- Benchmarks are documented with expected performance targets

## Performance Optimization

### Task 18: Profile and Optimize Performance Bottlenecks

Add profiling to identify and address performance bottlenecks:

```go
// Add to main.go:
var (
    cpuprofile = flag.String("cpuprofile", "", "Write cpu profile to file")
    memprofile = flag.String("memprofile", "", "Write memory profile to file")
)

// Update main() function:
func main() {
    flag.Parse()
    
    // CPU profiling
    if *cpuprofile != "" {
        f, err := os.Create(*cpuprofile)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Could not create CPU profile: %v\n", err)
            os.Exit(1)
        }
        defer f.Close()
        
        if err := pprof.StartCPUProfile(f); err != nil {
            fmt.Fprintf(os.Stderr, "Could not start CPU profile: %v\n", err)
            os.Exit(1)
        }
        defer pprof.StopCPUProfile()
    }
    
    // ... existing main logic ...
    
    // Memory profiling
    if *memprofile != "" {
        f, err := os.Create(*memprofile)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Could not create memory profile: %v\n", err)
        }
        defer f.Close()
        
        runtime.GC() // Get up-to-date statistics
        if err := pprof.WriteHeapProfile(f); err != nil {
            fmt.Fprintf(os.Stderr, "Could not write memory profile: %v\n", err)
        }
    }
}
```

**Acceptance Criteria:**
- Profiling capabilities are added to the CLI
- Documentation explains how to use profiling
- Common bottlenecks are identified and optimized
- Performance improvements are documented

### Task 19: Implement Parallel Processing

Add parallel processing to improve performance:

```go
// Update AnalyzeInterfaces in interface_analyzer.go:
func (a *ASTInterfaceAnalyzer) AnalyzeInterfaces(pkgs []*packages.Package) (map[string]*datamodel.Interface, error) {
    interfaces := make(map[string]*datamodel.Interface)
    var mutex sync.Mutex // Protect map access
    
    // Process packages in parallel
    var wg sync.WaitGroup
    errChan := make(chan error, len(pkgs))
    
    for _, pkg := range pkgs {
        if !a.isPackageAnalyzable(pkg) {
            continue
        }
        
        wg.Add(1)
        go func(pkg *packages.Package) {
            defer wg.Done()
            
            pkgInterfaces, err := a.analyzePackage(pkg)
            if err != nil {
                errChan <- fmt.Errorf("error analyzing package %s: %w", pkg.PkgPath, err)
                return
            }
            
            // Safely add to the shared map
            mutex.Lock()
            for k, v := range pkgInterfaces {
                if _, exists := interfaces[k]; !exists {
                    interfaces[k] = v
                } else {
                    log.Printf("Warning: Duplicate interface definition encountered for %s. Keeping first.", k)
                }
            }
            mutex.Unlock()
        }(pkg)
    }
    
    // Wait for all goroutines to finish
    wg.Wait()
    close(errChan)
    
    // Check for errors
    if len(errChan) > 0 {
        // Collect all errors
        var errMsgs []string
        for err := range errChan {
            errMsgs = append(errMsgs, err.Error())
        }
        return interfaces, fmt.Errorf("errors analyzing interfaces: %s", strings.Join(errMsgs, "; "))
    }
    
    return interfaces, nil
}

// New method to analyze a single package
func (a *ASTInterfaceAnalyzer) analyzePackage(pkg *packages.Package) (map[string]*datamodel.Interface, error) {
    interfaces := make(map[string]*datamodel.Interface)
    
    // ... existing analysis logic for a single package ...
    
    return interfaces, nil
}
```

**Acceptance Criteria:**
- Concurrent processing is implemented for independent operations
- Thread-safety is ensured for shared data structures
- Error handling works correctly in concurrent code
- Performance is improved for multi-core systems

### Task 20: Add Caching for Intermediate Results

Implement caching for intermediate analysis results:

```go
// Update AnalysisService in service.go:
type AnalysisService struct {
    loader               loader.Loader
    interfaceAnalyzer    analyzer.InterfaceAnalyzer
    implementationFinder analyzer.ImplementationFinder
    callGraphAnalyzer    analyzer.CallGraphAnalyzer
    
    // Cache for analyzed projects
    cache map[string]*datamodel.ProjectAnalysis
    mutex sync.RWMutex
}

// Update NewAnalysisService:
func NewAnalysisService(
    l loader.Loader,
    ia analyzer.InterfaceAnalyzer,
    idf analyzer.ImplementationFinder,
    cga analyzer.CallGraphAnalyzer,
) *AnalysisService {
    // ... existing validation ...
    
    return &AnalysisService{
        loader:               l,
        interfaceAnalyzer:    ia,
        implementationFinder: idf,
        callGraphAnalyzer:    cga,
        cache:                make(map[string]*datamodel.ProjectAnalysis),
    }
}

// Update AnalyzeProject:
func (s *AnalysisService) AnalyzeProject(path string) (*datamodel.ProjectAnalysis, error) {
    // Check cache first
    absPath, err := filepath.Abs(path)
    if err == nil {
        s.mutex.RLock()
        if cachedResult, ok := s.cache[absPath]; ok {
            s.mutex.RUnlock()
            log.Printf("Using cached analysis results for %s", path)
            return cachedResult, nil
        }
        s.mutex.RUnlock()
    }
    
    // ... existing analysis logic ...
    
    // Store in cache
    if err == nil && absPath != "" {
        s.mutex.Lock()
        s.cache[absPath] = projectAnalysis
        s.mutex.Unlock()
    }
    
    return projectAnalysis, nil
}

// Add a method to clear cache:
func (s *AnalysisService) ClearCache() {
    s.mutex.Lock()
    s.cache = make(map[string]*datamodel.ProjectAnalysis)
    s.mutex.Unlock()
}
```

**Acceptance Criteria:**
- Caching is implemented for project analysis results
- Cache is properly invalidated when needed
- Thread-safety is ensured for the cache
- Performance is improved for repeated analyses of the same project

## Documentation

### Task 21: Add Godoc Comments to All Exported Types and Functions

Improve documentation with comprehensive godoc comments:

```go
// Example improvements to interface_analyzer.go:

// ASTInterfaceAnalyzer implements InterfaceAnalyzer using AST traversal to identify
// and extract interface definitions from Go packages.
type ASTInterfaceAnalyzer struct{}

// NewASTInterfaceAnalyzer creates a new instance of ASTInterfaceAnalyzer.
// This analyzer uses Go's AST (Abstract Syntax Tree) to find interface
// definitions in Go source code.
func NewASTInterfaceAnalyzer() *ASTInterfaceAnalyzer {
    return &ASTInterfaceAnalyzer{}
}

// AnalyzeInterfaces scans the provided packages and extracts interface definitions.
// It returns a map where keys are fully qualified interface names (packagePath.interfaceName)
// and values are the corresponding Interface objects with detailed information.
//
// The analyzer extracts:
// - Interface names and locations
// - Documentation comments
// - Method signatures, parameters, and return types
// - Embedded interfaces
//
// If a package lacks required information (e.g., syntax tree, type info),
// it will be skipped with a warning.
func (a *ASTInterfaceAnalyzer) AnalyzeInterfaces(pkgs []*packages.Package) (map[string]*datamodel.Interface, error) {
    // ... implementation ...
}
```

**Acceptance Criteria:**
- All exported types and functions have godoc comments
- Comments follow the standard Go style (starting with the name of the entity)
- Comments explain purpose, behavior, and any special considerations
- Examples are included where appropriate

### Task 22: Create Project README and Documentation

Create a comprehensive README and documentation:

```markdown
# Go Module Call Path Analyzer (go-mcp)

Go-MCP is a static analysis tool that extracts interface definitions, implementations, and call paths from Go projects.

## Features

- Extracts detailed information about interfaces (methods, documentation, etc.)
- Finds concrete implementations of interfaces
- Analyzes call sites and call types
- Displays results in JSON format
- Optionally stores analysis in Neo4j graph database

## Installation

```bash
go install github.com/namikmesic/go-mcp/cmd/go-mcp@latest
```

## Usage

```bash
# Analyze current directory
go-mcp .

# Analyze with recursive pattern
go-mcp ./...

# Analyze specific package
go-mcp github.com/example/package

# Generate CPU profile
go-mcp -cpuprofile=cpu.prof .

# Output summary format
go-mcp -format=summary .

# Skip call analysis for faster results
go-mcp -skip-calls .

# Store results in Neo4j
go-mcp -neo4j-uri=bolt://localhost:7687 -neo4j-user=neo4j -neo4j-pass=password .
```

## Output Format

The tool produces a JSON structure with the following key components:

- **Project information**: Module path and directory
- **Packages**: List of analyzed packages
  - **Interfaces**: Interface definitions with methods, documentation, etc.
  - **Implementations**: Concrete types implementing interfaces
  - **Call sites**: Function/method call information

## Architecture

Go-MCP uses a modular architecture with the following components:

- **Loader**: Loads Go packages using go/packages
- **Interface Analyzer**: Extracts interface definitions using AST
- **Implementation Finder**: Finds implementations using type system
- **Call Graph Analyzer**: Analyzes calls using SSA

## Development

### Prerequisites

- Go 1.21+ 
- For storing results: Neo4j 4.0+

### Building from source

```bash
git clone https://github.com/namikmesic/go-mcp.git
cd go-mcp
go build ./cmd/go-mcp
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

[MIT License](LICENSE)
```

**Acceptance Criteria:**
- README provides comprehensive information about the tool
- Installation and usage instructions are clear
- Output format is documented
- Architecture and components are explained
- Contributing guidelines are included

## Error Handling

### Task 23: Implement Consistent Error Handling Strategy

Improve error handling throughout the codebase:

```go
// Add to internal/errors/errors.go:
package errors

import (
    "fmt"
)

// Common error types
var (
    ErrPackageLoad       = NewErrorType("package load error")
    ErrInterfaceAnalysis = NewErrorType("interface analysis error")
    ErrCallAnalysis      = NewErrorType("call analysis error")
    ErrImplementation    = NewErrorType("implementation finder error")
    ErrDataModel         = NewErrorType("data model error")
    ErrNeo4j             = NewErrorType("Neo4j error")
)

// ErrorType represents a category of errors
type ErrorType struct {
    category string
}

// NewErrorType creates a new error type with the given category
func NewErrorType(category string) *ErrorType {
    return &ErrorType{category: category}
}

// Error creates a new error of this type with the given message
func (et *ErrorType) Error(format string, args ...interface{}) error {
    return &TypedError{
        errorType: et,
        message:   fmt.Sprintf(format, args...),
    }
}

// Wrap creates a new error of this type that wraps an existing error
func (et *ErrorType) Wrap(err error, format string, args ...interface{}) error {
    return &TypedError{
        errorType: et,
        message:   fmt.Sprintf(format, args...),
        cause:     err,
    }
}

// TypedError is an error with a specific type and optional cause
type TypedError struct {
    errorType *ErrorType
    message   string
    cause     error
}

// Error implements the error interface
func (e *TypedError) Error() string {
    if e.cause != nil {
        return fmt.Sprintf("%s: %s: %v", e.errorType.category, e.message, e.cause)
    }
    return fmt.Sprintf("%s: %s", e.errorType.category, e.message)
}

// Unwrap returns the cause of this error for use with errors.Is and errors.As
func (e *TypedError) Unwrap() error {
    return e.cause
}

// Is reports whether this error matches the target error
func (e *TypedError) Is(target error) bool {
    t, ok := target.(*TypedError)
    if !ok {
        return false
    }
    return e.errorType == t.errorType
}
```

**Acceptance Criteria:**
- Consistent error handling strategy is implemented
- Error types categorize different failure modes
- Errors provide context and can wrap other errors
- Errors are compatible with Go's error handling patterns

### Task 24: Add Context to Error Messages

Improve error messages to provide more context:

```go
// Example in AnalyzeInterfaces:
func (a *ASTInterfaceAnalyzer) AnalyzeInterfaces(pkgs []*packages.Package) (map[string]*datamodel.Interface, error) {
    if len(pkgs) == 0 {
        return nil, errors.ErrInterfaceAnalysis.Error("no packages provided for analysis")
    }
    
    // ... existing code ...
    
    for _, pkg := range pkgs {
        if !a.isPackageAnalyzable(pkg) {
            continue
        }
        
        pkgInterfaces, err := a.analyzePackage(pkg)
        if err != nil {
            return interfaces, errors.ErrInterfaceAnalysis.Wrap(err, 
                "failed to analyze package %s", pkg.PkgPath)
        }
        
        // ... existing code ...
    }
    
    // ... existing code ...
}

// Example in implementation_finder.go:
func (f *TypeBasedImplementationFinder) FindImplementations(
    pkgs []*packages.Package,
    interfaces map[string]*datamodel.Interface,
    fset *token.FileSet,
) error {
    if fset == nil {
        return errors.ErrImplementation.Error(
            "no FileSet provided to FindImplementations, location finding will be impaired")
    }
    
    if len(pkgs) == 0 {
        return errors.ErrImplementation.Error("no packages provided for implementation finding")
    }
    
    if len(interfaces) == 0 {
        return errors.ErrImplementation.Error("no interfaces provided for implementation finding")
    }
    
    // ... existing code ...
}
```

**Acceptance Criteria:**
- Error messages include specific context about what failed
- Errors are wrapped to maintain the error chain
- Input validation adds helpful error messages
- Error handling code is consistent throughout the codebase

## Logging

### Task 25: Implement Structured Logging

Replace basic logging with structured logging:

```go
// Add to internal/log/log.go:
package log

import (
    "io"
    "log"
    "os"
)

// Level represents a logging level
type Level int

// Available logging levels
const (
    DebugLevel Level = iota
    InfoLevel
    WarnLevel
    ErrorLevel
)

// Logger provides structured logging capabilities
type Logger struct {
    logger *log.Logger
    level  Level
}

// Default logger instance
var defaultLogger = New(os.Stderr, InfoLevel)

// New creates a new logger with the specified output and level
func New(out io.Writer, level Level) *Logger {
    return &Logger{
        logger: log.New(out, "", log.LstdFlags),
        level:  level,
    }
}

// SetLevel changes the logging level
func (l *Logger) SetLevel(level Level) {
    l.level = level
}

// log formats and outputs a log message if the level is sufficient
func (l *Logger) log(level Level, format string, args ...interface{}) {
    if level < l.level {
        return
    }
    
    // Get level prefix
    var prefix string
    switch level {
    case DebugLevel:
        prefix = "DEBUG: "
    case InfoLevel:
        prefix = "INFO: "
    case WarnLevel:
        prefix = "WARNING: "
    case ErrorLevel:
        prefix = "ERROR: "
    }
    
    l.logger.Printf(prefix+format, args...)
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
    l.log(DebugLevel, format, args...)
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
    l.log(InfoLevel, format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
    l.log(WarnLevel, format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
    l.log(ErrorLevel, format, args...)
}

// Global functions that use the default logger

// SetDefaultLevel sets the level of the default logger
func SetDefaultLevel(level Level) {
    defaultLogger.SetLevel(level)
}

// Debug logs a debug message using the default logger
func Debug(format string, args ...interface{}) {
    defaultLogger.Debug(format, args...)
}

// Info logs an info message using the default logger
func Info(format string, args ...interface{}) {
    defaultLogger.Info(format, args...)
}

// Warn logs a warning message using the default logger
func Warn(format string, args ...interface{}) {
    defaultLogger.Warn(format, args...)
}

// Error logs an error message using the default logger
func Error(format string, args ...interface{}) {
    defaultLogger.Error(format, args...)
}
```

**Acceptance Criteria:**
- Structured logging is implemented with levels
- Log level is configurable
- Logging is used consistently throughout the codebase
- Log messages include appropriate context and level

### Task 26: Add Configurable Logging Verbosity

Make logging verbosity configurable via command-line flags:

```go
// Update main.go:
var (
    logLevel = flag.String("log-level", "info", "Logging level (debug, info, warn, error)")
)

// In main() function:
// Set up logging level
switch strings.ToLower(*logLevel) {
case "debug":
    log.SetDefaultLevel(log.DebugLevel)
case "info":
    log.SetDefaultLevel(log.InfoLevel)
case "warn":
    log.SetDefaultLevel(log.WarnLevel)
case "error":
    log.SetDefaultLevel(log.ErrorLevel)
default:
    fmt.Fprintf(os.Stderr, "Unknown log level: %s, using 'info'\n", *logLevel)
    log.SetDefaultLevel(log.InfoLevel)
}
```

**Acceptance Criteria:**
- Logging verbosity is configurable via command-line flag
- Different levels of messages are shown based on configuration
- Default level is appropriate for normal usage
- Documentation explains the logging levels

### Task 27: Add Proper .gitignore File

Create a comprehensive .gitignore file:

```
# Binaries for programs and plugins
*.exe
*.exe~
*.dll
*.so
*.dylib
go-mcp

# Test binary, built with `go test -c`
*.test

# Output of the go coverage tool
*.out
*.prof

# Dependency directories
vendor/

# IDE-specific files
.idea/
.vscode/
*.swp
*.swo

# OS-specific files
.DS_Store
Thumbs.db

# Log files
*.log

# Generated files
/dist/
/tmp/

# Neo4j data directory if used locally
/data/neo4j/
```

**Acceptance Criteria:**
- .gitignore is created with appropriate entries
- Common build artifacts are ignored
- IDE and OS-specific files are ignored
- No binary files or build artifacts are committed

### Task 28: Add Go Linting Configuration

Add golangci-lint configuration:

```yaml
# .golangci.yml
linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - typecheck
    - unused
    - gosec
    - prealloc
    - misspell
    - gofmt
    - goimports
    - unconvert
    - unparam
    - revive
    - nakedret
    - godot

linters-settings:
  errcheck:
    check-type-assertions: true
  govet:
    check-shadowing: true
  revive:
    rules:
      - name: exported
        arguments:
          - 'disableStutteringCheck'

issues:
  exclude-use-default: false
  max-issues-per-linter: 0
  max-same-issues: 0
  exclude-rules:
    - path: _test\.go
      linters:
        - errcheck
        - gosec

run:
  timeout: 5m
```

**Acceptance Criteria:**
- golangci-lint configuration is created
- Appropriate linters are enabled
- Configuration balances strictness with practicality
- Documentation explains how to run linting

### Task 29: Add GitHub Actions for CI/CD

Create GitHub Actions workflow for continuous integration:

```yaml
# .github/workflows/go.yml
name: Go

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  build:
    name: Build and Test
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'

    - name: Verify dependencies
      run: go mod verify

    - name: Build
      run: go build -v ./...

    - name: Test
      run: go test -v ./...

  lint:
    name: Lint
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'

    - name: golangci-lint
      uses: golangci/golangci-lint-action@v3
      with:
        version: latest

  integration:
    name: Integration Tests
    runs-on: ubuntu-latest
    needs: [build]
    steps:
    - uses: actions/checkout@v3

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'

    - name: Integration Tests
      run: go test -v -tags=integration ./...
```

**Acceptance Criteria:**
- GitHub Actions workflow is configured for CI/CD
- Workflow includes building, testing, and linting
- Integration tests are run separately
- Workflow runs on push to main and for pull requests

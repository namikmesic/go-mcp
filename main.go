package main

import (
	"fmt"
	"go/ast"

	// "go/parser" // We primarily use go/packages now
	"go/token"
	"go/types"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"         // SSA representation
	"golang.org/x/tools/go/ssa/ssautil" // SSA utility functions
)

// MethodInfo remains the same
type MethodInfo struct {
	Name         string
	Parameters   []ParameterInfo
	ReturnTypes  []string
	IsPointer    bool
	DocComment   string
	LineNumber   int
	ColumnNumber int
	Signature    string
}

// ParameterInfo remains the same
type ParameterInfo struct {
	Name      string
	Type      string
	IsPointer bool
}

// InterfaceInfo remains the same
type InterfaceInfo struct {
	Name            string
	Methods         []MethodInfo
	File            string
	LineNumber      int
	ColumnNumber    int
	DocComment      string
	Package         string
	Embeds          []string
	Implementations []Implementation
	TypeInfo        *types.Interface
}

// Implementation remains the same
type Implementation struct {
	TypeName     string
	PackagePath  string
	PackageName  string
	IsPointer    bool
	File         string
	LineNumber   int
	ColumnNumber int
}

// CallInfo stores information about a single call site
type CallInfo struct {
	CallerFunc string // Name of the function/method containing the call
	CalleeDesc string // Description of the called function/method/interface method
	CallType   string // Static, Interface, Go, Defer
	Location   string // File:line:column of the call site
}

// PackageInfo updated to include SSA package and calls
type PackageInfo struct {
	Name          string
	Path          string
	Files         []string
	Imports       []string
	Interfaces    []InterfaceInfo
	Module        *packages.Module
	EmbedFiles    []string
	EmbedPatterns []string
	SsaPackage    *ssa.Package      // Store the built SSA package
	Calls         []CallInfo        // Store calls found within this package
	PkgDef        *packages.Package // Keep the original package definition for context
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <path-to-go-files>")
		os.Exit(1)
	}

	targetPath := os.Args[1] // Renamed for clarity
	pkgsInfo, err := analyzePackages(targetPath)
	if err != nil {
		log.Fatalf("Error analyzing packages: %v", err)
	}

	// Print detailed information about all packages
	fmt.Println("===== DETAILED PACKAGE ANALYSIS =====")
	fmt.Printf("Total packages analyzed: %d\\n\\n", len(pkgsInfo))

	// Print package information
	for i, pkgInfo := range pkgsInfo {
		fmt.Printf("PACKAGE [%d/%d]: %s (%s)\\n", i+1, len(pkgsInfo), pkgInfo.Name, pkgInfo.Path)
		fmt.Printf("----------------------------------------\\n")

		// Print module information
		fmt.Printf("Module: ")
		if pkgInfo.Module != nil {
			fmt.Printf("%s\\n", pkgInfo.Module.Path)
			fmt.Printf("  Version: %s\\n", pkgInfo.Module.Version)
			fmt.Printf("  Directory: %s\\n", pkgInfo.Module.Dir)
			fmt.Printf("  Is Main Module: %v\\n", pkgInfo.Module.Main)
			fmt.Printf("  go.mod: %s\\n", pkgInfo.Module.GoMod)
		} else {
			fmt.Printf("[None/Standard Library]\\n")
		}

		// Print file information
		fmt.Printf("Source Files: %d\\n", len(pkgInfo.Files))
		fmt.Println("  File list:")
		if len(pkgInfo.Files) > 0 {
			for _, file := range pkgInfo.Files {
				fmt.Printf("    - %s\\n", file)
			}
		} else {
			fmt.Println("    [No files]")
		}

		// Print imports
		fmt.Printf("Imports: %d\\n", len(pkgInfo.Imports))
		fmt.Println("  Import list:")
		if len(pkgInfo.Imports) > 0 {
			for _, imp := range pkgInfo.Imports {
				fmt.Printf("    - %s\\n", imp)
			}
		} else {
			fmt.Println("    [No imports]")
		}

		// Print embed info
		fmt.Printf("Embedded files: %d\\n", len(pkgInfo.EmbedFiles))
		fmt.Println("  Embed file list:")
		if len(pkgInfo.EmbedFiles) > 0 {
			for _, file := range pkgInfo.EmbedFiles {
				fmt.Printf("    - %s\\n", file)
			}
		} else {
			fmt.Println("    [No embedded files]")
		}

		fmt.Printf("Embed patterns: %d\\n", len(pkgInfo.EmbedPatterns))
		fmt.Println("  Pattern list:")
		if len(pkgInfo.EmbedPatterns) > 0 {
			for _, pattern := range pkgInfo.EmbedPatterns {
				fmt.Printf("    - %s\\n", pattern)
			}
		} else {
			fmt.Println("    [No embed patterns]")
		}

		// Print interface count
		fmt.Printf("Interfaces: %d\\n", len(pkgInfo.Interfaces))

		// Print detailed interface information
		fmt.Println("\\n  INTERFACE DETAILS:")
		if len(pkgInfo.Interfaces) > 0 {
			for j, iface := range pkgInfo.Interfaces {
				fmt.Printf("  [%d/%d] Interface: %s\\n", j+1, len(pkgInfo.Interfaces), iface.Name)
				fmt.Printf("    Location: %s:%d:%d\\n", iface.File, iface.LineNumber, iface.ColumnNumber)

				fmt.Printf("    Documentation: ")
				if iface.DocComment != "" {
					docComment := strings.TrimSpace(iface.DocComment)
					fmt.Printf("%s\\n", docComment)
				} else {
					fmt.Printf("[No documentation]\\n")
				}

				fmt.Printf("    Embedded Interfaces (%d): ", len(iface.Embeds))
				if len(iface.Embeds) > 0 {
					fmt.Printf("%s\\n", strings.Join(iface.Embeds, ", "))
				} else {
					fmt.Printf("[None]\\n")
				}

				fmt.Printf("    Methods (%d):\\n", len(iface.Methods))
				if len(iface.Methods) > 0 {
					for k, method := range iface.Methods {
						fmt.Printf("      [%d] %s\\n", k+1, method.Signature)

						fmt.Printf("        Doc: ")
						if method.DocComment != "" {
							docComment := strings.TrimSpace(method.DocComment)
							fmt.Printf("%s\\n", docComment)
						} else {
							fmt.Printf("[No documentation]\\n")
						}

						fmt.Printf("        Params (%d): ", len(method.Parameters))
						if len(method.Parameters) > 0 {
							var paramStrs []string
							for _, param := range method.Parameters {
								ptrStr := ""
								if param.IsPointer {
									ptrStr = "*"
								}
								paramName := param.Name
								if paramName == "" {
									paramName = "_"
								}
								paramStrs = append(paramStrs, fmt.Sprintf("%s %s%s", paramName, ptrStr, param.Type))
							}
							fmt.Printf("%s\\n", strings.Join(paramStrs, ", "))
						} else {
							fmt.Printf("[No parameters]\\n")
						}

						fmt.Printf("        Returns (%d): ", len(method.ReturnTypes))
						if len(method.ReturnTypes) > 0 {
							fmt.Printf("%s\\n", strings.Join(method.ReturnTypes, ", "))
						} else {
							fmt.Printf("[No return values]\\n")
						}
					}
				} else {
					fmt.Printf("      [No methods defined]\\n")
				}

				fmt.Printf("    Implementations (%d):\\n", len(iface.Implementations))
				if len(iface.Implementations) > 0 {
					for k, impl := range iface.Implementations {
						ptrStr := ""
						if impl.IsPointer {
							ptrStr = "*"
						}
						fmt.Printf("      [%d] %s%s (package %s)\\n", k+1, ptrStr, impl.TypeName, impl.PackageName)
						if impl.File != "" {
							relPath := impl.File
							// Use targetPath for making relative paths consistent
							if rel, err := filepath.Rel(targetPath, impl.File); err == nil && !strings.HasPrefix(rel, "..") {
								relPath = rel
							}
							fmt.Printf("        Location: %s:%d\\n", relPath, impl.LineNumber)
						} else {
							fmt.Printf("        Location: [Unknown]\\n")
						}
					}
				} else {
					fmt.Println("      [No implementations found]")
				}

				if j < len(pkgInfo.Interfaces)-1 {
					fmt.Println() // Add a blank line between interfaces
				}
			}
		} else {
			fmt.Println("    [No interfaces defined]")
		}

		// --- Print Call Graph Info ---
		fmt.Println("\\n  CALL GRAPH DETAILS:")
		fmt.Printf("    Total Calls Found in Package: %d\\n", len(pkgInfo.Calls))
		if len(pkgInfo.Calls) > 0 {
			for k, call := range pkgInfo.Calls {
				relLocation := call.Location
				// Attempt to make location relative
				if pkgInfo.PkgDef != nil && pkgInfo.PkgDef.Fset != nil {
					// This part is tricky without direct access to the FileSet used by SSA
					// We'll keep the absolute path for now, but ideally, map back using pkg.PkgDef.Fset
				}

				fmt.Printf("    [%d] %s\n", k+1, call.CallType)
				fmt.Printf("        Caller: %s\n", call.CallerFunc)
				fmt.Printf("        Callee: %s\n", call.CalleeDesc)
				fmt.Printf("        Location: %s\n", relLocation) // Print original location for now
			}
		} else {
			fmt.Println("    [No calls found in this package's SSA]")
		}
		// --- End Call Graph Info ---

		// Add blank line between packages
		if i < len(pkgsInfo)-1 {
			fmt.Println("\\n")
		}
	}
}

func analyzePackages(path string) ([]PackageInfo, error) {
	// We need NeedTypes, NeedSyntax, NeedTypesInfo for SSA building.
	// The existing mode includes these.
	config := &packages.Config{
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
		Dir:   path,
		Tests: true, // Analyze test files as well
	}

	// Load the packages using go/packages
	initialPkgs, err := packages.Load(config, "./...")
	if err != nil {
		return nil, fmt.Errorf("loading packages: %v", err)
	}
	if packages.PrintErrors(initialPkgs) > 0 {
		// Continue even if there are errors, but log them
		log.Println("Encountered errors during package loading, analysis might be incomplete.")
	}

	// Build SSA for the loaded packages.
	// We build SSA for all packages plus their dependencies.
	prog, ssaPkgs := ssautil.Packages(initialPkgs, ssa.InstantiateGenerics|ssa.SanityCheckFunctions|ssa.BuildSerially) // Add flags for robustness
	if prog == nil {
		return nil, fmt.Errorf("failed to build SSA program")
	}
	prog.Build() // Build the whole SSA program

	// Map ssa.Package back to packages.Package for easier processing
	ssaPackageMap := make(map[*packages.Package]*ssa.Package)
	for i, p := range initialPkgs {
		if i < len(ssaPkgs) && ssaPkgs[i] != nil { // Ensure index is valid and SSA package was built
			ssaPackageMap[p] = ssaPkgs[i]
		}
	}

	// Extract package information
	var result []PackageInfo
	for _, pkg := range initialPkgs {
		// Skip packages that failed to load entirely (though PrintErrors handles most)
		if pkg.Types == nil {
			log.Printf("Skipping package %s due to loading errors (no types)", pkg.ID)
			continue
		}

		pkgInfo := PackageInfo{
			Name:          pkg.Name,
			Path:          pkg.PkgPath,
			Files:         pkg.GoFiles,
			Module:        pkg.Module,
			EmbedFiles:    pkg.EmbedFiles,
			EmbedPatterns: pkg.EmbedPatterns,
			PkgDef:        pkg,                // Store the original package
			SsaPackage:    ssaPackageMap[pkg], // Get corresponding SSA package
			Calls:         []CallInfo{},       // Initialize calls slice
		}

		// Extract imports
		pkgInfo.Imports = []string{} // Initialize
		for _, imp := range pkg.Imports {
			pkgInfo.Imports = append(pkgInfo.Imports, imp.PkgPath) // Use PkgPath for consistency
		}

		// Find interfaces (using existing logic)
		interfaceMap := make(map[string]*InterfaceInfo)
		if pkg.Types != nil { // Only process if types are available
			for _, file := range pkg.Syntax {
				fset := pkg.Fset
				fileName := fset.File(file.Pos()).Name()

				ast.Inspect(file, func(n ast.Node) bool {
					typeSpec, ok := n.(*ast.TypeSpec)
					if !ok || typeSpec.Name == nil { // Check Name not nil
						return true
					}

					interfaceType, ok := typeSpec.Type.(*ast.InterfaceType)
					if !ok {
						return true
					}

					// Use Definition position for potentially better accuracy
					defPos := fset.Position(typeSpec.Name.Pos())

					iface := InterfaceInfo{
						Name:            typeSpec.Name.Name,
						File:            fileName,
						LineNumber:      defPos.Line,
						ColumnNumber:    defPos.Column,
						Package:         pkg.Name,
						Methods:         []MethodInfo{},
						Embeds:          []string{},
						Implementations: []Implementation{},
					}

					if typeSpec.Doc != nil {
						iface.DocComment = typeSpec.Doc.Text()
					}

					// --- Implementation Finding Logic (Requires careful handling with SSA types) ---
					// Find the types.Interface for this interface using TypesInfo
					obj := pkg.TypesInfo.Defs[typeSpec.Name]
					if obj != nil {
						if typeName, ok := obj.(*types.TypeName); ok {
							if typeInterface, ok := typeName.Type().Underlying().(*types.Interface); ok {
								iface.TypeInfo = typeInterface

								// Look for implementations across ALL loaded packages
								for _, otherPkgDef := range initialPkgs {
									if otherPkgDef.Types == nil {
										continue
									} // Skip packages that failed
									scope := otherPkgDef.Types.Scope()
									for _, name := range scope.Names() {
										implObj := scope.Lookup(name)
										if implObj == nil {
											continue
										}

										// Check if it's an exported type name or if it's in the same package
										if !implObj.Exported() && otherPkgDef != pkg {
											continue
										}

										if implTypeName, ok := implObj.(*types.TypeName); ok {
											t := implTypeName.Type()
											// Check value implementation
											if types.Implements(t, typeInterface) {
												addImplementation(&iface, implTypeName, otherPkgDef, false, prog.Fset)
											}
											// Check pointer implementation
											ptrType := types.NewPointer(t)
											if types.Implements(ptrType, typeInterface) {
												addImplementation(&iface, implTypeName, otherPkgDef, true, prog.Fset)
											}
										}
									}
								}
							}
						}
					}
					// --- End Implementation Finding ---

					// Extract methods (existing logic)
					for _, methodField := range interfaceType.Methods.List {
						if len(methodField.Names) == 0 { // Embedded interface
							switch t := methodField.Type.(type) {
							case *ast.Ident:
								iface.Embeds = append(iface.Embeds, t.Name)
							case *ast.SelectorExpr:
								if x, ok := t.X.(*ast.Ident); ok {
									iface.Embeds = append(iface.Embeds, x.Name+"."+t.Sel.Name)
								}
							}
							continue
						}

						// Regular method
						methodName := methodField.Names[0].Name
						methodPos := fset.Position(methodField.Pos())
						methodInfo := MethodInfo{
							Name:         methodName,
							LineNumber:   methodPos.Line,
							ColumnNumber: methodPos.Column,
							Parameters:   []ParameterInfo{},
							ReturnTypes:  []string{},
						}

						if methodField.Doc != nil {
							methodInfo.DocComment = methodField.Doc.Text()
						}

						if funcType, ok := methodField.Type.(*ast.FuncType); ok {
							methodInfo.Signature = getMethodSignature(methodName, funcType)
							// Extract parameters
							if funcType.Params != nil {
								for _, param := range funcType.Params.List {
									paramTypeStr := getTypeString(param.Type)
									isPtr := isPointerType(param.Type)
									if len(param.Names) > 0 {
										for _, name := range param.Names {
											methodInfo.Parameters = append(methodInfo.Parameters, ParameterInfo{Name: name.Name, Type: paramTypeStr, IsPointer: isPtr})
										}
									} else {
										methodInfo.Parameters = append(methodInfo.Parameters, ParameterInfo{Name: "", Type: paramTypeStr, IsPointer: isPtr})
									}
								}
							}
							// Extract return types
							if funcType.Results != nil {
								for _, result := range funcType.Results.List {
									methodInfo.ReturnTypes = append(methodInfo.ReturnTypes, getTypeString(result.Type))
								}
							}
						}
						iface.Methods = append(iface.Methods, methodInfo)
					}

					interfaceMap[iface.Name] = &iface
					return true // Continue inspecting
				})
			}
		}

		// Add interfaces to package info
		pkgInfo.Interfaces = []InterfaceInfo{} // Initialize
		for _, iface := range interfaceMap {
			pkgInfo.Interfaces = append(pkgInfo.Interfaces, *iface)
		}

		// Extract call graph information if SSA package exists
		if pkgInfo.SsaPackage != nil {
			pkgInfo.Calls = extractCallsFromSsa(pkgInfo.SsaPackage, prog.Fset)
		}

		result = append(result, pkgInfo)
	}

	return result, nil
}

// Helper to add implementation details, finding file position
func addImplementation(iface *InterfaceInfo, typeName *types.TypeName, pkg *packages.Package, isPointer bool, fset *token.FileSet) {
	implFile := ""
	implLine := 0
	implCol := 0

	// Find the AST node corresponding to the TypeName's definition
	// This requires iterating through the package's syntax trees.
	for _, syntaxFile := range pkg.Syntax {
		ast.Inspect(syntaxFile, func(n ast.Node) bool {
			if spec, ok := n.(*ast.TypeSpec); ok {
				if spec.Name != nil && spec.Name.Name == typeName.Name() {
					// Check if the TypeSpec's definition matches the TypeName object
					if pkg.TypesInfo.Defs[spec.Name] == typeName {
						pos := fset.Position(spec.Name.Pos()) // Use position of the name identifier
						implFile = pos.Filename
						implLine = pos.Line
						implCol = pos.Column
						return false // Stop searching in this subtree
					}
				}
			}
			return true // Continue searching
		})
		if implFile != "" {
			break // Stop searching other files once found
		}
	}

	// Avoid duplicate entries if a type implements via both value and pointer satisfying the interface check
	// A simple check based on name and pointer status might suffice here.
	isDuplicate := false
	for _, existingImpl := range iface.Implementations {
		if existingImpl.TypeName == typeName.Name() && existingImpl.PackagePath == pkg.PkgPath && existingImpl.IsPointer == isPointer {
			isDuplicate = true
			break
		}
	}

	if !isDuplicate {
		iface.Implementations = append(iface.Implementations, Implementation{
			TypeName:     typeName.Name(),
			PackagePath:  pkg.PkgPath,
			PackageName:  pkg.Name,
			IsPointer:    isPointer,
			File:         implFile,
			LineNumber:   implLine,
			ColumnNumber: implCol,
		})
	}
}

// Extracts call information from an SSA package
func extractCallsFromSsa(pkg *ssa.Package, fset *token.FileSet) []CallInfo {
	var calls []CallInfo
	if pkg == nil {
		return calls
	}

	// Use Members to iterate through functions and globals defined in the package
	for _, member := range pkg.Members {
		if fn, ok := member.(*ssa.Function); ok {
			if fn.Blocks == nil {
				continue
			} // Skip functions without basic blocks (e.g., external functions)

			callerName := fn.String() // Get a readable name for the caller

			for _, b := range fn.Blocks {
				if b == nil {
					continue
				} // Defensive check
				for _, instr := range b.Instrs {
					if instr == nil {
						continue
					} // Defensive check

					pos := fset.Position(instr.Pos()) // Get source position
					location := fmt.Sprintf("%s:%d:%d", pos.Filename, pos.Line, pos.Column)
					var callInfo *CallInfo // Use pointer to avoid copying

					switch call := instr.(type) {
					case *ssa.Call:
						common := call.Common()
						if common == nil {
							continue
						}

						// Check if this is an interface method call
						if common.IsInvoke() {
							// Interface method call
							desc := fmt.Sprintf("Interface method %s on %s", common.Method.Name(), common.Value.Type().String())
							callInfo = &CallInfo{
								CallerFunc: callerName,
								CalleeDesc: desc,
								CallType:   "Interface",
								Location:   location,
							}
						} else {
							// Regular function call
							callee := common.StaticCallee() // Static calls have a direct target
							desc := "Unknown Static Callee"
							if callee != nil {
								desc = callee.String()
							} else if common.Value != nil { // Handle calls via function values
								desc = fmt.Sprintf("Dynamic via %s (%s)", common.Value.Name(), common.Value.Type().String())
							}
							callInfo = &CallInfo{
								CallerFunc: callerName,
								CalleeDesc: desc,
								CallType:   "Static",
								Location:   location,
							}
						}
					case *ssa.Go:
						common := call.Common()
						if common == nil {
							continue
						}
						callee := common.StaticCallee()
						desc := "Unknown Go Callee"
						if callee != nil {
							desc = callee.String()
						} else if common.Value != nil {
							desc = fmt.Sprintf("Dynamic via %s (%s)", common.Value.Name(), common.Value.Type().String())
						}
						callInfo = &CallInfo{
							CallerFunc: callerName,
							CalleeDesc: desc,
							CallType:   "Go",
							Location:   location,
						}
					case *ssa.Defer:
						common := call.Common()
						if common == nil {
							continue
						}
						callee := common.StaticCallee()
						desc := "Unknown Defer Callee"
						if callee != nil {
							desc = callee.String()
						} else if common.Value != nil {
							desc = fmt.Sprintf("Dynamic via %s (%s)", common.Value.Name(), common.Value.Type().String())
						}
						callInfo = &CallInfo{
							CallerFunc: callerName,
							CalleeDesc: desc,
							CallType:   "Defer",
							Location:   location,
						}
					}

					if callInfo != nil {
						calls = append(calls, *callInfo)
					}
				}
			}
		}
	}
	return calls
}

// Rest of the helper functions (getTypeString, getArrayLength, etc.) remain unchanged
func getTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + getTypeString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + getTypeString(t.Elt)
		}
		return "[" + getArrayLength(t.Len) + "]" + getTypeString(t.Elt)
	case *ast.MapType:
		return "map[" + getTypeString(t.Key) + "]" + getTypeString(t.Value)
	case *ast.InterfaceType:
		// Handle anonymous interfaces within signatures more explicitly if needed
		if t.Methods == nil || len(t.Methods.List) == 0 {
			return "interface{}"
		}
		return "interface{...}" // Abbreviate complex anonymous interfaces
	case *ast.SelectorExpr:
		// Resolve qualified types
		return getTypeString(t.X) + "." + t.Sel.Name
	case *ast.ChanType:
		dir := ""
		switch t.Dir {
		case ast.SEND:
			dir = "chan<- "
		case ast.RECV:
			dir = "<-chan "
		default:
			dir = "chan "
		}
		return dir + getTypeString(t.Value)
	case *ast.FuncType:
		return "func" + getFuncTypeString(t) // Recursively format func types
	case *ast.StructType:
		return "struct{...}" // Abbreviate anonymous struct types
	case *ast.Ellipsis:
		return "..." + getTypeString(t.Elt)
	default:
		// Fallback for unhandled types
		// Using go/types string representation if available would be more robust
		// but requires mapping ast.Expr back to types.Type which is complex here.
		buf := new(strings.Builder)
		err := ast.Fprint(buf, token.NewFileSet(), expr, ast.NotNilFilter)
		if err == nil {
			return buf.String() // Use AST printer as fallback
		}
		return fmt.Sprintf("UnhandledType<%T>", expr) // Last resort
	}
}

func getArrayLength(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.BasicLit:
		if t.Kind == token.INT {
			return t.Value
		}
	// Handle other cases like constants if needed
	default:
		// Could try to evaluate constant expressions here if necessary
		return "N" // Placeholder for non-literal lengths
	}
	return "N"
}

func getFuncTypeString(ft *ast.FuncType) string {
	var params, results []string

	if ft.Params != nil {
		for _, p := range ft.Params.List {
			pType := getTypeString(p.Type)
			if len(p.Names) > 0 {
				// Collect names sharing the same type
				var names []string
				for _, name := range p.Names {
					names = append(names, name.Name)
				}
				params = append(params, strings.Join(names, ", ")+" "+pType)
			} else {
				params = append(params, pType) // Unnamed parameter
			}
		}
	}

	if ft.Results != nil {
		for _, r := range ft.Results.List {
			rType := getTypeString(r.Type)
			if len(r.Names) > 0 {
				var names []string
				for _, name := range r.Names {
					names = append(names, name.Name)
				}
				results = append(results, strings.Join(names, ", ")+" "+rType)
			} else {
				results = append(results, rType) // Unnamed result
			}
		}
	}

	paramStr := strings.Join(params, ", ")
	resultStr := strings.Join(results, ", ")

	s := fmt.Sprintf("(%s)", paramStr)
	if len(results) > 0 {
		if len(results) == 1 && len(ft.Results.List[0].Names) == 0 {
			// Single unnamed return value
			s += " " + resultStr
		} else {
			// Multiple or named return values
			s += fmt.Sprintf(" (%s)", resultStr)
		}
	}

	return s
}

func isPointerType(expr ast.Expr) bool {
	_, ok := expr.(*ast.StarExpr)
	return ok
}

func getMethodSignature(name string, ft *ast.FuncType) string {
	// Use the improved getFuncTypeString for the signature part
	return name + getFuncTypeString(ft)
}

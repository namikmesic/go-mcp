package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

// MethodInfo represents detailed information about a method
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

// ParameterInfo represents information about a method parameter
type ParameterInfo struct {
	Name      string
	Type      string
	IsPointer bool
}

// InterfaceInfo represents information about a found interface
type InterfaceInfo struct {
	Name            string
	Methods         []MethodInfo
	File            string
	LineNumber      int
	ColumnNumber    int
	DocComment      string
	Package         string
	Embeds          []string         // Embedded interfaces
	Implementations []Implementation // Types implementing this interface
	TypeInfo        *types.Interface // The actual type information
}

// Implementation represents a concrete type that implements an interface
type Implementation struct {
	TypeName     string
	PackagePath  string
	PackageName  string
	IsPointer    bool
	File         string
	LineNumber   int
	ColumnNumber int
}

// PackageInfo represents detailed information about a package
type PackageInfo struct {
	Name          string
	Path          string
	Files         []string
	Imports       []string
	Interfaces    []InterfaceInfo
	Module        *packages.Module // Module information
	EmbedFiles    []string         // Embedded files
	EmbedPatterns []string         // Embed patterns
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <path-to-go-files>")
		os.Exit(1)
	}

	path := os.Args[1]
	pkgs, err := analyzePackages(path)
	if err != nil {
		log.Fatalf("Error analyzing packages: %v", err)
	}

	// Print detailed information about all packages
	fmt.Println("===== DETAILED PACKAGE ANALYSIS =====")
	fmt.Printf("Total packages analyzed: %d\n\n", len(pkgs))

	// Print package information
	for i, pkg := range pkgs {
		fmt.Printf("PACKAGE [%d/%d]: %s (%s)\n", i+1, len(pkgs), pkg.Name, pkg.Path)
		fmt.Printf("----------------------------------------\n")

		// Print module information
		fmt.Printf("Module: ")
		if pkg.Module != nil {
			fmt.Printf("%s\n", pkg.Module.Path)
			fmt.Printf("  Version: %s\n", pkg.Module.Version)
			fmt.Printf("  Directory: %s\n", pkg.Module.Dir)
			fmt.Printf("  Is Main Module: %v\n", pkg.Module.Main)
			fmt.Printf("  go.mod: %s\n", pkg.Module.GoMod)
		} else {
			fmt.Printf("[None/Standard Library]\n")
		}

		// Print file information
		fmt.Printf("Source Files: %d\n", len(pkg.Files))
		fmt.Println("  File list:")
		if len(pkg.Files) > 0 {
			for _, file := range pkg.Files {
				fmt.Printf("    - %s\n", file)
			}
		} else {
			fmt.Println("    [No files]")
		}

		// Print imports
		fmt.Printf("Imports: %d\n", len(pkg.Imports))
		fmt.Println("  Import list:")
		if len(pkg.Imports) > 0 {
			for _, imp := range pkg.Imports {
				fmt.Printf("    - %s\n", imp)
			}
		} else {
			fmt.Println("    [No imports]")
		}

		// Print embed info
		fmt.Printf("Embedded files: %d\n", len(pkg.EmbedFiles))
		fmt.Println("  Embed file list:")
		if len(pkg.EmbedFiles) > 0 {
			for _, file := range pkg.EmbedFiles {
				fmt.Printf("    - %s\n", file)
			}
		} else {
			fmt.Println("    [No embedded files]")
		}

		fmt.Printf("Embed patterns: %d\n", len(pkg.EmbedPatterns))
		fmt.Println("  Pattern list:")
		if len(pkg.EmbedPatterns) > 0 {
			for _, pattern := range pkg.EmbedPatterns {
				fmt.Printf("    - %s\n", pattern)
			}
		} else {
			fmt.Println("    [No embed patterns]")
		}

		// Print interface count
		fmt.Printf("Interfaces: %d\n", len(pkg.Interfaces))

		// Print detailed interface information
		fmt.Println("\n  INTERFACE DETAILS:")
		if len(pkg.Interfaces) > 0 {
			for j, iface := range pkg.Interfaces {
				fmt.Printf("  [%d/%d] Interface: %s\n", j+1, len(pkg.Interfaces), iface.Name)
				fmt.Printf("    Location: %s:%d:%d\n", iface.File, iface.LineNumber, iface.ColumnNumber)

				fmt.Printf("    Documentation: ")
				if iface.DocComment != "" {
					docComment := strings.TrimSpace(iface.DocComment)
					fmt.Printf("%s\n", docComment)
				} else {
					fmt.Printf("[No documentation]\n")
				}

				fmt.Printf("    Embedded Interfaces (%d): ", len(iface.Embeds))
				if len(iface.Embeds) > 0 {
					fmt.Printf("%s\n", strings.Join(iface.Embeds, ", "))
				} else {
					fmt.Printf("[None]\n")
				}

				fmt.Printf("    Methods (%d):\n", len(iface.Methods))
				if len(iface.Methods) > 0 {
					for k, method := range iface.Methods {
						fmt.Printf("      [%d] %s\n", k+1, method.Signature)

						fmt.Printf("        Doc: ")
						if method.DocComment != "" {
							docComment := strings.TrimSpace(method.DocComment)
							fmt.Printf("%s\n", docComment)
						} else {
							fmt.Printf("[No documentation]\n")
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
							fmt.Printf("%s\n", strings.Join(paramStrs, ", "))
						} else {
							fmt.Printf("[No parameters]\n")
						}

						fmt.Printf("        Returns (%d): ", len(method.ReturnTypes))
						if len(method.ReturnTypes) > 0 {
							fmt.Printf("%s\n", strings.Join(method.ReturnTypes, ", "))
						} else {
							fmt.Printf("[No return values]\n")
						}
					}
				} else {
					fmt.Printf("      [No methods defined]\n")
				}

				fmt.Printf("    Implementations (%d):\n", len(iface.Implementations))
				if len(iface.Implementations) > 0 {
					for k, impl := range iface.Implementations {
						ptrStr := ""
						if impl.IsPointer {
							ptrStr = "*"
						}
						fmt.Printf("      [%d] %s%s (package %s)\n", k+1, ptrStr, impl.TypeName, impl.PackageName)
						if impl.File != "" {
							relPath := impl.File
							if rel, err := filepath.Rel(path, impl.File); err == nil && !strings.HasPrefix(rel, "..") {
								relPath = rel
							}
							fmt.Printf("        Location: %s:%d\n", relPath, impl.LineNumber)
						} else {
							fmt.Printf("        Location: [Unknown]\n")
						}
					}
				} else {
					fmt.Println("      [No implementations found]")
				}

				if j < len(pkg.Interfaces)-1 {
					fmt.Println() // Add a blank line between interfaces
				}
			}
		} else {
			fmt.Println("    [No interfaces defined]")
		}

		// Add blank line between packages
		if i < len(pkgs)-1 {
			fmt.Println("\n")
		}
	}
}

func analyzePackages(path string) ([]PackageInfo, error) {
	// Configure the packages loader with ALL load modes for maximum detail
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
		Dir: path,
		// Enable tests to get test files analysis
		Tests: true,
	}

	// Load the packages
	pkgs, err := packages.Load(config, "./...")
	if err != nil {
		return nil, fmt.Errorf("loading packages: %v", err)
	}

	// Extract package information
	var result []PackageInfo
	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			for _, err := range pkg.Errors {
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}
		}

		pkgInfo := PackageInfo{
			Name:          pkg.Name,
			Path:          pkg.PkgPath,
			Files:         pkg.GoFiles,
			Module:        pkg.Module,
			EmbedFiles:    pkg.EmbedFiles,
			EmbedPatterns: pkg.EmbedPatterns,
		}

		// Extract imports
		for imp := range pkg.Imports {
			pkgInfo.Imports = append(pkgInfo.Imports, imp)
		}

		// Find interfaces
		interfaceMap := make(map[string]*InterfaceInfo)
		for _, file := range pkg.Syntax {
			fset := pkg.Fset
			fileName := pkg.Fset.File(file.Pos()).Name()

			// Visit all nodes to find interfaces
			ast.Inspect(file, func(n ast.Node) bool {
				typeSpec, ok := n.(*ast.TypeSpec)
				if !ok {
					return true
				}

				interfaceType, ok := typeSpec.Type.(*ast.InterfaceType)
				if !ok {
					return true
				}

				// Create interface info
				iface := InterfaceInfo{
					Name:            typeSpec.Name.Name,
					File:            fileName,
					LineNumber:      fset.Position(typeSpec.Pos()).Line,
					ColumnNumber:    fset.Position(typeSpec.Pos()).Column,
					Package:         pkg.Name,
					Methods:         []MethodInfo{},     // Initialize as empty slice
					Embeds:          []string{},         // Initialize as empty slice
					Implementations: []Implementation{}, // Initialize as empty slice
				}

				// Get interface documentation
				if typeSpec.Doc != nil {
					iface.DocComment = typeSpec.Doc.Text()
				}

				// Find the types.Interface for this interface
				obj := pkg.TypesInfo.Defs[typeSpec.Name]
				if obj != nil {
					if typeName, ok := obj.(*types.TypeName); ok {
						if typeInterface, ok := typeName.Type().Underlying().(*types.Interface); ok {
							iface.TypeInfo = typeInterface

							// Look for implementations
							for _, otherPkg := range pkgs {
								scope := otherPkg.Types.Scope()
								for _, name := range scope.Names() {
									obj := scope.Lookup(name)
									if obj == nil || !obj.Exported() {
										continue
									}

									if typeName, ok := obj.(*types.TypeName); ok {
										t := typeName.Type()
										if types.Implements(t, typeInterface) {
											implFile := ""
											implLine := 0
											implCol := 0

											// Find position information for the implementation
											for _, f := range otherPkg.Syntax {
												ast.Inspect(f, func(n ast.Node) bool {
													ts, ok := n.(*ast.TypeSpec)
													if !ok || ts.Name == nil || ts.Name.Name != name {
														return true
													}

													implFile = otherPkg.Fset.Position(ts.Pos()).Filename
													implLine = otherPkg.Fset.Position(ts.Pos()).Line
													implCol = otherPkg.Fset.Position(ts.Pos()).Column
													return false
												})

												if implFile != "" {
													break
												}
											}

											iface.Implementations = append(iface.Implementations, Implementation{
												TypeName:     name,
												PackagePath:  otherPkg.PkgPath,
												PackageName:  otherPkg.Name,
												IsPointer:    false,
												File:         implFile,
												LineNumber:   implLine,
												ColumnNumber: implCol,
											})
										}

										ptrType := types.NewPointer(t)
										if types.Implements(ptrType, typeInterface) {
											implFile := ""
											implLine := 0
											implCol := 0

											// Find position information for the implementation
											for _, f := range otherPkg.Syntax {
												ast.Inspect(f, func(n ast.Node) bool {
													ts, ok := n.(*ast.TypeSpec)
													if !ok || ts.Name == nil || ts.Name.Name != name {
														return true
													}

													implFile = otherPkg.Fset.Position(ts.Pos()).Filename
													implLine = otherPkg.Fset.Position(ts.Pos()).Line
													implCol = otherPkg.Fset.Position(ts.Pos()).Column
													return false
												})

												if implFile != "" {
													break
												}
											}

											iface.Implementations = append(iface.Implementations, Implementation{
												TypeName:     name,
												PackagePath:  otherPkg.PkgPath,
												PackageName:  otherPkg.Name,
												IsPointer:    true,
												File:         implFile,
												LineNumber:   implLine,
												ColumnNumber: implCol,
											})
										}
									}
								}
							}
						}
					}
				}

				// Extract methods
				for _, method := range interfaceType.Methods.List {
					// Check if this is an embedded interface
					if len(method.Names) == 0 {
						// This is an embedded interface
						switch t := method.Type.(type) {
						case *ast.Ident:
							iface.Embeds = append(iface.Embeds, t.Name)
						case *ast.SelectorExpr:
							if x, ok := t.X.(*ast.Ident); ok {
								iface.Embeds = append(iface.Embeds, x.Name+"."+t.Sel.Name)
							}
						}
						continue
					}

					methodInfo := MethodInfo{
						Name:         method.Names[0].Name,
						LineNumber:   fset.Position(method.Pos()).Line,
						ColumnNumber: fset.Position(method.Pos()).Column,
						Parameters:   []ParameterInfo{}, // Initialize as empty slice
						ReturnTypes:  []string{},        // Initialize as empty slice
					}

					// Get method documentation
					if method.Doc != nil {
						methodInfo.DocComment = method.Doc.Text()
					}

					// Get method type
					if funcType, ok := method.Type.(*ast.FuncType); ok {
						// Extract method signature
						methodInfo.Signature = getMethodSignature(method.Names[0].Name, funcType)

						// Extract parameters
						if funcType.Params != nil {
							for _, param := range funcType.Params.List {
								paramType := getTypeString(param.Type)
								isPtr := isPointerType(param.Type)

								if len(param.Names) > 0 {
									for _, name := range param.Names {
										paramInfo := ParameterInfo{
											Name:      name.Name,
											Type:      paramType,
											IsPointer: isPtr,
										}
										methodInfo.Parameters = append(methodInfo.Parameters, paramInfo)
									}
								} else {
									paramInfo := ParameterInfo{
										Name:      "",
										Type:      paramType,
										IsPointer: isPtr,
									}
									methodInfo.Parameters = append(methodInfo.Parameters, paramInfo)
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
				return true
			})
		}

		// Add interfaces to package info
		for _, iface := range interfaceMap {
			pkgInfo.Interfaces = append(pkgInfo.Interfaces, *iface)
		}

		result = append(result, pkgInfo)
	}

	return result, nil
}

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
		return "interface{}"
	case *ast.SelectorExpr:
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
		return "func" + getFuncTypeString(t)
	case *ast.StructType:
		return "struct{...}"
	case *ast.Ellipsis:
		return "..." + getTypeString(t.Elt)
	default:
		return fmt.Sprintf("%T", expr)
	}
}

func getArrayLength(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.BasicLit:
		return t.Value
	default:
		return "N"
	}
}

func getFuncTypeString(ft *ast.FuncType) string {
	var params, results []string

	if ft.Params != nil {
		for _, p := range ft.Params.List {
			params = append(params, getTypeString(p.Type))
		}
	}

	if ft.Results != nil {
		for _, r := range ft.Results.List {
			results = append(results, getTypeString(r.Type))
		}
	}

	s := "(" + strings.Join(params, ", ") + ")"
	if len(results) > 0 {
		s += " (" + strings.Join(results, ", ") + ")"
	}

	return s
}

func isPointerType(expr ast.Expr) bool {
	_, ok := expr.(*ast.StarExpr)
	return ok
}

func getMethodSignature(name string, ft *ast.FuncType) string {
	var params, results []string

	if ft.Params != nil {
		for _, p := range ft.Params.List {
			paramType := getTypeString(p.Type)
			if len(p.Names) > 0 {
				for _, name := range p.Names {
					params = append(params, name.Name+" "+paramType)
				}
			} else {
				params = append(params, paramType)
			}
		}
	}

	if ft.Results != nil {
		for _, r := range ft.Results.List {
			resultType := getTypeString(r.Type)
			if len(r.Names) > 0 {
				for _, name := range r.Names {
					results = append(results, name.Name+" "+resultType)
				}
			} else {
				results = append(results, resultType)
			}
		}
	}

	sig := name + "(" + strings.Join(params, ", ") + ")"
	if len(results) == 0 {
		return sig
	} else if len(results) == 1 && len(ft.Results.List[0].Names) == 0 {
		return sig + " " + results[0]
	} else {
		return sig + " (" + strings.Join(results, ", ") + ")"
	}
}

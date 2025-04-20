// analyzer/ast/interface_analyzer.go
package ast

import (
	"go/ast"
	"go/types"

	// "go/types" // Removed as not directly used here, utils handles type strings
	"log"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/namikmesic/go-mcp/internal/analyzer/utils" // Adjusted import path
	"github.com/namikmesic/go-mcp/internal/datamodel"      // Adjusted import path
)

// ASTInterfaceAnalyzer implements InterfaceAnalyzer using AST traversal.
type ASTInterfaceAnalyzer struct{}

func NewASTInterfaceAnalyzer() *ASTInterfaceAnalyzer {
	return &ASTInterfaceAnalyzer{}
}

func (a *ASTInterfaceAnalyzer) AnalyzeInterfaces(pkgs []*packages.Package) (map[string]*datamodel.Interface, error) {
	interfaces := make(map[string]*datamodel.Interface) // Key: packagePath + "." + interfaceName

	for _, pkg := range pkgs {
		// Ensure necessary components are available
		if pkg.Types == nil || pkg.Fset == nil || len(pkg.Syntax) == 0 || pkg.TypesInfo == nil {
			log.Printf("Skipping package %s for interface analysis: missing types, fileset, syntax trees, or types info.", pkg.ID)
			continue // Skip packages without essential info
		}
		fset := pkg.Fset

		for _, file := range pkg.Syntax {
			if file == nil {
				continue // Defensive check
			}
			// fileName := fset.File(file.Pos()).Name() // Keep if needed for logging

			ast.Inspect(file, func(n ast.Node) bool {
				typeSpec, ok := n.(*ast.TypeSpec)
				if !ok || typeSpec.Name == nil {
					return true // Not a type spec or name is nil, continue
				}

				interfaceType, ok := typeSpec.Type.(*ast.InterfaceType)
				if !ok {
					return true // Not an interface type, continue
				}

				// Check if the definition exists in TypesInfo - helps filter out issues
				// Use Defs for type definitions
				obj := pkg.TypesInfo.Defs[typeSpec.Name]
				if obj == nil {
					// It might be a Use if the type is defined elsewhere but used here.
					// We are interested in definitions found within the syntax tree.
					log.Printf("Warning: No type definition object found for %s in package %s using TypesInfo.Defs, skipping.", typeSpec.Name.Name, pkg.PkgPath)
					return true // Skip if type info doesn't know about this type spec as a definition
				}
				// Further check if the object corresponds to an interface type
				if _, ok := obj.Type().Underlying().(*types.Interface); !ok {
					// This TypeSpec is not defining an interface according to type info
					return true
				}

				defPos := fset.Position(typeSpec.Name.Pos())
				iface := &datamodel.Interface{
					Name:            typeSpec.Name.Name,
					PackageName:     pkg.Name,
					PackagePath:     pkg.PkgPath,
					Location:        datamodel.NewLocation(defPos),
					Methods:         []datamodel.Method{},         // Initialize explicitly
					Embeds:          []string{},                   // Initialize explicitly
					Implementations: []datamodel.Implementation{}, // Initialize explicitly
				}

				if typeSpec.Doc != nil {
					iface.DocComment = strings.TrimSpace(typeSpec.Doc.Text())
				}

				// Extract methods and embeds
				if interfaceType.Methods != nil {
					for _, field := range interfaceType.Methods.List {
						if field == nil {
							continue // Defensive check
						}

						// Embedded interface
						if len(field.Names) == 0 && field.Type != nil {
							// Use helper for qualified names, ensure pkg is passed
							embedName := utils.ExprToString(field.Type, pkg)
							if embedName != "" && embedName != "?" { // Avoid adding invalid names
								iface.Embeds = append(iface.Embeds, embedName)
							}
							continue
						}

						// Regular method
						if len(field.Names) > 0 && field.Names[0] != nil && field.Type != nil {
							methodName := field.Names[0].Name
							methodPos := fset.Position(field.Pos()) // Position of the method field itself
							methodInfo := datamodel.Method{
								Name:        methodName,
								Location:    datamodel.NewLocation(methodPos),
								Parameters:  []datamodel.Parameter{}, // Initialize
								ReturnTypes: []string{},              // Initialize
							}

							if field.Doc != nil {
								methodInfo.DocComment = strings.TrimSpace(field.Doc.Text())
							}

							if funcType, ok := field.Type.(*ast.FuncType); ok {
								// Use utility functions for formatting and extraction
								methodInfo.Signature = utils.FormatMethodSignature(methodName, funcType, pkg)
								methodInfo.Parameters = utils.ExtractParameters(funcType, pkg)
								methodInfo.ReturnTypes = utils.ExtractReturnTypes(funcType, pkg)
							} else {
								// Handle cases where method type is not FuncType (e.g., error in code)
								log.Printf("Warning: Method '%s' in interface '%s' (pkg: %s) has non-function type %T", methodName, iface.Name, pkg.PkgPath, field.Type)
								methodInfo.Signature = methodName + "(...) // Analysis Error: Non-FuncType" // Placeholder signature
							}
							iface.Methods = append(iface.Methods, methodInfo)
						}
					}
				}

				// Store using a unique key (package path + name)
				mapKey := pkg.PkgPath + "." + iface.Name
				// Check for duplicates before adding (could happen if file is listed multiple times?)
				if _, exists := interfaces[mapKey]; !exists {
					interfaces[mapKey] = iface
				} else {
					log.Printf("Warning: Duplicate interface definition encountered for %s. Keeping first.", mapKey)
				}
				// Don't return false here, allow inspection to continue for other types in the file.
				// Returning true continues the walk; returning false prunes the walk at this node.
				// We want to find all top-level type specs.
				return true
			})
		}
	}
	return interfaces, nil
}

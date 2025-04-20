// analyzer/typesystem/implementation_finder.go
package typesystem

import (
	"go/ast"
	"go/token"
	"go/types"
	"log"

	"golang.org/x/tools/go/packages"

	"github.com/namikmesic/go-mcp/internal/datamodel" // Adjusted import path
)

// TypeBasedImplementationFinder finds implementations using go/types.
type TypeBasedImplementationFinder struct{}

func NewTypeBasedImplementationFinder() *TypeBasedImplementationFinder {
	return &TypeBasedImplementationFinder{}
}

func (f *TypeBasedImplementationFinder) FindImplementations(
	pkgs []*packages.Package,
	interfaces map[string]*datamodel.Interface, // Key: packagePath + "." + interfaceName
	fset *token.FileSet, // Use the FileSet from SSA/prog for consistency
) error {
	if fset == nil {
		log.Println("Warning: No FileSet provided to FindImplementations. Location finding for implementations will be impaired or potentially inaccurate.")
		// Proceeding without a guaranteed consistent FileSet might lead to incorrect locations.
		// Consider returning an error or using a default fset if absolutely necessary, but locations might not match SSA.
		// fset = token.NewFileSet() // Avoid this unless you understand the implications
	}

	// Build map from types.Interface to our datamodel.Interface for lookup
	typeToInterfaceMap := make(map[*types.Interface]*datamodel.Interface)
	interfaceKeyToTypeMap := make(map[string]*types.Interface) // For reverse lookup if needed

	for key, ifaceData := range interfaces {
		// Find the types.Interface corresponding to our datamodel.Interface
		pkg := findPackage(pkgs, ifaceData.PackagePath)
		if pkg == nil || pkg.Types == nil || pkg.TypesInfo == nil {
			log.Printf("Warning: Could not find loaded package or type info for '%s' while mapping interface '%s'. Skipping implementation checks for this interface.", ifaceData.PackagePath, ifaceData.Name)
			continue
		}
		scope := pkg.Types.Scope()
		if scope == nil {
			log.Printf("Warning: Package scope is nil for '%s', cannot look up interface '%s'.", ifaceData.PackagePath, ifaceData.Name)
			continue
		}
		obj := scope.Lookup(ifaceData.Name)
		if obj == nil {
			log.Printf("Warning: Could not lookup interface '%s' in package '%s' scope.", ifaceData.Name, ifaceData.PackagePath)
			continue
		}

		typeName, ok := obj.(*types.TypeName)
		if !ok {
			log.Printf("Warning: Looked up object '%s' in '%s' is not a TypeName (%T).", ifaceData.Name, ifaceData.PackagePath, obj)
			continue
		}

		typeInterface, ok := typeName.Type().Underlying().(*types.Interface)
		if !ok {
			// This can happen if the name exists but isn't an interface (e.g., type alias)
			log.Printf("Warning: Underlying type of '%s' in '%s' is not *types.Interface (%T).", ifaceData.Name, ifaceData.PackagePath, typeName.Type().Underlying())
			continue
		}

		typeToInterfaceMap[typeInterface] = ifaceData
		interfaceKeyToTypeMap[key] = typeInterface // Store reverse mapping
		// Store the underlying type back in the datamodel if needed (optional)
		ifaceData.UnderlyingType = typeInterface
	}

	log.Printf("Mapped %d interfaces to their types.Interface representations.", len(typeToInterfaceMap))
	if len(typeToInterfaceMap) < len(interfaces) {
		log.Printf("Warning: Mismatch between initial interfaces (%d) and successfully mapped types (%d). Some interfaces may not have implementation checks performed.", len(interfaces), len(typeToInterfaceMap))
	}

	// Iterate through all types in all packages to check for implementations
	processedTypes := make(map[types.Type]bool) // Avoid redundant checks

	for _, pkg := range pkgs {
		if pkg.Types == nil || pkg.TypesInfo == nil || pkg.Fset == nil { // Ensure Fset is available for location finding
			log.Printf("Skipping implementation check in package %s: missing types, typesInfo, or fset.", pkg.ID)
			continue
		}
		scope := pkg.Types.Scope()
		if scope == nil {
			log.Printf("Skipping implementation check in package %s: scope is nil.", pkg.ID)
			continue
		}

		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			if obj == nil {
				continue
			}

			typeName, ok := obj.(*types.TypeName)
			if !ok {
				continue // We only care about named types for implementations
			}

			implementingType := typeName.Type()
			if implementingType == nil || processedTypes[implementingType] {
				continue // Skip nil types or already processed types
			}
			// Also check pointer to named type if the base type is named
			if named, isNamed := implementingType.(*types.Named); isNamed {
				ptrToNamed := types.NewPointer(named)
				if !processedTypes[ptrToNamed] {
					processedTypes[ptrToNamed] = true // Mark pointer type as processed too
				}
			}
			processedTypes[implementingType] = true

			// Check implementation for each known interface
			for typeInterface, ifaceData := range typeToInterfaceMap {
				// Check value receiver implementation
				if types.Implements(implementingType, typeInterface) {
					// Use the correct FileSet (passed in, ideally from SSA)
					addImplementation(ifaceData, typeName, pkg, false, fset)
				}

				// Check pointer receiver implementation
				// Create pointer type *before* checking Implements and addImplementation
				ptrType := types.NewPointer(implementingType)
				if types.Implements(ptrType, typeInterface) {
					// Use the correct FileSet
					addImplementation(ifaceData, typeName, pkg, true, fset)
				}
			}
		}
	}

	return nil
}

// Helper to find a package by path
func findPackage(pkgs []*packages.Package, path string) *packages.Package {
	for _, p := range pkgs {
		if p.PkgPath == path {
			return p
		}
	}
	return nil
}

// Helper (adapted for datamodel and using provided FileSet)
func addImplementation(iface *datamodel.Interface, typeName *types.TypeName, pkg *packages.Package, isPointer bool, fset *token.FileSet) {
	implLoc := datamodel.Location{}
	var foundNode ast.Node // Keep track of the specific node

	// --- Location Finding Logic ---
	// Priority: Use the provided fset (ideally from SSA) to find the AST node.
	if fset != nil {
		// Find the AST node corresponding to the TypeName's definition using the provided fset
		for _, syntaxFile := range pkg.Syntax {
			if syntaxFile == nil {
				continue
			}
			ast.Inspect(syntaxFile, func(n ast.Node) bool {
				if foundNode != nil {
					return false
				} // Stop searching if already found
				if spec, ok := n.(*ast.TypeSpec); ok {
					if spec.Name != nil && spec.Name.Name == typeName.Name() {
						// Check if the TypeSpec's definition matches the TypeName object using TypesInfo
						if pkg.TypesInfo != nil && pkg.TypesInfo.Defs[spec.Name] == typeName {
							// Use the provided fset to get the position
							pos := fset.Position(spec.Name.Pos()) // Use position of the name identifier
							if pos.IsValid() {
								implLoc = datamodel.NewLocation(pos)
								foundNode = n // Mark as found
								return false  // Stop searching in this subtree
							}
						}
					}
				}
				return true // Continue searching
			})
			if foundNode != nil {
				break // Stop searching other files once found
			}
		}

		if foundNode == nil {
			// Fallback 1: Try obj.Pos() with the provided fset (less reliable than finding AST node)
			pos := fset.Position(typeName.Pos())
			if pos.IsValid() {
				implLoc = datamodel.NewLocation(pos)
				log.Printf("Warning: Could not find AST node for implementation %s.%s. Using fallback location from typeName.Pos() with provided fset: %s:%d", pkg.PkgPath, typeName.Name(), implLoc.Filename, implLoc.Line)
			} else {
				log.Printf("Warning: Could not find AST node or valid typeName.Pos() for implementation %s.%s using provided FileSet.", pkg.PkgPath, typeName.Name())
				// Location remains empty
			}
		}
	} else {
		// Fallback 2: No fset provided. Try using pkg.Fset (might be inconsistent with SSA)
		log.Printf("Warning: Finding location for implementation %s.%s without provided fset. Using pkg.Fset as fallback.", pkg.PkgPath, typeName.Name())
		if pkg.Fset != nil {
			pos := pkg.Fset.Position(typeName.Pos())
			if pos.IsValid() {
				implLoc = datamodel.NewLocation(pos)
				log.Printf("Fallback location found for %s using pkg.Fset: %s:%d", typeName.Name(), implLoc.Filename, implLoc.Line)
			} else {
				log.Printf("Warning: typeName.Pos() for %s.%s is invalid even with pkg.Fset.", pkg.PkgPath, typeName.Name())
			}
		} else {
			log.Printf("Warning: Cannot find location for implementation %s.%s: No fset provided and pkg.Fset is also nil.", pkg.PkgPath, typeName.Name())
		}
	}

	// If no valid location could be found after all fallbacks, skip adding the implementation.
	if implLoc.Filename == "" {
		log.Printf("Skipping implementation %s.%s (pointer: %v) for interface %s due to missing location information.", pkg.PkgPath, typeName.Name(), isPointer, iface.Name)
		return
	}
	// --- End Location Finding ---

	// Avoid duplicate entries (check type name, package path, and pointer status)
	for _, existingImpl := range iface.Implementations {
		if existingImpl.TypeName == typeName.Name() &&
			existingImpl.PackagePath == pkg.PkgPath &&
			existingImpl.IsPointer == isPointer {
			// Optional: Update location if the new one is more specific? For now, just skip duplicates.
			// log.Printf("Debug: Duplicate implementation found for %s.%s (pointer: %v) for interface %s. Skipping.", pkg.PkgPath, typeName.Name(), isPointer, iface.Name)
			return // Already added
		}
	}

	// Add the implementation
	iface.Implementations = append(iface.Implementations, datamodel.Implementation{
		TypeName:    typeName.Name(),
		PackagePath: pkg.PkgPath,
		PackageName: pkg.Name,
		IsPointer:   isPointer,
		Location:    implLoc,
	})
}

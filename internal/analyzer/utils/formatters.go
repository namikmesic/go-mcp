// analyzer/utils/formatters.go
package utils

import (
	"fmt"
	"go/ast"
	"go/token" // Import token for BasicLit Kind check
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/namikmesic/go-mcp/internal/datamodel" // Adjusted import path
)

// FormatMethodSignature creates a readable method signature string.
func FormatMethodSignature(name string, ft *ast.FuncType, pkg *packages.Package) string {
	return name + FormatFuncType(ft, pkg)
}

// FormatFuncType creates a readable function type string (params and results).
func FormatFuncType(ft *ast.FuncType, pkg *packages.Package) string {
	params := FormatFieldList(ft.Params, pkg)
	results := FormatFieldList(ft.Results, pkg)

	paramStr := strings.Join(params, ", ")
	resultStr := strings.Join(results, ", ")

	s := fmt.Sprintf("(%s)", paramStr)
	if len(results) > 0 {
		// Check if it's a single, unnamed return value for simpler formatting
		isSingleUnnamed := ft.Results != nil && len(ft.Results.List) == 1 && (ft.Results.List[0].Names == nil || len(ft.Results.List[0].Names) == 0)
		if isSingleUnnamed {
			s += " " + resultStr
		} else {
			// Multiple or named return values require parentheses
			s += fmt.Sprintf(" (%s)", resultStr)
		}
	}
	return s
}

// FormatFieldList formats a list of parameters or results (ast.FieldList).
func FormatFieldList(list *ast.FieldList, pkg *packages.Package) []string {
	if list == nil {
		return nil
	}
	var parts []string
	for _, f := range list.List {
		if f == nil {
			continue // Defensive check
		}
		typeStr := ExprToString(f.Type, pkg)
		if len(f.Names) > 0 {
			// Field has names (e.g., "a, b int")
			var names []string
			for _, name := range f.Names {
				if name != nil {
					names = append(names, name.Name)
				}
			}
			if len(names) > 0 {
				parts = append(parts, strings.Join(names, ", ")+" "+typeStr)
			} else {
				// Names list exists but is empty/nil names? Fallback to type.
				parts = append(parts, typeStr)
			}
		} else {
			// Field has no names (e.g., "error" in return, or unnamed param)
			parts = append(parts, typeStr)
		}
	}
	return parts
}

// ExtractParameters extracts parameter info from a function type.
func ExtractParameters(ft *ast.FuncType, pkg *packages.Package) []datamodel.Parameter {
	if ft.Params == nil {
		return []datamodel.Parameter{} // Return empty slice, not nil
	}
	var params []datamodel.Parameter
	for _, field := range ft.Params.List {
		if field == nil || field.Type == nil {
			continue // Skip invalid fields
		}

		// Determine if it's a pointer and get the base type string
		isPtr, baseTypeName := IsPointerType(field.Type, pkg)
		typeName := baseTypeName // Start with base type name
		if !isPtr {
			// If not a pointer, get the regular type string
			typeName = ExprToString(field.Type, pkg)
		}

		if len(field.Names) > 0 {
			// Named parameters
			for _, name := range field.Names {
				if name != nil {
					params = append(params, datamodel.Parameter{
						Name:      name.Name,
						Type:      typeName, // Use the (potentially base) type name
						IsPointer: isPtr,
					})
				}
			}
		} else {
			// Unnamed parameter
			params = append(params, datamodel.Parameter{
				Name:      "",       // Keep name empty for unnamed
				Type:      typeName, // Use the (potentially base) type name
				IsPointer: isPtr,
			})
		}
	}
	return params
}

// ExtractReturnTypes extracts return type strings from a function type.
func ExtractReturnTypes(ft *ast.FuncType, pkg *packages.Package) []string {
	if ft.Results == nil {
		return []string{} // Return empty slice, not nil
	}
	var results []string
	for _, field := range ft.Results.List {
		if field == nil || field.Type == nil {
			continue // Skip invalid fields
		}

		typeName := ExprToString(field.Type, pkg)
		// Note: Return types can have names (e.g., `(count int, err error)`).
		// If names are needed, adapt ExtractParameters logic. Here, we just get the type string.
		// We might want to include names in the string if present for clarity.
		if len(field.Names) > 0 {
			var names []string
			for _, name := range field.Names {
				if name != nil {
					names = append(names, name.Name)
				}
			}
			if len(names) > 0 {
				// Optionally format as "name type"
				// results = append(results, strings.Join(names, ", ") + " " + typeName)
				// For now, just append the type string as per original logic, potentially multiple times if multiple names share a type
				for i := 0; i < len(names); i++ {
					results = append(results, typeName)
				}
			} else {
				// Field has Names slice but no actual names? Append type once.
				results = append(results, typeName)
			}
		} else {
			// Unnamed return type, append once.
			results = append(results, typeName)
		}
	}
	return results
}

// IsPointerType checks if an AST expression represents a pointer type (*T)
// and returns true and the underlying base type string (T) if it is.
// Otherwise, returns false and an empty string.
func IsPointerType(expr ast.Expr, pkg *packages.Package) (isPointer bool, baseTypeString string) {
	if starExpr, ok := expr.(*ast.StarExpr); ok {
		if starExpr.X != nil {
			return true, ExprToString(starExpr.X, pkg) // Get string representation of the pointed-to type
		}
		// Pointer to something unidentifiable? Return true but maybe a placeholder string?
		return true, "?"
	}
	return false, ""
}

// ExprToString converts an AST expression (representing a type) to its string representation,
// attempting to handle qualified identifiers using package type information when available.
func ExprToString(expr ast.Expr, pkg *packages.Package) string {
	// Prefer using types.TypeString for accuracy if type info is available
	if pkg != nil && pkg.TypesInfo != nil && pkg.TypesInfo.Types != nil {
		if tv, ok := pkg.TypesInfo.Types[expr]; ok && tv.Type != nil {
			// Use a qualifier function to handle imports correctly within the current package context
			qualifier := func(other *types.Package) string {
				// If the type's package is the same as the current package, no qualifier needed.
				if pkg.Types == other {
					return ""
				}
				// If the type's package is imported, find its local name.
				// This requires iterating through pkg.Imports, which isn't directly stored on packages.Package.
				// A common workaround is to rely on the package name, assuming no conflicts.
				// For robust handling, one might need to build an import map beforehand.
				// Let's use the imported package's name as the qualifier.
				// Check if 'other' package is directly imported by 'pkg'
				// Note: pkg.Imports maps import path to *packages.Package
				for _, importedPkg := range pkg.Imports { // Removed unused importPath
					if importedPkg.Types == other {
						// Found the imported package. Need its local name.
						// This is tricky as the local name isn't directly on packages.Package.
						// We might need to inspect the AST import specs or rely on the default name.
						// Using other.Name() is usually sufficient if there are no renames.
						// TODO: Handle import renames if necessary by inspecting AST ImportSpec.
						return other.Name()
					}
				}
				// If not found in direct imports, it might be a standard library package (like 'error')
				// or from a dependency not explicitly listed in this specific pkg.Imports?
				// Fallback to using the package name.
				return other.Name()
			}
			return types.TypeString(tv.Type, qualifier)
		}
	}

	// Fallback: Basic AST-based string conversion (less accurate for qualified types from other packages)
	switch t := expr.(type) {
	case *ast.Ident:
		// Could be a built-in type or a type defined in the current package.
		return t.Name
	case *ast.StarExpr:
		// Pointer type
		if t.X != nil {
			return "*" + ExprToString(t.X, pkg) // Recursive call for the underlying type
		}
		return "*?" // Pointer to unknown
	case *ast.SelectorExpr:
		// Qualified identifier (e.g., pkg.Type)
		// Without type info, we can only guess the package name part.
		xStr := ExprToString(t.X, pkg) // Recursively get the qualifier (package identifier)
		if xStr != "" && t.Sel != nil {
			return xStr + "." + t.Sel.Name
		}
		if t.Sel != nil {
			return "?." + t.Sel.Name // Best guess if qualifier is unknown
		}
		return "?.?" // Both parts unknown
	case *ast.ArrayType:
		// Array or slice
		lenStr := ""
		if t.Len != nil {
			lenStr = ExprToString(t.Len, pkg) // Get length expression string (could be const or number)
		}
		eltStr := "?"
		if t.Elt != nil {
			eltStr = ExprToString(t.Elt, pkg)
		}
		return "[" + lenStr + "]" + eltStr
	case *ast.MapType:
		keyStr := "?"
		valStr := "?"
		if t.Key != nil {
			keyStr = ExprToString(t.Key, pkg)
		}
		if t.Value != nil {
			valStr = ExprToString(t.Value, pkg)
		}
		return "map[" + keyStr + "]" + valStr
	case *ast.InterfaceType:
		// Interface type definition
		if t.Methods == nil || len(t.Methods.List) == 0 {
			return "interface{}" // Empty interface
		}
		return "interface{...}" // Non-empty interface, abbreviate for simplicity
	case *ast.ChanType:
		// Channel type
		dir := "chan "
		if t.Dir == ast.SEND {
			dir = "chan<- "
		} else if t.Dir == ast.RECV {
			dir = "<-chan "
		}
		valStr := "?"
		if t.Value != nil {
			valStr = ExprToString(t.Value, pkg)
		}
		return dir + valStr
	case *ast.FuncType:
		// Function type
		return "func" + FormatFuncType(t, pkg) // Use existing formatter
	case *ast.StructType:
		// Struct type definition
		if t.Fields == nil || len(t.Fields.List) == 0 {
			return "struct{}" // Empty struct
		}
		return "struct{...}" // Non-empty struct, abbreviate
	case *ast.Ellipsis:
		// Variadic parameter type (e.g., ...string)
		eltStr := "?"
		if t.Elt != nil {
			eltStr = ExprToString(t.Elt, pkg)
		}
		return "..." + eltStr
	case *ast.BasicLit:
		// Basic literal (e.g., array length)
		if t.Kind == token.INT || t.Kind == token.STRING || t.Kind == token.CHAR {
			return t.Value
		}
		return "?" // Other literal types?
	case *ast.ParenExpr:
		// Parenthesized expression (e.g., *(pkg.Type))
		if t.X != nil {
			return "(" + ExprToString(t.X, pkg) + ")"
		}
		return "(?)"
	// Add other common cases if needed
	default:
		// Fallback for unhandled AST node types
		// Using ast.Fprint might be too verbose or fail.
		// buf := new(strings.Builder)
		// err := ast.Fprint(buf, token.NewFileSet(), expr, ast.NotNilFilter)
		// if err == nil { return buf.String() }
		return fmt.Sprintf("?<%T>", expr) // Indicate unknown type and its Go type
	}
}

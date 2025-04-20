// analyzer/ssa/call_analyzer.go
package ssa

import (
	"fmt"
	"go/token"
	"go/types"
	"log"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"

	"github.com/namikmesic/go-mcp/internal/datamodel" // Adjusted import path
)

// SSACallGraphAnalyzer implements CallGraphAnalyzer using SSA.
type SSACallGraphAnalyzer struct{}

func NewSSACallGraphAnalyzer() *SSACallGraphAnalyzer {
	return &SSACallGraphAnalyzer{}
}

func (a *SSACallGraphAnalyzer) AnalyzeCalls(pkgs []*packages.Package) (map[*packages.Package][]datamodel.CallSite, *ssa.Program, *token.FileSet, error) {
	// Build SSA for the loaded packages.
	// BuildSerially can help avoid certain race conditions in the builder
	// InstantiateGenerics is important for handling generic code.
	// SanityCheckFunctions adds extra checks during SSA construction.
	ssaBuildMode := ssa.InstantiateGenerics | ssa.BuildSerially | ssa.SanityCheckFunctions
	prog, ssaPkgs := ssautil.Packages(pkgs, ssaBuildMode)
	if prog == nil {
		// This can happen if pkgs is empty or has critical errors preventing SSA construction.
		log.Println("Error: ssautil.Packages returned nil program. Check package load errors.")
		// Check pkgs length and errors if debugging is needed.
		if len(pkgs) == 0 {
			log.Println("Reason: No packages provided to ssautil.Packages.")
		} else {
			// Log errors from input packages
			for _, pkg := range pkgs {
				if len(pkg.Errors) > 0 {
					log.Printf("Errors in package %s:", pkg.ID)
					for _, err := range pkg.Errors {
						log.Printf("  - %s", err)
					}
				}
			}
		}
		return nil, nil, nil, fmt.Errorf("failed to build SSA program (check package load errors)")
	}

	// It's crucial to build the whole program *before* analyzing members.
	prog.Build()

	fset := prog.Fset // Use the FileSet from the SSA program for consistent positions
	if fset == nil {
		// This would be highly unusual but check just in case
		return nil, nil, nil, fmt.Errorf("SSA program built successfully but has a nil FileSet")
	}
	callsByPackage := make(map[*packages.Package][]datamodel.CallSite)

	// Map ssa.Package back to the original packages.Package for result association
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
			// This might happen if ssautil includes packages not in the original input list (e.g., dependencies for certain build modes)
			// log.Printf("Debug: SSA package %s built but not found in original input package list (index %d).", ssaPkg.Pkg.Path(), i)
		}
	}

	// Iterate through all functions in the SSA program
	allFuncs := ssautil.AllFunctions(prog)
	for fn := range allFuncs {
		// Basic sanity checks for the function and its components
		if fn == nil || fn.Package() == nil || fn.Package().Pkg == nil || fn.Blocks == nil {
			// log.Printf("Debug: Skipping SSA function analysis (nil function, package, Pkg, or blocks): %v", fn)
			continue // Skip functions without bodies or essential package info
		}

		origPkg, ok := ssaToOrigMap[fn.Package()]
		if !ok {
			// This might happen for synthesized functions (like bound methods, thunks for generics)
			// or if mapping failed. Often safe to ignore for basic call graph.
			// log.Printf("Warning: Could not map SSA package '%s' (for function '%s') back to original package. Skipping calls within.", fn.Package().Pkg.Path(), fn.String())
			continue
		}

		callerName := fn.String() // Readable name for the caller function

		for _, b := range fn.Blocks {
			if b == nil {
				continue
			} // Defensive check
			for _, instr := range b.Instrs {
				if instr == nil {
					continue
				} // Defensive check

				// Get source position using the SSA program's FileSet
				pos := fset.Position(instr.Pos())
				if !pos.IsValid() {
					// log.Printf("Debug: Skipping instruction with invalid position in %s: %v", callerName, instr)
					continue // Skip calls without valid source positions
				}

				location := datamodel.NewLocation(pos)
				var callInfo *datamodel.CallSite

				// Use type switch on the instruction itself first
				switch call := instr.(type) {
				case ssa.CallInstruction: // Common interface for Call, Go, Defer
					common := call.Common()
					// Check if common is nil (can happen for certain synthetic instructions)
					if common == nil {
						// log.Printf("Debug: Skipping CallInstruction with nil Common() in %s: %v", callerName, instr)
						continue
					}

					var callType, calleeDesc string

					// Determine call type and description based on the concrete type
					switch c := call.(type) {
					case *ssa.Call:
						callType = "Static" // Default assumption
						if common.IsInvoke() {
							callType = "Interface"
							// Method and Value should be non-nil for invokes
							if common.Method != nil && common.Value != nil && common.Value.Type() != nil {
								// Try to get the concrete type being called if available
								calleeDesc = fmt.Sprintf("Interface method %s on %s", common.Method.Name(), types.TypeString(common.Value.Type(), nil))
							} else {
								calleeDesc = "Unknown Interface Call (nil method/value/type)"
								log.Printf("Warning: Interface call with nil components in %s: Method=%v, Value=%v", callerName, common.Method, common.Value)
							}
						} else {
							// Regular static or dynamic function call
							callee := common.StaticCallee()
							if callee != nil {
								calleeDesc = callee.String() // Static call
							} else if common.Value != nil && common.Value.Type() != nil {
								// Dynamic call via function value
								callType = "Dynamic" // More specific than just 'Static'
								name := common.Value.Name()
								if name == "" {
									name = "anonymous_func_value"
								} // Handle unnamed function values
								calleeDesc = fmt.Sprintf("Dynamic via %s (%s)", name, types.TypeString(common.Value.Type(), nil))
							} else {
								calleeDesc = "Unknown Static/Dynamic Call"
								log.Printf("Warning: Non-invoke call with nil StaticCallee and nil/invalid Value in %s: %v", callerName, common.Value)
							}
						}
					case *ssa.Go:
						callType = "Go"
						callee := common.StaticCallee()
						if callee != nil {
							calleeDesc = callee.String()
						} else if common.Value != nil && common.Value.Type() != nil {
							name := common.Value.Name()
							if name == "" {
								name = "anonymous_func_value"
							}
							calleeDesc = fmt.Sprintf("Dynamic via %s (%s)", name, types.TypeString(common.Value.Type(), nil))
						} else {
							calleeDesc = "Unknown Go Callee"
							log.Printf("Warning: Go instruction with nil StaticCallee and nil/invalid Value in %s: %v", callerName, common.Value)
						}
					case *ssa.Defer:
						callType = "Defer"
						callee := common.StaticCallee()
						if callee != nil {
							calleeDesc = callee.String()
						} else if common.Value != nil && common.Value.Type() != nil {
							name := common.Value.Name()
							if name == "" {
								name = "anonymous_func_value"
							}
							calleeDesc = fmt.Sprintf("Dynamic via %s (%s)", name, types.TypeString(common.Value.Type(), nil))
						} else {
							calleeDesc = "Unknown Defer Callee"
							log.Printf("Warning: Defer instruction with nil StaticCallee and nil/invalid Value in %s: %v", callerName, common.Value)
						}
					default:
						// Should not happen if it implements CallInstruction, but good practice
						log.Printf("Debug: Unhandled CallInstruction type %T in %s", c, callerName)
						continue
					}

					// Ensure calleeDesc is not empty
					if calleeDesc == "" {
						calleeDesc = "Analysis Error: Empty Callee Description"
						log.Printf("Error: Empty callee description generated for %s call in %s. Instruction: %v", callType, callerName, instr)
					}

					callInfo = &datamodel.CallSite{
						CallerFuncDesc: callerName,
						CalleeDesc:     calleeDesc,
						CallType:       callType,
						Location:       location,
					}
					// Add cases for other instruction types if needed in the future
					// case *ssa.Send:
					// case *ssa.Select:
					// ...
				}

				// Append the valid call info to the correct package's list
				if callInfo != nil {
					// Ensure the slice exists for the package
					if _, exists := callsByPackage[origPkg]; !exists {
						callsByPackage[origPkg] = []datamodel.CallSite{}
					}
					callsByPackage[origPkg] = append(callsByPackage[origPkg], *callInfo)
				}
			}
		}
	}

	// Return the map, the program, the fileset, and no error
	return callsByPackage, prog, fset, nil
}

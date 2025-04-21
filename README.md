# goMCP

The main component is the "analyzer" component which extracts structural information, including package details, interface definitions, interface implementations, and call graphs.
The vision is to build a service that will enable building LLM "accelerators", such as MCP knowledge serves, context synthesizers etc.

Non exhaustive list of potential applications that gMCP can accelerate:

- Microservice Architecture Discovery agents - Automatically map out complex microservice ecosystems by analyzing interface implementations and call patterns. Instead of spending weeks understanding a new system, you can visualize the entire architecture in minutes and identify key communication pathways.
- Technical Debt Detection agents - Identify areas of your codebase with excessive interface usage complexity, implementation overloads, or circular dependencies. By quantifying technical debt with concrete metrics, you can make data-driven decisions about refactoring priorities.
- Onboarding Acceleration agents - New team members can understand large codebases exponentially faster through visual exploration of interfaces and implementations rather than linearly reading through files. This could reduce onboarding time from months to weeks or even days.
- API Evolution Planning agents - Track interface changes over time by running regular analyses and storing results. This historical view lets you see how your API surface is evolving, which interfaces are becoming bloated, and where breaking changes might have downstream impacts.
- Dependency Impact Analysis agents - Before making a change to an interface, instantly see all implementations that would be affected across your entire organization. This prevents those "how did changing this one method break production?" moments.
- Architectural Conformance Checking agents - Verify that your implementation adheres to your intended design by defining rules about which packages should implement which interfaces. Run automated checks to catch architectural drift before it becomes problematic.
- Dead Code Elimination agents - Find interfaces with no implementations or implementations never called by any client code. This can substantially reduce your codebase size and maintenance burden by identifying truly unused abstractions.
- Refactoring Opportunity Detection agents - Automatically identify patterns where interface splitting would be beneficial (interfaces with disparate method clusters) or where composition could replace inheritance chains.
- Knowledge Graph Construction agents - Build a team knowledge graph by connecting code ownership data with interface analysis to identify who knows which parts of the system best. When you need to make changes, you'll know exactly who to consult.
- Cross-Language Integration Mapping agents - For polyglot systems, analyze where your Go services connect to services in other languages via common interfaces or API boundaries. This creates a holistic view of your system regardless of implementation language.

The transformative power of this MCP server comes from converting implicit, hidden code relationships into explicit, queryable knowledge. Instead of mentally mapping these connections as you read through code (which doesn't scale past a certain codebase size), you can externalize this understanding into a database that your whole team can explore, query, and build upon.

## How to Run (CLI mode with JSON output)

1.  **Prerequisites:** Ensure you have Go installed (version 1.23.7 or later recommended, as per `go.mod`).
2.  **Clone the repository:** (If you haven't already)
    ```bash
    git clone <repository-url>
    cd go-mcp
    ```
3.  **Run the analysis:** Execute the main program, providing the path to the Go project or package you want to analyze as a command-line argument.

    *   Analyze the current directory:
        ```bash
        go run ./cmd/go-mcp/main.go .
        ```
    *   Analyze a specific project directory:
        ```bash
        go run ./cmd/go-mcp/main.go /path/to/your/go/project
        ```
    *   Analyze all packages within a directory (recursive):
        ```bash
        # The tool automatically appends '/...' when given a directory
        go run ./cmd/go-mcp/main.go /path/to/your/go/project
        ```

The program will output the analysis results in JSON format to standard output and a summary to standard error. Goal is to turn it into an MCP.

## JSON Output Structure

The tool produces an optimized JSON output with the following notable characteristics:

1. **Module information at the top level:**
   - `ModulePath`: The Go module path
   - `ModuleDir`: The absolute directory path where the module resides

2. **Relative file paths:** All file paths are relative to the module directory, making the output more portable.

3. **Optimized field inclusion:**
   - The `Column` field is excluded from all location information
   - Empty arrays like `EmbedFiles`, `EmbedPatterns`, and `Calls` are omitted when they contain no data
   - The `UnderlyingType` field used for internal analysis is excluded from the output

This optimized structure reduces redundancy and improves readability of the JSON output.

## Project Structure

```
go-mcp/
├── cmd/
│   └── go-mcp/
│       └── main.go        # Main application entry point
├── examples/              # Example Go packages for testing/demonstration
│   ├── demo.go
│   └── demo_extended.go
├── internal/              # Internal application packages (not intended for external use)
│   ├── analyzer/          # Core code analysis components
│   │   ├── analyzer.go    # Interfaces for different analyzers
│   │   ├── ast/           # AST-based analysis (e.g., interface definitions)
│   │   │   └── interface_analyzer.go
│   │   ├── ssa/           # SSA-based analysis (e.g., call graphs)
│   │   │   └── call_analyzer.go
│   │   ├── typesystem/    # Type system-based analysis (e.g., implementation finding)
│   │   │   └── implementation_finder.go
│   │   └── utils/         # Utility functions for analysis
│   │       └── formatters.go
│   ├── datamodel/         # Defines the data structures for analysis results
│   │   └── datamodel.go
│   ├── loader/            # Handles loading Go packages
│   │   ├── gopackages.go  # Implementation using golang.org/x/tools/go/packages
│   │   └── loader.go      # Loader interface
│   ├── neo4jstore/        # (Stub) Component for storing results in Neo4j
│   │   └── neo4jstore.go
│   └── service/           # Orchestrates the analysis workflow
│       └── service.go
├── go.mod                 # Go module definition
├── go.sum                 # Dependency checksums
└── README.md              # This file
```

*   **`cmd/go-mcp/main.go`**: Parses command-line arguments, initializes the `AnalysisService`, runs the analysis, and prints the results.
*   **`internal/`**: Contains all the core logic of the application, organized into sub-packages:
    *   **`loader/`**: Responsible for loading Go package information.
    *   **`analyzer/`**: Contains the logic for different types of code analysis (AST, SSA, typesystem).
    *   **`datamodel/`**: Defines the Go structs that hold the extracted information.
    *   **`service/`**: The `AnalysisService` coordinates the loading and analysis steps.
    *   **`neo4jstore/`**: Contains components for persisting analysis results (currently a stub).
*   **`examples/`**: Contains sample Go code that can be used as input for analysis during development or testing (previously `pkg/`).

## Dependencies

*   `golang.org/x/tools/go/packages`: For loading Go package information.
*   `golang.org/x/tools/go/ssa`: For building the SSA representation used in call graph analysis.

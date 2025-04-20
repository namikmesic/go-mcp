# Go MCP Analyzer

This project analyzes Go source code to extract structural information, including package details, interface definitions, interface implementations, and call graphs. The goal is to build a representation of the Go codebase suitable for various code analysis and visualization tasks.

The aim is to assess how large‑context LLMs (e.g., Gemini 2.5 Pro) can independently pursue a predefined objective when given, up front, a comprehensive structural overview, interface specifications, implementation mappings, and package relationships.


## How to Run

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


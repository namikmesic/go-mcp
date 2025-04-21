**Task 1: Modifying Parser for Path & Module Optimization**

**Goal:** Modify the provided Go parser code to optimize the JSON output structure for context efficiency and clarity. Specifically, implement Optimization #1: Path Simplification and Module Context Consolidation.

**Current Parser Behavior (Implied):**
*   The parser currently includes a nested "Module" block within each "Package" object in the JSON output.
*   File paths (in the "Files" array and "Location.Filename" fields) are currently absolute paths.

**Desired Parser Behavior / JSON Output Structure:**
*   The root JSON object should have two new top-level fields: `"ModulePath"` (string) and `"ModuleDir"` (string). These should be populated once with the base module path and directory determined during parsing.
*   The nested `"Module"` block should be completely removed from the JSON output for each package.
*   All file paths stored in the `"Files"` array (within each package) and in the `"Filename"` field (within each "Location" object) should be relative to the base `"ModuleDir"`.



**Task 2: Modifying Parser for Field Removal Optimization**
**Goal:** Modify the provided Go parser code to optimize the JSON output structure by removing low-value or redundant fields (Optimization #2).

**Current Parser Behavior (Implied):**
*   "Location" objects might include a "Column" field.
*   "Package" objects might include "Calls", "EmbedFiles", or "EmbedPatterns" fields, even when they are empty arrays.
*   "Interface" objects might include an "UnderlyingType" field, even when it's an empty object.

**Desired Parser Behavior / JSON Output Structure:**
*   The `"Column"` field should be completely removed from all "Location" objects in the JSON output.
*   The `"Calls"`, `"EmbedFiles"`, and `"EmbedPatterns"` fields should only appear in the "Package" objects if they are *not* empty arrays.
*   The `"UnderlyingType"` field should only appear in the "Interface" objects if it is *not* an empty object.

// Multichecker performs static analysis with standard analyzers
// To execute against root project directory for all packages:
// ./multicheck --all ./...
// To get help ./multicheck --help .
package main

import (
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/appends"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/atomicalign"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/ctrlflow"
	"golang.org/x/tools/go/analysis/passes/deepequalerrors"
	"golang.org/x/tools/go/analysis/passes/defers"
	"golang.org/x/tools/go/analysis/passes/directive"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/fieldalignment"
	"golang.org/x/tools/go/analysis/passes/findcall"
	"golang.org/x/tools/go/analysis/passes/framepointer"
	"golang.org/x/tools/go/analysis/passes/httpmux"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/pkgfact"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/reflectvaluecompare"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/slog"
	"golang.org/x/tools/go/analysis/passes/sortslice"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stdversion"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/testinggoroutine"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/timeformat"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"golang.org/x/tools/go/analysis/passes/unusedwrite"
	"golang.org/x/tools/go/analysis/passes/usesgenerics"
)

func main() {
	multichecker.Main(
		OsExitAnalyzer,           // checks to ensure os.Exit not used in main package
		appends.Analyzer,         // detects if there is only one variable in append.
		asmdecl.Analyzer,         // reports mismatches between assembly files and Go declarations.
		assign.Analyzer,          // detects useless assignments.
		atomic.Analyzer,          // checks for common mistakes using the sync/atomic package.
		atomicalign.Analyzer,     // checks for non-64-bit-aligned arguments to sync/atomic functions.
		bools.Analyzer,           // detects common mistakes involving boolean operators.
		buildssa.Analyzer,        // constructs the SSA representation of an error-free package.
		buildtag.Analyzer,        // checks build tags.
		cgocall.Analyzer,         // detects some violations of the cgo pointer passing rules.
		composite.Analyzer,       // checks for unkeyed composite literals.
		copylock.Analyzer,        // checks for locks erroneously passed by value.
		ctrlflow.Analyzer,        // provides a syntactic control-flow graph (CFG) for the body of a function.
		deepequalerrors.Analyzer, // checks for the use of reflect.DeepEqual with error values.
		defers.Analyzer,          // checks for common mistakes in defer statements.
		directive.Analyzer,       // checks known Go toolchain directives.
		errorsas.Analyzer,        // checks that the second argument to errors.As points to error type.
		fieldalignment.Analyzer,  // detects structs for memory efficiency.
		findcall.Analyzer,        // serves as a trivial example and test of the Analysis API.
		framepointer.Analyzer,    // reports assembly code that clobbers the frame pointer before saving it.
		httpmux.Analyzer,         //
		httpresponse.Analyzer,    // Analyzer that checks for mistakes using HTTP responses.
		ifaceassert.Analyzer,     // flags impossible interface-interface type assertions.
		inspect.Analyzer,         // provides an AST inspector (golang.org/x/tools/go/ast/inspector.Inspector)
		// for the syntax trees of a package.
		loopclosure.Analyzer, // checks for references to enclosing loop variables from within nested functions.
		lostcancel.Analyzer,  // checks for failure to call a context cancellation function.
		nilfunc.Analyzer,     // checks for useless comparisons against nil.
		nilness.Analyzer,     // inspects the control-flow graph of an SSA function and reports
		// errors such as nil pointer dereferences and degenerate nil pointer comparisons.
		pkgfact.Analyzer,             // demonstration and test of the package fact mechanism.
		printf.Analyzer,              // checks consistency of Printf format strings and arguments.
		reflectvaluecompare.Analyzer, // checks for accidentally using == or reflect.DeepEqual to compare
		// reflect.Value values.
		shadow.Analyzer,      // defines an Analyzer that checks for shadowed variables.
		shift.Analyzer,       // checks for shifts that exceed the width of an integer.
		sigchanyzer.Analyzer, // detects misuse of unbuffered signal as argument to signal.Notify.
		slog.Analyzer,        // checks for mismatched key-value pairs in log/slog calls.
		sortslice.Analyzer,   // checks for calls to sort.Slice that do not use a slice type as first argument.
		stdmethods.Analyzer,  // checks for misspellings in the signatures of methods similar to well-known interfaces.
		stdversion.Analyzer,  // uses of standard library symbols that are "too new" for the Go version
		// in force in the referring file.
		stringintconv.Analyzer,    // defines an Analyzer that flags type conversions from integers to strings.
		structtag.Analyzer,        // checks struct field tags are well formed.
		testinggoroutine.Analyzer, // defines an Analyzerfor detecting calls to Fatal from a test goroutine.
		tests.Analyzer,            // checks for common mistaken usages of tests and examples.
		timeformat.Analyzer,       // defines an Analyzer that checks for the use of time.Format or time.Parse
		// calls with a bad format.
		unmarshal.Analyzer, // defines an Analyzer that checks for passing non-pointer or non-interface types
		// to unmarshal and decode functions.
		unreachable.Analyzer,  // defines an Analyzer that checks for unreachable code.
		unsafeptr.Analyzer,    // defines an Analyzer that checks for invalid conversions of uintptr to unsafe.Pointer.
		unusedresult.Analyzer, // defines an analyzer that checks for unused results of calls to certain pure functions.
		unusedwrite.Analyzer,  // checks for unused writes to the elements of a struct or array object.
		usesgenerics.Analyzer, // defines an Analyzer that checks for usage of generic features added in Go 1.18
	)
}

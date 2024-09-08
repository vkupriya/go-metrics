// Custom analyzer checks that there are no direct calls of os.Exit in package main.
package main

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

var OsExitAnalyzer = &analysis.Analyzer{
	Name: "osexit",
	Doc:  "direct calls of os.Exit in package main not allowed .",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "main" {
				continue
			}

			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				sel, ok := call.Fun.(*ast.SelectorExpr)
				if ok {
					ident, ok := sel.X.(*ast.Ident)
					if ok && ident.Name == "os" && sel.Sel.Name == "Exit" {
						pass.Reportf(call.Pos(), "direct calls to os.Exit in main package are not allowed.")
					}
				}
				return true
			})
		}
	}

	return nil, nil
}

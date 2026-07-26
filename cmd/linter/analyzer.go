package main

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "noexit",
	Doc:  "reports panic calls and log.Fatal or os.Exit calls outside main.main",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		inspectFile(pass, file)
	}
	return nil, nil
}

func inspectFile(pass *analysis.Pass, file *ast.File) {
	ast.Inspect(file, func(node ast.Node) bool {
		if function, ok := node.(*ast.FuncDecl); ok {
			inspectBody(pass, function.Body, function)
			return false
		}
		return true
	})
}

func inspectBody(pass *analysis.Pass, body *ast.BlockStmt, function *ast.FuncDecl) {
	ast.Inspect(body, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.FuncLit:
			inspectBody(pass, node.Body, nil)
			return false
		case *ast.CallExpr:
			checkCall(pass, node, function)
		}
		return true
	})
}

func checkCall(pass *analysis.Pass, call *ast.CallExpr, function *ast.FuncDecl) {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		if builtin, ok := pass.TypesInfo.Uses[fun].(*types.Builtin); ok && builtin.Name() == "panic" {
			pass.Reportf(fun.Pos(), "использование встроенной функции panic запрещено")
		}
	case *ast.SelectorExpr:
		if isMainFunction(pass, function) {
			return
		}

		object := pass.TypesInfo.Uses[fun.Sel]
		if object == nil || object.Pkg() == nil {
			return
		}

		packagePath := object.Pkg().Path()
		if packagePath == "log" && object.Name() == "Fatal" {
			pass.Reportf(fun.Pos(), "вызов log.Fatal разрешён только в функции main пакета main")
		}
		if packagePath == "os" && object.Name() == "Exit" {
			pass.Reportf(fun.Pos(), "вызов os.Exit разрешён только в функции main пакета main")
		}
	}
}

func isMainFunction(pass *analysis.Pass, function *ast.FuncDecl) bool {
	return pass.Pkg.Name() == "main" && function != nil && function.Recv == nil && function.Name.Name == "main"
}

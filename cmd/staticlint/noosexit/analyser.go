// Package noosexit - анализатор os.Exit.
package noosexit

import (
	"go/ast"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = `noosexit checks for direct calls to os.Exit in main function of main package

This analyzer reports any direct calls to os.Exit in the main function
of the main package. Such calls can make the code harder to test and
can prevent proper cleanup of resources. Consider using return or
other exit mechanisms instead.`

var Analyzer = &analysis.Analyzer{
	Name:     "noosexit",
	Doc:      doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	// Получаем инспектор из зависимого анализатора
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Фильтруем только вызовы функций
	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	// Проверяем, находимся ли мы в пакете main
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		call := n.(*ast.CallExpr)
		fun, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return // не селектор (например, прямой вызов функции)
		}

		// Проверяем, что это вызов os.Exit
		pkg, ok := fun.X.(*ast.Ident)
		if !ok {
			return
		}

		if pkg.Name != "os" || fun.Sel.Name != "Exit" {
			return
		}

		// Проверяем тип, чтобы быть уверенным, что это действительно os.Exit
		if sel, ok := pass.TypesInfo.Selections[fun]; ok {
			if sel.Obj().Name() == "Exit" && sel.Obj().Pkg().Path() == "os" {
				// Проверяем, находимся ли мы в функции main
				for _, file := range pass.Files {
					for _, decl := range file.Decls {
						if fd, ok := decl.(*ast.FuncDecl); ok && fd.Name.Name == "main" {
							// Проверяем, содержит ли функция main вызов os.Exit
							if containsNode(fd.Body, call) {
								pass.Reportf(call.Pos(), "direct call to os.Exit in main function of main package")
							}
						}
					}
				}
			}
		}
	})

	return nil, nil
}

// containsNode проверяет, содержится ли узел target в дереве node
func containsNode(node ast.Node, target ast.Node) bool {
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if n == target {
			found = true
			return false
		}
		return !found
	})
	return found
}

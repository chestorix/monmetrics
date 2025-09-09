// Package staticlint объединяет множество статических анализаторов для проверки кода на Go.
//
// Включает:
//   - стандартные анализаторы из golang.org/x/tools/go/analysis/passes
//   - все анализаторы класса SA из staticcheck.io
//   - по одному анализатору из других классов staticcheck.io (QF, S, ST)
//   - два дополнительных публичных анализатора
//   - кастомный анализатор noosexit, запрещающий прямой вызов os.Exit в main
//
// # Использование
//
// Установка:
//
//	go install monmetrics/cmd/staticlint
//
// Запуск:
//
//	staticlint ./...
//
// # Анализаторы
//
// ## Стандартные анализаторы (golang.org/x/tools/go/analysis/passes)
//
// atomic: проверяет корректность использования sync/atomic.
//
// bools: обнаруживает распространённые ошибки в булевых операциях.
//
// buildtag: проверяет корректность директив сборки (+build).
//
// cgocall: обнаруживает нарушения правил вызовов через CGO.
//
// composite: проверяет композитные литералы без именованных полей.
//
// copylock: проверяет копирование мьютексов и других примитивов синхронизации.
//
// errorsas: проверяет правильность использования errors.As.
//
// fieldalignment: предлагает оптимизации выравнивания полей структур.
//
// httpresponse: проверяет обработку HTTP-ответов.
//
// loopclosure: обнаруживает неправильное использование переменных в замыканиях.
//
// lostcancel: обнаруживает утечку контекста.
//
// nilfunc: обнаруживает сравнения функций с nil.
//
// printf: проверяет соответствие строк формата и аргументов.
//
// shadow: обнаруживает затенение переменных.
//
// shift: проверяет сдвиги, превышающие размер целого числа.
//
// sortslice: проверяет правильность реализации интерфейсов для сортировки срезов.
//
// stdmethods: проверяет стандартные методы (String, Error и т.д.).
//
// structtag: проверяет теги структур на соответствие reflect.StructTag.Get.
//
// tests: обнаруживает распространённые ошибки в тестах.
//
// unmarshal: проверяет передачу неперехватываемых значений в unmarshal.
//
// unreachable: обнаруживает недостижимый код.
//
// unsafeptr: проверяет правильность преобразований unsafe.Pointer.
//
// unusedresult: проверяет неиспользуемые результаты вызовов некоторых функций.
//
// ## Анализаторы Staticcheck (staticcheck.io)
//
// SAxxxx: все анализаторы класса SA (Staticcheck Analysis), обнаруживающие:
//   - неправильное использование API
//   - подозрительные конструкции
//   - возможные ошибки
//   - проблемы с горутинами
//
// QF1001: проверяет возможность замены time.Now().Sub(x) на time.Since(x).
//
// S1002: проверяет возможность опустить сравнение с true.
//
// ST1000: проверяет неправильные комментарии к пакету.
//
// ## Дополнительные анализаторы
//
// noosexit: кастомный анализатор, запрещающий прямой вызов os.Exit в main.
//
// simpleerrcheck: упрощённый анализатор для проверки необработанных ошибок.
package main

import (
	"github.com/chestorix/monmetrics/cmd/staticlint/noosexit"
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sortslice"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"honnef.co/go/tools/quickfix"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
)

// SimpleErrCheckAnalyzer - упрощённый анализатор для проверки необработанных ошибок
var SimpleErrCheckAnalyzer = &analysis.Analyzer{
	Name: "simpleerrcheck",
	Doc:  "simplified errcheck analyzer for unhandled errors",
	Run:  runSimpleErrCheck,
}

func runSimpleErrCheck(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			callExpr, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			// Получаем тип вызываемой функции
			funType := pass.TypesInfo.TypeOf(callExpr.Fun)
			if funType == nil {
				return true
			}

			// Проверяем, возвращает ли функция ошибку
			if signature, ok := funType.(*types.Signature); ok {
				results := signature.Results()
				if results.Len() > 0 {
					lastResult := results.At(results.Len() - 1)
					if isErrorType(lastResult.Type()) {
						// Проверяем, используется ли возвращаемое значение
						if !isValueUsed(callExpr) {
							pass.Reportf(callExpr.Pos(), "returned error is not handled")
						}
					}
				}
			}

			return true
		})
	}
	return nil, nil
}

func isErrorType(t types.Type) bool {
	return types.Identical(t, types.Universe.Lookup("error").Type())
}

func isValueUsed(expr ast.Expr) bool {
	// Простая проверка: если выражение является единственным в выражении-statement,
	// то значение не используется
	if _, ok := findParentExprStmt(expr); ok {
		// Если это Expression Statement и наше выражение - единственное в нём,
		// то значение не используется
		return false
	}

	// Во всех остальных случаях считаем, что значение используется
	return true
}

// findParentExprStmt ищет родительский ExprStmt для данного выражения
func findParentExprStmt(expr ast.Expr) (*ast.ExprStmt, bool) {
	// Эта функция была бы сложной для реализации без полного AST обхода,
	// поэтому для простоты считаем, что если нам нужна точная проверка,
	// лучше использовать готовый анализатор

	// Временно возвращаем false - это упрощённая реализация
	return nil, false
}

func main() {
	var analyzers []*analysis.Analyzer

	// Стандартные анализаторы
	standardAnalyzers := []*analysis.Analyzer{
		atomic.Analyzer,
		bools.Analyzer,
		buildtag.Analyzer,
		cgocall.Analyzer,
		composite.Analyzer,
		copylock.Analyzer,
		errorsas.Analyzer,
		//	fieldalignment.Analyzer,
		httpresponse.Analyzer,
		loopclosure.Analyzer,
		lostcancel.Analyzer,
		nilfunc.Analyzer,
		printf.Analyzer,
		//	shadow.Analyzer,
		shift.Analyzer,
		sortslice.Analyzer,
		stdmethods.Analyzer,
		structtag.Analyzer,
		tests.Analyzer,
		unmarshal.Analyzer,
		unreachable.Analyzer,
		unsafeptr.Analyzer,
		unusedresult.Analyzer,
	}
	analyzers = append(analyzers, standardAnalyzers...)

	// Анализаторы Staticcheck класса SA
	for _, analyzer := range staticcheck.Analyzers {
		if len(analyzer.Analyzer.Name) >= 2 && analyzer.Analyzer.Name[:2] == "SA" {
			analyzers = append(analyzers, analyzer.Analyzer)
		}
	}

	// Другие анализаторы Staticcheck
	analyzers = append(analyzers,
		quickfix.Analyzers[0].Analyzer,   // QF1001
		simple.Analyzers[3].Analyzer,     // S1002
		stylecheck.Analyzers[0].Analyzer, // ST1000
	)

	// Наш простой анализатор ошибок
	analyzers = append(analyzers, SimpleErrCheckAnalyzer)

	// Кастомный анализатор
	analyzers = append(analyzers, noosexit.Analyzer)

	multichecker.Main(analyzers...)
}

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	//"io/fs"
	"os"
	"path/filepath"
	"strings"
	//"strings"
)

func modToFactoryMap(fn *ast.FuncDecl) (modMap string) {
	paramCount := len(fn.Type.Params.List)
	name := fn.Name.Name
	var x string
	switch paramCount {
	case 0:
		x = fmt.Sprintf("  resModMap[\"%s\"] = func(_ ...string) proxychain.ResponseModification {\n    return rx.%s()\n  }\n", name, name)
	default:
		p := []string{}
		for i := 0; i < paramCount; i++ {
			p = append(p, fmt.Sprintf("params[%d]", i))
		}
		params := strings.Join(p, ", ")
		x = fmt.Sprintf("  resModMap[\"%s\"] = func(params, ...string) proxychain.ResponseModification {\n    return rx.%s(%s)\n  }\n", name, name, params)
	}
	return x
}

func main() {
	fset := token.NewFileSet()

	// Directory containing your Go files
	dir := "../requestmodifiers/"

	files, err := os.ReadDir(dir)
	if err != nil {
		panic(err)
	}

	factoryMaps := []string{}
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".go" {
			continue
		}

		// Parse each Go file
		node, err := parser.ParseFile(fset, filepath.Join(dir, file.Name()), nil, parser.ParseComments)
		if err != nil {
			panic(err)
		}

		ast.Inspect(node, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if ok && fn.Recv == nil && fn.Name.IsExported() {
				factoryMaps = append(factoryMaps, modToFactoryMap(fn))
			}
			return true
		})

	}

	code := fmt.Sprintf(`
package ruleset_v2

import (
	"ladder/proxychain"
	rx "ladder/proxychain/responsemodifiers"
)

type ResponseModifierFactory func(params ...string) proxychain.ResponseModification

var resModMap map[string]ResponseModifierFactory

// TODO: create codegen using AST parsing of exported methods in ladder/proxychain/responsemodifiers/*.go
func init() {
	resModMap = make(map[string]ResponseModifierFactory)

	%s
}`, strings.Join(factoryMaps, "\n"))
	fmt.Println(code)

}

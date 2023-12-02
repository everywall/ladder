// code gen for struct def seen in https://github.com/joncrangle/ladder/blob/feat/playground/handlers/playground.go

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"path/filepath"
	"strings"
)

func main() {
	fset := token.NewFileSet()
	structFields := []string{}

	// Directory containing your Go files
	dir := "./proxychain/requestmodifers/"

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".go" {
			// Parse each Go file
			node, err := parser.ParseFile(fset, filepath.Join(dir, file.Name()), nil, parser.ParseComments)
			if err != nil {
				panic(err)
			}

			ast.Inspect(node, func(n ast.Node) bool {
				fn, ok := n.(*ast.FuncDecl)
				if ok && fn.Recv == nil && fn.Name.IsExported() {
					fieldName := fn.Name.Name
					jsonTag := strings.ToLower(fieldName)
					fieldType := "bool" // default type if no parameters

					// Check if the function has parameters
					if len(fn.Type.Params.List) > 0 {
						// Assuming only one parameter of type string
						paramName := fn.Type.Params.List[0].Names[0].Name
						paramType := "string" // assuming string type
						fieldType = fmt.Sprintf("struct{ %s %s `json:\"%s\"` }", paramName, paramType, paramName)
					}

					structField := fmt.Sprintf("%s %s `json:\"%s\"`", fieldName, fieldType, jsonTag)
					structFields = append(structFields, structField)
				}
				return true
			})
		}
	}

	structDef := "type ResponseModifierQuery struct {\n\t" + strings.Join(structFields, "\n\t") + "\n}"
	fmt.Println(structDef)
}

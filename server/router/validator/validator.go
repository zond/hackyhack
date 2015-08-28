package validator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

var allowedImports = map[string]bool{
	"github.com/zond/hackyhack/proc/interfaces": true,
	"github.com/zond/hackyhack/proc/slave":      true,
}

type validator struct {
	disallowed []string
}

func (v *validator) Visit(n ast.Node) ast.Visitor {
	if importNode, isImport := n.(*ast.ImportSpec); isImport {
		if importNode.Path != nil {
			if !allowedImports[importNode.Path.Value] {
				v.disallowed = append(v.disallowed, importNode.Path.Value)
			}
		}
	}
	return v
}

func Validate(code string) error {
	f, err := parser.ParseFile(&token.FileSet{}, "", code, 0)
	if err != nil {
		return err
	}
	v := &validator{}
	ast.Walk(v, f)
	if len(v.disallowed) > 0 {
		return fmt.Errorf("Code imports disallowed packages: %+v", v.disallowed)
	}
	return nil
}

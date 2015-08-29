package validator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

const (
	slave = "\"github.com/zond/hackyhack/proc/slave\""
)

var allowedImports = map[string]bool{
	slave:                                                true,
	"\"strings\"":                                        true,
	"\"github.com/zond/hackyhack/client/commands\"":      true,
	"\"github.com/zond/hackyhack/client/util\"":          true,
	"\"github.com/zond/hackyhack/proc/interfaces\"":      true,
	"\"github.com/zond/hackyhack/proc/messages\"":        true,
	"\"github.com/zond/hackyhack/proc/slave/delegator\"": true,
}

type validator struct {
	disallowed   []string
	importsSlave bool
}

func (v *validator) Visit(n ast.Node) ast.Visitor {
	if importNode, isImport := n.(*ast.ImportSpec); isImport {
		if importNode.Path != nil {
			if !allowedImports[importNode.Path.Value] {
				v.disallowed = append(v.disallowed, importNode.Path.Value)
			}
			if importNode.Path.Value == slave {
				v.importsSlave = true
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
	if !v.importsSlave {
		return fmt.Errorf("Code doesn't import required package %v", slave)
	}
	return nil
}

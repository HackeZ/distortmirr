package brush

import (
	"fmt"
	"go/ast"
	"strings"
)

func GetIOutput(fields *ast.FieldList) []string {
	if fields == nil {
		return nil
	}

	ioutputs := make([]string, 0, len(fields.List))
	for _, f := range fields.List {
		names := make([]string, 0, len(f.Names))
		for _, n := range f.Names {
			names = append(names, n.Name)
		}

		typeName := GetTypeName(f.Type)
		ioutputs = append(ioutputs, strings.Join(names, ", ")+" "+typeName)
	}

	return ioutputs
}

func GetParamNames(fields *ast.FieldList) []string {
	names := make([]string, 0, len(fields.List))
	for _, f := range fields.List {
		for _, n := range f.Names {
			names = append(names, n.Name)
		}
	}
	return names
}

func GetTypeName(f ast.Expr) string {
	if f == nil {
		return ""
	}

	switch f.(type) {
	case *ast.StarExpr:
		return "*" + GetTypeName(f.(*ast.StarExpr).X)
	case *ast.Ident:
		return f.(*ast.Ident).Name

	case *ast.SelectorExpr:
		return f.(*ast.SelectorExpr).X.(*ast.Ident).Name +
			"." +
			f.(*ast.SelectorExpr).Sel.Name

	case *ast.InterfaceType:
		if len(f.(*ast.InterfaceType).Methods.List) > 0 {
			panic("anonymous interface not support")
		}
		return "interface{}"

	case *ast.MapType:
		t := f.(*ast.MapType)
		return "map[ " + GetTypeName(t.Value) + " ]" +
			GetTypeName(t.Value)

	case *ast.ArrayType:
		t := f.(*ast.ArrayType)
		return "[" + GetTypeName(t.Len) + "]" + GetTypeName(t.Elt)

	case *ast.ChanType:
		t := f.(*ast.ChanType)
		switch t.Dir {
		case ast.SEND:
			return "<-" + GetTypeName(t.Value)
		case ast.RECV:
			return GetTypeName(t.Value) + "<-"
		default:
			// must both SEND & RECV
			return GetTypeName(t.Value)
		}

	case *ast.Ellipsis:
		t := f.(*ast.Ellipsis)
		return "..." + GetTypeName(t.Elt)

	default:
		// attempt to known what type is it for debug...
		_ = f.(*ast.BadExpr)
		panic(fmt.Errorf("unknown expression: %+v", f))
	}
}

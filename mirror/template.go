package mirror

import (
	"fmt"
	"go/ast"
	"io"
	"strings"
	"text/template"
)

var (
	typeTemp *template.Template
	methTemp *template.Template
	funcTemp *template.Template
)

func init() {
	var err error
	typeTemp, err = template.New("mirr-type").Parse(`
type {{ .TypeName }} struct {
    {{ .InnerName }} {{ .PkgName }}.{{ .TypeName }}
}
`)
	if err != nil {
		panic(err)
	}

	methTemp, err = template.New("mirr-meth").Parse(`
func ({{ .InnerName }} {{ .TypeName }}) {{ .MethName }} ({{ .Input }}) {{ if ne .Output "" }}({{.Output}}){{ end }} {
    {{ if eq .Output "" -}}
	{{ .InnerName }}.{{ .InnerName }}.{{ .MethName }}({{ .Parameters }})
	{{- else -}}
	return {{ .InnerName }}.{{ .InnerName }}.{{ .MethName }}({{ .Parameters }})
	{{- end }}
}
`)
	if err != nil {
		panic(err)
	}

	funcTemp, err = template.New("mirr-func").Parse(`
func {{ .FuncName }}({{ .Input }}) {{ if ne .Output "" }}({{.Output}}){{ end }} {
    {{ if eq .Output "" -}}
	{{ .PkgName }}.{{ .FuncName }}({{ .Parameters }})
	{{- else -}}
	return {{ .PkgName }}.{{ .FuncName }}({{ .Parameters }})
	{{- end }}
}
`)
	if err != nil {
		panic(err)
	}

}

func (m *Mirror) renderType(wr io.Writer, decl *ast.GenDecl) error {
	typeDecl := decl.Specs[0].(*ast.TypeSpec)
	data := struct {
		TypeName  string
		InnerName string
		PkgName   string
	}{
		TypeName:  typeDecl.Name.Name,
		InnerName: strings.ToLower(typeDecl.Name.Name[:1]),
		PkgName:   m.pkgname,
	}

	return typeTemp.Execute(wr, data)
}

func (m *Mirror) renderMeth(wr io.Writer, typeName string, decl *ast.FuncDecl) error {
	data := struct {
		TypeName   string
		InnerName  string
		MethName   string
		Input      string
		Output     string
		Parameters string
	}{
		TypeName:   typeName,
		InnerName:  strings.ToLower(typeName[:1]),
		MethName:   decl.Name.Name,
		Input:      strings.Join(getIOutput(decl.Type.Params), ", "),
		Output:     strings.Join(getIOutput(decl.Type.Results), ", "),
		Parameters: strings.Join(getParamNames(decl.Type.Params), ", "),
	}
	return methTemp.Execute(wr, data)
}

func (m *Mirror) renderFunc(wr io.Writer, decl *ast.FuncDecl) error {
	data := struct {
		FuncName   string
		Input      string
		Output     string
		PkgName    string
		Parameters string
	}{
		FuncName:   decl.Name.Name,
		Input:      strings.Join(getIOutput(decl.Type.Params), ", "),
		Output:     strings.Join(getIOutput(decl.Type.Results), ", "),
		PkgName:    m.pkgname,
		Parameters: strings.Join(getParamNames(decl.Type.Params), ", "),
	}
	return funcTemp.Execute(wr, data)
}

func getIOutput(fields *ast.FieldList) []string {
	if fields == nil {
		return nil
	}

	ioutputs := make([]string, 0, len(fields.List))
	for _, f := range fields.List {
		names := make([]string, 0, len(f.Names))
		for _, n := range f.Names {
			names = append(names, n.Name)
		}

		typeName := getTypeName(f.Type)
		ioutputs = append(ioutputs, strings.Join(names, ", ")+" "+typeName)
	}

	return ioutputs
}

func getParamNames(fields *ast.FieldList) []string {
	names := make([]string, 0, len(fields.List))
	for _, f := range fields.List {
		for _, n := range f.Names {
			names = append(names, n.Name)
		}
	}
	return names
}

func getTypeName(f ast.Expr) string {
	if f == nil {
		return ""
	}

	switch f.(type) {
	case *ast.StarExpr:
		return "*" + getTypeName(f.(*ast.StarExpr).X)
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
		return "map[ " + getTypeName(t.Value) + " ]" +
			getTypeName(t.Value)

	case *ast.ArrayType:
		t := f.(*ast.ArrayType)
		return "[" + getTypeName(t.Len) + "]" + getTypeName(t.Elt)

	case *ast.ChanType:
		t := f.(*ast.ChanType)
		switch t.Dir {
		case ast.SEND:
			return "<-" + getTypeName(t.Value)
		case ast.RECV:
			return getTypeName(t.Value) + "<-"
		default:
			// must both SEND & RECV
			return getTypeName(t.Value)
		}
	default:
		// attempt to known what type is it for debug...
		_ = f.(*ast.BadExpr)
		panic(fmt.Errorf("unknown expression: %+v", f))
	}
}

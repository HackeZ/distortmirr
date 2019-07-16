package davinci

import (
	"fmt"
	"go/ast"
	"html/template"
	"io"
	"strings"

	"github.com/hackerzgz/distortmirr/mirror/brush"
)

type Davinci struct {
	pkgname string

	types map[string]*ast.GenDecl
	meths map[string]map[string]*ast.FuncDecl
	funcs map[string]*ast.FuncDecl
}

var (
	typeTemp *template.Template
	methTemp *template.Template
	funcTemp *template.Template
)

func init() {
	var err error
	typeTemp, err = template.New("davinci-type").Parse(`
type {{ .TypeName }} struct {
    {{ .InnerName }} {{ .PkgName }}.{{ .TypeName }}
}
`)
	if err != nil {
		panic(err)
	}

	methTemp, err = template.New("davinci-meth").Parse(`
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

	funcTemp, err = template.New("davinci-func").Parse(`
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

func New(pkgname string, types map[string]*ast.GenDecl,
	meths map[string]map[string]*ast.FuncDecl, funcs map[string]*ast.FuncDecl,
) *Davinci {
	return &Davinci{
		pkgname: pkgname,
		types:   types,
		meths:   meths,
		funcs:   funcs,
	}
}

// Render a wrapper of package
func (m *Davinci) Render(wr io.Writer) (err error) {
	fmt.Println("davinci start printing...")

	for tname, typ := range m.types {
		err = m.renderType(wr, typ)
		if err != nil {
			return err
		}
		for _, meth := range m.meths[tname] {
			err = m.renderMeth(wr, tname, meth)
			if err != nil {
				return err
			}
		}
	}

	for _, fn := range m.funcs {
		err = m.renderFunc(wr, fn)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Davinci) renderType(wr io.Writer, decl *ast.GenDecl) error {
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

func (m *Davinci) renderMeth(wr io.Writer, typeName string, decl *ast.FuncDecl) error {
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
		Input:      strings.Join(brush.GetIOutput(decl.Type.Params), ", "),
		Output:     strings.Join(brush.GetIOutput(decl.Type.Results), ", "),
		Parameters: strings.Join(brush.GetParamNames(decl.Type.Params), ", "),
	}
	return methTemp.Execute(wr, data)
}

func (m *Davinci) renderFunc(wr io.Writer, decl *ast.FuncDecl) error {
	data := struct {
		FuncName   string
		Input      string
		Output     string
		PkgName    string
		Parameters string
	}{
		FuncName:   decl.Name.Name,
		Input:      strings.Join(brush.GetIOutput(decl.Type.Params), ", "),
		Output:     strings.Join(brush.GetIOutput(decl.Type.Results), ", "),
		PkgName:    m.pkgname,
		Parameters: strings.Join(brush.GetParamNames(decl.Type.Params), ", "),
	}
	return funcTemp.Execute(wr, data)
}

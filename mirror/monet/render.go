package monet

import (
	"go/ast"
	"html/template"
	"io"
	"strings"

	"github.com/hackerzgz/distortmirr/mirror/brush"
)

var (
	interfaceTemp *template.Template
)

func init() {
	var err error
	interfaceTemp, err = template.New("monet-interface").Parse(`
type {{ .TypeName }}er interface {
    {{- range $idx, $meth := .Meths }}
	    {{ $meth.Name }} ({{ $meth.Input }}) {{- if ne $meth.Ouput "" }} ({{$meth.Ouput}}) {{ end -}}
	{{ end }}
}
`)
	if err != nil {
		panic(err)
	}
}

type Monet struct {
	types map[string]*ast.GenDecl
	meths map[string]map[string]*ast.FuncDecl
}

func New(types map[string]*ast.GenDecl,
	meths map[string]map[string]*ast.FuncDecl) *Monet {
	return &Monet{
		types: types,
		meths: meths,
	}
}

func (m *Monet) Render(wr io.Writer) (err error) {
	for tname := range m.types {
		if err = m.renderInterface(wr, tname); err != nil {
			return err
		}
	}
	return nil
}

func (m *Monet) renderInterface(wr io.Writer, name string) error {
	data := struct {
		TypeName string
		Meths    []struct {
			Name  string
			Input string
			Ouput string
		}
	}{
		TypeName: name,
	}

	data.Meths = make([]struct {
		Name  string
		Input string
		Ouput string
	}, 0, len(m.meths[name]))
	for mname, meth := range m.meths[name] {
		data.Meths = append(data.Meths, struct {
			Name  string
			Input string
			Ouput string
		}{
			Name:  mname,
			Input: strings.Join(brush.GetIOutput(meth.Type.Params), ", "),
			Ouput: strings.Join(brush.GetIOutput(meth.Type.Results), ", "),
		})
	}
	// skip the struct have no methods
	if len(data.Meths) == 0 {
		return nil
	}

	return interfaceTemp.Execute(wr, data)
}

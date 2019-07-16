package monet

import (
	"go/ast"
	"html/template"
	"io"
	"strings"

	"github.com/emirpasic/gods/maps/treemap"
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
	types *treemap.Map
	meths *treemap.Map
}

func New(types *treemap.Map, meths *treemap.Map) *Monet {
	return &Monet{
		types: types,
		meths: meths,
	}
}

func (m *Monet) Render(wr io.Writer) (err error) {
	typeIter := m.types.Iterator()
	for typeIter.Next() {
		tname := typeIter.Key().(string)
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
	var meths *treemap.Map
	if mm, found := m.meths.Get(name); found {
		meths = mm.(*treemap.Map)
	} else {
		return nil
	}
	data.Meths = make([]struct {
		Name  string
		Input string
		Ouput string
	}, 0, meths.Size())

	methIter := meths.Iterator()
	for methIter.Next() {
		mname, meth := methIter.Key().(string), methIter.Value().(*ast.FuncDecl)
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

package mirror

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hackerzgz/distortmirr/mirror/davinci"
	"github.com/hackerzgz/distortmirr/mirror/monet"
)

type ScanMode int

const (
	ScanAll ScanMode = iota
	ScanPublic
)

type Mirror struct {
	mode ScanMode

	gopaths []string
	pkgpath string
	pkgname string

	mtx   *sync.Mutex
	types map[string]*ast.GenDecl
	meths map[string]map[string]*ast.FuncDecl
	funcs map[string]*ast.FuncDecl
}

func New(pkgpath string, mode ScanMode) (*Mirror, error) {
	if pkgpath == "" {
		return nil, errors.New("invalid arguments: package path cannot be empty")
	}
	gopath := os.Getenv("GOPATH")
	if "" == gopath {
		return nil, errors.New("failed to find GOPATH environment")
	}

	m := &Mirror{
		mode:    mode,
		gopaths: strings.Split(gopath, ":"),
		pkgpath: strings.TrimSuffix(pkgpath, "/"),
	}
	m.pkgname = m.pkgpath[strings.LastIndex(m.pkgpath, "/")+1:]

	m.mtx = new(sync.Mutex)
	m.types = make(map[string]*ast.GenDecl)
	m.meths = make(map[string]map[string]*ast.FuncDecl)
	m.funcs = make(map[string]*ast.FuncDecl)

	return m, nil
}

func (m *Mirror) Scan() error {
	for _, p := range m.gopaths {
		// create a new file set on each path
		fs := token.NewFileSet()
		// combine a complete package path
		if !strings.HasSuffix(p, "/") {
			p += "/"
		}
		p += "src/" + m.pkgpath

		if _, err := os.Stat(p); err != nil {
			continue
		}

		// attempt to find out package
		err := filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
			switch {
			case err != nil:
				return err
			// skip to next step if directory met
			case info.IsDir(),
				!strings.HasSuffix(info.Name(), ".go"),
				strings.HasSuffix(info.Name(), "_test.go"):
				return nil
			}

			return m.parseFile(fs, p+"/"+info.Name())
		})
		if err != nil {
			if err == os.ErrNotExist {
				continue
			}
		}
	}

	return nil
}

func (m *Mirror) parseFile(fs *token.FileSet, fname string) error {
	f, err := parser.ParseFile(fs, fname, nil, 0)
	if err != nil {
		return err
	}

	for _, decl := range f.Decls {
		switch decl.(type) {
		case *ast.GenDecl:
			if decl.(*ast.GenDecl).Tok == token.TYPE {
				m.registerType(decl.(*ast.GenDecl))
			}
		case *ast.FuncDecl:
			if decl.(*ast.FuncDecl).Recv != nil {
				m.registerMeth(decl.(*ast.FuncDecl))
			} else {
				m.registerFunc(decl.(*ast.FuncDecl))
			}
		}
	}
	return nil
}

func (m *Mirror) registerType(decl *ast.GenDecl) {
	var typeName string
	for _, spec := range decl.Specs {
		if s, ok := spec.(*ast.TypeSpec); ok {
			typeName = s.Name.Name
			break
		}
	}
	if m.mode == ScanPublic && !ast.IsExported(typeName) {
		return
	}

	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.types[typeName] = decl
}

func (m *Mirror) registerMeth(decl *ast.FuncDecl) {
	if m.mode == ScanPublic && !ast.IsExported(decl.Name.Name) {
		return
	}

	if len(decl.Recv.List) <= 0 {
		panic("unknown method receiver: " + decl.Name.Name)
	}

	// find out the explicit type of method receiver
	typeName := getTypeName(decl.Recv.List[0].Type)
	if typeName == "" {
		panic("failed to find out explicit type of method: " + decl.Name.Name)
	}
	typeName = strings.TrimPrefix(typeName, "*")

	// lock the mirror for register methods
	m.mtx.Lock()
	defer m.mtx.Unlock()

	if _, found := m.meths[typeName]; !found {
		m.meths[typeName] = make(map[string]*ast.FuncDecl)
	}
	m.meths[typeName][decl.Name.Name] = decl
}

func (m *Mirror) registerFunc(decl *ast.FuncDecl) {
	if m.mode == ScanPublic && !ast.IsExported(decl.Name.Name) {
		return
	}

	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.funcs[decl.Name.Name] = decl
}

type Renderer interface {
	Render(wr io.Writer) error
}

func (m *Mirror) newRenderer(name string) (Renderer, error) {
	switch name {
	case "davinci":
		return davinci.New(m.pkgname, m.types, m.meths, m.funcs), nil
	case "monet":
		return monet.New(m.types, m.meths), nil
	default:
		return nil, errors.New("renderer not supported: " + name)
	}
}

// Render a wrapper of package
func (m *Mirror) Render(name string, wr io.Writer) (err error) {
	fmt.Println("start render...")
	m.mtx.Lock()
	defer m.mtx.Unlock()

	r, e := m.newRenderer(name)
	if e != nil {
		return e
	}
	return r.Render(wr)
}

package bson

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/15mga/kiwi/util"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func Run() {
	dir := flag.String("d", "", "pb out dir")
	flag.Parse()
	root := *dir
	err := filepath.WalkDir(root, func(fullName string, d fs.DirEntry, err error) error {
		if fullName == root || d.IsDir() {
			return nil
		}
		bs, err := os.ReadFile(fullName)
		if err != nil {
			return err
		}
		fileSet := token.NewFileSet()
		f, err := parser.ParseFile(fileSet, fullName, bs, parser.ParseComments)
		if err != nil {
			return err
		}
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts := spec.(*ast.TypeSpec)
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				for _, fd := range st.Fields.List {
					if len(fd.Names) != 1 || fd.Tag == nil {
						continue
					}
					fn := fd.Names[0].Name
					val := fd.Tag.Value
					if strings.Index(val, "bson:") > -1 {
						continue
					}
					valLen := len(val)
					is := ""
					if fn == "Id" {
						is = " bson:\"_id\""
					} else {
						is = fmt.Sprintf(" bson:\"%s\"", util.ToUnderline(fn))
					}
					fd.Tag.Value = val[:valLen-1] + is + val[valLen-1:]
				}
			}
		}
		var buff bytes.Buffer
		err = format.Node(&buff, fileSet, f)
		if err != nil {
			return err
		}
		return os.WriteFile(fullName, buff.Bytes(), fs.ModePerm)
	})
	if err != nil {
		panic(err)
	}
}

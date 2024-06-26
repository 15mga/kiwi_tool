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
				for _, field := range st.Fields.List {
					if len(field.Names) != 1 || field.Tag == nil {
						continue
					}
					fieldName := field.Names[0].Name
					val := field.Tag.Value
					if strings.Index(val, "bson:") > -1 {
						continue
					}
					valLen := len(val)
					is := ""
					if fieldName == "Id" {
						is = " bson:\"_id\""
					} else {
						is = fmt.Sprintf(" bson:\"%s\"", util.ToUnderline(fieldName))
					}
					field.Tag.Value = val[:valLen-1] + is + val[valLen-1:]
					//移除忽略 0 值
					field.Tag.Value = strings.Replace(field.Tag.Value, ",omitempty", "", -1)
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

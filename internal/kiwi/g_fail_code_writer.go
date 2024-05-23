package kiwi

import (
	"errors"
	"fmt"
	"github.com/15mga/kiwi/util"
	tool "github.com/15mga/kiwi_tool"
	"sort"
	"strings"
)

func newGFailCodeWriter() IWriter {
	return &gFailCodeWriter{}
}

type gFailCodeWriter struct {
	baseWriter
}

func (w *gFailCodeWriter) Reset() {
	w.SetDirty(true)
}

func (w *gFailCodeWriter) Save() error {
	bd := strings.Builder{}
	bd.WriteString("package common")
	keyMap := make(map[string]*tool.Fail)
	codeMap := make(map[int32]*tool.Fail)
	for _, svc := range w.Builder().svcSlc {
		failSlc := make([]*tool.Fail, 0, len(svc.Fail))
		for _, fail := range svc.Fail {
			if f, ok := keyMap[fail.Key]; ok {
				if f.Code != fail.Code {
					return errors.New(fmt.Sprintf("key %s had %d and %d", f.Key, fail.Code, f.Code))
				}
				continue
			}
			if f, ok := codeMap[fail.Code]; ok {
				if f.Key != fail.Key {
					return errors.New(fmt.Sprintf("code %d had %s and %s", f.Code, fail.Key, f.Key))
				}
				continue
			}
			keyMap[fail.Key] = fail
			codeMap[fail.Code] = fail
			failSlc = append(failSlc, fail)
		}
		if len(failSlc) == 0 {
			continue
		}
		sort.Slice(failSlc, func(i, j int) bool {
			a, b := failSlc[i], failSlc[j]
			return a.Code < b.Code
		})
		bd.WriteString(fmt.Sprintf("\n\n// %s", svc.Name))
		bd.WriteString("\nconst (")
		for _, fail := range failSlc {
			bd.WriteString(fmt.Sprintf("\n\t//%s", fail.Msg))
			bd.WriteString(fmt.Sprintf("\n\tEc%s = %d", util.ToBigHump(fail.Key), fail.Code))
		}
		bd.WriteString("\n)")
	}
	return w.save("/common/fail.go", bd.String())
}

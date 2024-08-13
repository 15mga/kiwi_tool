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
	constBuilder := strings.Builder{}
	constBuilder.WriteString("package common")

	codeToStrBuilder := strings.Builder{}
	codeToStrBuilder.WriteString("\n\nfunc init() {")
	codeToStrBuilder.WriteString("\n\tutil.SetErrCodesToStrMap(map[util.TErrCode]string {")

	keyMap := make(map[string]*tool.Fail)
	codeMap := make(map[int32]*tool.Fail)
	for _, svc := range w.Builder().svcSlc {
		failSlc := make([]*tool.Fail, 0, len(svc.Fail))
		for _, fail := range svc.Fail {
			fail.Error = strings.ReplaceAll(fail.Error, "_", " ")
			if f, ok := keyMap[fail.Error]; ok {
				if f.Code != fail.Code {
					return errors.New(fmt.Sprintf("key %s had %d and %d", f.Error, fail.Code, f.Code))
				}
				continue
			}
			if f, ok := codeMap[fail.Code]; ok {
				if f.Error != fail.Error {
					return errors.New(fmt.Sprintf("code %d had %s and %s", f.Code, fail.Error, f.Error))
				}
				continue
			}
			keyMap[fail.Error] = fail
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
		constBuilder.WriteString(fmt.Sprintf("\n\n// %s %d", svc.Name, svc.Id))
		constBuilder.WriteString("\nconst (")
		for _, fail := range failSlc {
			constBuilder.WriteString(fmt.Sprintf("\n\t//%s", fail.Comment))
			constBuilder.WriteString(fmt.Sprintf("\n\tEc%s = %d", util.ToBigHump(fail.Error), fail.Code))

			codeToStrBuilder.WriteString(fmt.Sprintf("\n\t\t%d: \"%s\",", fail.Code, fail.Error))
		}
		constBuilder.WriteString("\n)")
	}

	codeToStrBuilder.WriteString("\n\t})")
	codeToStrBuilder.WriteString("\n}")
	return w.save("/common/fail.go", constBuilder.String()+codeToStrBuilder.String())
}

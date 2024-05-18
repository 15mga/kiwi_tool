package kiwi

import (
	"fmt"
	"github.com/15mga/kiwi/util"
	tool "github.com/15mga/kiwi_tool"
	"google.golang.org/protobuf/proto"
	"sort"
	"strings"
)

func newGFailCodeWriter() IWriter {
	return &gFailCodeWriter{}
}

type gFailCodeWriter struct {
	baseWriter
	keyToCode map[string]*tool.Fail
	failSlc   []*tool.Fail
}

func (w *gFailCodeWriter) Reset() {
	w.keyToCode = make(map[string]*tool.Fail)
}

func (w *gFailCodeWriter) SetSvc(svc *Svc) {
	w.baseWriter.SetSvc(svc)
	for _, file := range svc.Files {
		s := proto.GetExtension(file.Desc.Options(), tool.E_Svc).(*tool.Svc)
		slc := s.Fail
		if len(slc) == 0 {
			return
		}
		w.SetDirty(true)
		for _, fail := range slc {
			fail.Key = strings.TrimSpace(fail.Key)
			fail.Msg = strings.TrimSpace(fail.Msg)
			_, ok := w.keyToCode[fail.Key]
			if ok {
				continue
			}
			w.keyToCode[fail.Key] = fail
			w.failSlc = append(w.failSlc, fail)
		}
	}
}

//func (w *gFailCodeWriter) WriteMsg(idx int, msg *Msg) {
//	if msg.Type != EMsgRes {
//		return
//	}
//	slc := proto.GetExtension(msg.Msg.Desc.Options(), tool.E_Fail).([]*tool.Fail)
//	if len(slc) == 0 {
//		return
//	}
//	w.SetDirty(true)
//	for _, fail := range slc {
//		fail.Key = strings.TrimSpace(fail.Key)
//		fail.Msg = strings.TrimSpace(fail.Msg)
//		_, ok := w.keyToCode[fail.Key]
//		if ok {
//			continue
//		}
//		w.keyToCode[fail.Key] = fail
//		w.failSlc = append(w.failSlc, fail)
//	}
//}

func (w *gFailCodeWriter) Save() error {
	sort.Slice(w.failSlc, func(i, j int) bool {
		a, b := w.failSlc[i], w.failSlc[j]
		return a.Code < b.Code
	})
	bd := strings.Builder{}
	bd.WriteString("package common")
	bd.WriteString("\n\nconst (")
	for _, fail := range w.failSlc {
		bd.WriteString(fmt.Sprintf("\n\t//%s", fail.Msg))
		bd.WriteString(fmt.Sprintf("\n\tEc%s = %d", util.ToBigHump(fail.Key), fail.Code))
	}
	bd.WriteString("\n)")
	return w.save("/common/fail.go", bd.String())
}

type failData struct {
	key  string
	code int32
	msg  string
}

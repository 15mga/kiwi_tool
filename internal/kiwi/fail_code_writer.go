package kiwi

import (
	"fmt"
	"github.com/15mga/kiwi/util"
	tool "github.com/15mga/kiwi_tool"
	"google.golang.org/protobuf/proto"
	"strings"
)

func NewFailCodeWriter() IWriter {
	return &failCodeWriter{}
}

type failCodeWriter struct {
	baseWriter
	builder   *strings.Builder
	svcToFail map[string]map[string][]*tool.Fail
}

func (w *failCodeWriter) Reset() {
	w.builder = &strings.Builder{}
	w.svcToFail = make(map[string]map[string][]*tool.Fail)
}

func (w *failCodeWriter) WriteHeader() {
	w.builder.WriteString("package " + w.Svc().Name)
	w.builder.WriteString("\n\nconst (")
}

func (w *failCodeWriter) WriteMsg(idx int, msg *Msg) {
	if msg.Type != EMsgRes {
		return
	}
	slc := proto.GetExtension(msg.Msg.Desc.Options(), tool.E_Fail).([]*tool.Fail)
	if len(slc) == 0 {
		return
	}
	w.SetDirty(true)
	for _, fail := range slc {
		w.builder.WriteString(fmt.Sprintf("\n\t//%s", fail.Msg))
		w.builder.WriteString(fmt.Sprintf("\n\tEc%s_%s = %d", msg.Method, util.ToBigHump(fail.Key), fail.Code))
	}
}

func (w *failCodeWriter) WriteFooter() {
	w.builder.WriteString("\n)")
}

func (w *failCodeWriter) Save() error {
	path := fmt.Sprintf("%s/fail.go", w.Svc().Name)
	return w.save(path, w.builder.String())
}

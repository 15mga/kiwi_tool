package kiwi

import (
	"fmt"
	"sort"
	"strings"
)

func NewPusWriter() IWriter {
	return &pusWriter{}
}

type pusWriter struct {
	baseWriter
	builder *strings.Builder
	msgMap  map[string]*Msg
	msgSlc  []*Msg
}

func (w *pusWriter) Reset() {
	w.builder = &strings.Builder{}
	w.msgMap = make(map[string]*Msg, 8)
	w.SetDirty(true)
}

func (w *pusWriter) WriteHeader() {
	w.builder.WriteString(fmt.Sprintf("package %s", w.svc.Name))
	w.builder.WriteString("\n\nimport (")
	w.builder.WriteString(fmt.Sprintf("\n\t\"%s/proto/pb\"", w.Module()))
	w.builder.WriteString("\n\t\"github.com/15mga/kiwi\"")
	w.builder.WriteString("\n\t\"github.com/15mga/kiwi/util\"")
	w.builder.WriteString("\n)")
}

func (w *pusWriter) WriteMsg(idx int, msg *Msg) {
	if msg.Type != EMsgPus {
		return
	}

	w.msgMap[HandlerPrefix+msg.Method] = msg
}

func (w *pusWriter) Save() error {
	l := len(w.msgMap)
	if l == 0 {
		return nil
	}
	w.msgSlc = make([]*Msg, 0, l)
	for _, msg := range w.msgMap {
		w.msgSlc = append(w.msgSlc, msg)
	}

	svcName := w.svc.Name
	sort.Slice(w.msgSlc, func(i, j int) bool {
		return w.msgSlc[i].Code < w.msgSlc[j].Code
	})
	for _, msg := range w.msgSlc {
		w.builder.WriteString("\n")
		msgName := msg.Name
		shortName := msg.Method
		methodName := fmt.Sprintf("%s%s", HandlerPrefix, shortName)
		w.builder.WriteString(fmt.Sprintf("\nfunc (s *svc) %s(pkt kiwi.IRcvPush, pus *pb.%s) {",
			methodName, msgName))
		w.builder.WriteString("\n\tpkt.Err2(util.EcNotImplement, util.M{\"pus\": pus})")
		w.builder.WriteString("\n}")
	}
	fp := fmt.Sprintf("%s/pus_gen.go", svcName)
	return w.save(fp, w.builder.String())
}

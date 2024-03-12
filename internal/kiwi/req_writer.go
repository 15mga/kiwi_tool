package kiwi

import (
	"fmt"
	"sort"
	"strings"
)

func NewReqWriter() IWriter {
	return &reqWriter{}
}

type reqWriter struct {
	baseWriter
	builder *strings.Builder
	msgMap  map[string]*Msg
	msgSlc  []*Msg
}

func (w *reqWriter) Reset() {
	w.builder = &strings.Builder{}
	w.msgMap = make(map[string]*Msg, 8)
	w.SetDirty(true)
}

func (w *reqWriter) WriteMsg(idx int, msg *Msg) {
	if msg.Type != EMsgReq {
		return
	}

	w.msgMap[HandlerPrefix+msg.Method] = msg
}

func (w *reqWriter) WriteHeader() {
	w.builder.WriteString(fmt.Sprintf("package %s", w.svc.Name))
	w.builder.WriteString("\n\nimport (")
	w.builder.WriteString(fmt.Sprintf("\n\t\"%s/proto/pb\"", w.Module()))
	w.builder.WriteString("\n\t\"github.com/15mga/kiwi\"")
	w.builder.WriteString("\n\t\"github.com/15mga/kiwi/util\"")
	w.builder.WriteString("\n)")
}

func (w *reqWriter) Save() error {
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
		w.builder.WriteString(fmt.Sprintf("\nfunc (s *svc) %s(pkt kiwi.IRcvRequest, req *pb.%s, res *pb.%s) {",
			methodName, msgName, reqToRes(msgName)))
		w.builder.WriteString("\n\tpkt.Err2(util.EcNotImplement, util.M{\"req\": req})")
		w.builder.WriteString("\n}")
	}
	fp := fmt.Sprintf("%s/req_gen.go", svcName)
	return w.save(fp, w.builder.String())
}

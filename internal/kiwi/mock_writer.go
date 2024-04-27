package kiwi

import (
	"fmt"
	"strings"
)

func NewMockWriter() IWriter {
	return &mockWriter{}
}

type mockWriter struct {
	baseWriter
	builder       *strings.Builder
	handleBuilder *strings.Builder
	msgSlc        []*Msg
}

func (w *mockWriter) Reset() {
	w.builder = &strings.Builder{}
	w.handleBuilder = &strings.Builder{}
	w.msgSlc = nil
	w.SetDirty(true)
}

func (w *mockWriter) WriteMsg(idx int, msg *Msg) {
	if msg.Type == EMsgReq || msg.Type == EMsgRes || msg.Type == EMsgPus {
		w.msgSlc = append(w.msgSlc, msg)
	}
}

func (w *mockWriter) Save() error {
	l := len(w.msgSlc)
	if l == 0 {
		return nil
	}

	svcName := w.svc.Name
	w.builder.WriteString(fmt.Sprintf("package %s", svcName))
	w.builder.WriteString("\n\nimport (")
	w.builder.WriteString(fmt.Sprintf("\n\t\"%s/proto/pb\"", w.Builder().module))
	w.builder.WriteString(fmt.Sprintf("\n\t\"%s/internal/common\"", w.Builder().module))
	w.builder.WriteString("\n\t\"github.com/15mga/kiwi\"")
	w.builder.WriteString("\n\t\"github.com/15mga/kiwi/graph\"")
	w.builder.WriteString("\n\t\"github.com/15mga/kiwi/mock\"")
	w.builder.WriteString("\n\t\"github.com/15mga/kiwi/util\"")
	w.builder.WriteString(fmt.Sprintf("\n\t\"%s/internal/%s\"", w.Builder().module, svcName))
	w.builder.WriteString("\n)")
	w.builder.WriteString("\n\ntype Svc struct {")
	w.builder.WriteString("\n\tsvc")
	w.builder.WriteString("\n}")
	w.builder.WriteString("\n\ntype svc struct {")
	w.builder.WriteString("\nclient *mock.Client")
	w.builder.WriteString("\n}")
	w.builder.WriteString("\n\nfunc Init(client *mock.Client) {")
	w.builder.WriteString(fmt.Sprintf("\n%s.BindCodecFac()", svcName))
	w.builder.WriteString(fmt.Sprintf("\n%s.BindReqToRes()", svcName))
	w.builder.WriteString("\n\ts := &Svc{svc{client: client}}")
	for _, msg := range w.msgSlc {
		switch msg.Type {
		case EMsgReq:
			w.builder.WriteString(fmt.Sprintf("\n\ts.client.BindPointMsg(\"%s\", \"%s\", s.in%s)", msg.Svc.Name, msg.Method, msg.Name))
			w.handleBuilder.WriteString(fmt.Sprintf("\n\nfunc (s *svc) in%s(msg graph.IMsg) *util.Err {", msg.Name))
			w.handleBuilder.WriteString(fmt.Sprintf("\n\treq, ok := util.MGet[*pb.%s](s.client.Graph().Data(), \"%s\")", msg.Name, msg.Method))
			w.handleBuilder.WriteString("\n\tif !ok {")
			w.handleBuilder.WriteString(fmt.Sprintf("\n\t\treq = &pb.%s{}", msg.Name))
			w.handleBuilder.WriteString("\n\t}")
			w.handleBuilder.WriteString(fmt.Sprintf("\n\tfn, ok := util.MGet[func(*mock.Client, util.IMsg)](s.client.Graph().Data(), \"%sDecorator\")", msg.Method))
			w.handleBuilder.WriteString("\n\tif ok && fn != nil {")
			w.handleBuilder.WriteString("\n\t\tfn(s.client, req)")
			w.handleBuilder.WriteString("\n\t}")
			w.handleBuilder.WriteString("\n\treturn s.Req(req)")
			w.handleBuilder.WriteString("\n}")
		case EMsgRes:
			w.builder.WriteString(fmt.Sprintf("\n\ts.client.BindNetMsg(&pb.%s{}, s.on%s)", msg.Name, msg.Name))
			w.handleBuilder.WriteString(fmt.Sprintf("\n\nfunc (s *svc) on%s(msg util.IMsg) (point string, data any) {", msg.Name))
			w.handleBuilder.WriteString(fmt.Sprintf("\n\ts.client.Graph().Data().Set(\"%s\", msg)", msg.Name))
			w.handleBuilder.WriteString(fmt.Sprintf("\n\treturn \"%s\", nil", msg.Method))
			w.handleBuilder.WriteString("\n}")
		case EMsgPus:
			w.builder.WriteString(fmt.Sprintf("\n\ts.client.BindNetMsg(&pb.%s{}, s.on%s)", msg.Name, msg.Name))
			w.handleBuilder.WriteString(fmt.Sprintf("\n\nfunc (s *svc) on%s(msg util.IMsg) (point string, data any) {", msg.Name))
			w.handleBuilder.WriteString(fmt.Sprintf("\n\tkiwi.Debug(\"on %s\", util.M{\"msg\":msg})", msg.Name))
			w.handleBuilder.WriteString("\n\treturn \"\", nil")
			w.handleBuilder.WriteString("\n}")
		}
	}
	w.builder.WriteString("\n}")
	w.builder.WriteString("\n\nfunc (s *svc) Dispose() {")
	w.builder.WriteString("\n}")
	w.builder.WriteString("\n\nfunc (s *svc) Req(req util.IMsg) *util.Err {")
	w.builder.WriteString("\n\tkiwi.Debug(\"request\", util.M{string(req.ProtoReflect().Descriptor().Name()):req})")
	w.builder.WriteString("\n\tsvc, code := kiwi.Codec().MsgToSvcCode(req)")
	w.builder.WriteString("\n\tbytes, err := common.PackUserReq(svc, code, req)")
	w.builder.WriteString("\n\tif err != nil {")
	w.builder.WriteString("\n\t\treturn err")
	w.builder.WriteString("\n\t}")
	w.builder.WriteString("\n\treturn s.client.Dialer().Agent().Send(bytes)")
	w.builder.WriteString("\n}")

	fp := fmt.Sprintf("mock/%s/svc_gen.go", svcName)
	return w.save(fp, w.builder.String()+
		w.handleBuilder.String(),
	)
}

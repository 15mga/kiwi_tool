package kiwi

import (
	"fmt"
	"github.com/15mga/kiwi/util"
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

func (w *mockWriter) WriteMsg(idx int, msg *Msg) error {
	if msg.Type == EMsgReq || msg.Type == EMsgRes || msg.Type == EMsgPus {
		w.msgSlc = append(w.msgSlc, msg)
	}
	return nil
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
	w.builder.WriteString(fmt.Sprintf("\n\t\"%s/internal/codec\"", w.Builder().module))
	w.builder.WriteString("\n\t\"github.com/15mga/kiwi\"")
	w.builder.WriteString("\n\t\"github.com/15mga/kiwi/graph\"")
	w.builder.WriteString("\n\t\"github.com/15mga/kiwi/mock\"")
	w.builder.WriteString("\n\t\"github.com/15mga/kiwi/util\"")
	w.builder.WriteString("\n\t\"strconv\"")
	w.builder.WriteString("\n)")
	w.builder.WriteString("\n\ntype Svc struct {")
	w.builder.WriteString("\n\tsvc")
	w.builder.WriteString("\n}")
	w.builder.WriteString("\n\ntype svc struct {")
	w.builder.WriteString("\nclient *mock.Client")
	w.builder.WriteString("\n}")
	w.builder.WriteString("\n\nfunc InitClient(client *mock.Client) {")
	w.builder.WriteString("\n\ts := &Svc{svc{client: client}}")
	for _, msg := range w.msgSlc {
		switch msg.Type {
		case EMsgReq:
			w.builder.WriteString(fmt.Sprintf("\n\ts.client.BindPointMsg(\"%s\", \"%s\", s.in%s)", msg.Svc.Name, msg.MethodName, msg.MsgName))
			w.handleBuilder.WriteString(fmt.Sprintf("\n\nfunc (s *svc) in%s(msg graph.IMsg) *util.Err {", msg.MsgName))
			w.handleBuilder.WriteString(fmt.Sprintf("\n\treq := s.client.GetRequest(common.%s, codec.%s)", util.ToBigHump(msg.Svc.Name), msg.MsgName))
			w.handleBuilder.WriteString("\n\treturn s.Req(req)")
			w.handleBuilder.WriteString("\n}")
		case EMsgRes:
			w.builder.WriteString(fmt.Sprintf("\n\ts.client.BindNetMsg(&pb.%s{}, s.on%s)", msg.MsgName, msg.MsgName))
			w.handleBuilder.WriteString(fmt.Sprintf("\n\nfunc (s *svc) on%s(msg util.IMsg) (point string, data any) {", msg.MsgName))
			w.handleBuilder.WriteString("\n\tsc := kiwi.MergeSvcCode(kiwi.Codec().MsgToSvcCode(msg))")
			w.handleBuilder.WriteString("\n\ts.client.Graph().Data().Set(strconv.Itoa(int(sc)), msg)")
			w.handleBuilder.WriteString(fmt.Sprintf("\n\treturn \"%s\", nil", msg.MethodName))
			w.handleBuilder.WriteString("\n}")
		case EMsgPus:
			w.builder.WriteString(fmt.Sprintf("\n\ts.client.BindNetMsg(&pb.%s{}, s.on%s)", msg.MsgName, msg.MsgName))
			w.handleBuilder.WriteString(fmt.Sprintf("\n\nfunc (s *svc) on%s(msg util.IMsg) (point string, data any) {", msg.MsgName))
			w.handleBuilder.WriteString("\n\tsc := kiwi.MergeSvcCode(kiwi.Codec().MsgToSvcCode(msg))")
			w.handleBuilder.WriteString("\n\ts.client.Graph().Data().Set(strconv.Itoa(int(sc)), msg)")
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

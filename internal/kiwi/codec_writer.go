package kiwi

import (
	"fmt"
	"strings"

	"github.com/15mga/kiwi/util"
)

func NewCodecWriter() IWriter {
	return &codecWriter{}
}

type codecWriter struct {
	baseWriter
	constBuilder *strings.Builder
	facBuilder   *strings.Builder
}

func (w *codecWriter) Reset() {
	w.constBuilder = &strings.Builder{}
	w.facBuilder = &strings.Builder{}
}

func (w *codecWriter) WriteHeader() {
	w.constBuilder.WriteString(fmt.Sprintf("package %s", w.Svc().Name))
	w.constBuilder.WriteString("\n\nimport (")
	w.constBuilder.WriteString(fmt.Sprintf("\n\t\"%s/internal/common\"", w.Module()))
	w.constBuilder.WriteString(fmt.Sprintf("\n\t\"%s/proto/pb\"", w.Module()))
	w.constBuilder.WriteString("\n\n\t\"github.com/15mga/kiwi\"")
	w.constBuilder.WriteString("\n\t\"github.com/15mga/kiwi/util\"")
	w.constBuilder.WriteString("\n)")
	w.constBuilder.WriteString("\n\nconst (")

	w.facBuilder.WriteString("\n\nfunc (svc *svc) bindCodecFac() {")
}

func (w *codecWriter) WriteMsg(idx int, msg *Msg) {
	w.SetDirty(true)
	if msg.Type != EMsgPus &&
		msg.Type != EMsgReq &&
		msg.Type != EMsgRes &&
		msg.Type != EMsgNtc {
		return
	}
	w.constBuilder.WriteString(fmt.Sprintf("\n%s", msg.Msg.Comments.Leading.String()))
	w.constBuilder.WriteString(fmt.Sprintf("\t%s kiwi.TCode = %d", msg.Name, msg.Code))

	svcName := util.ToBigHump(msg.Svc.Name)
	w.facBuilder.WriteString(fmt.Sprintf("\n\tkiwi.Codec().BindFac(common.%s, %s, func() util.IMsg {",
		svcName, msg.Name))
	w.facBuilder.WriteString(fmt.Sprintf("\n\t\treturn &pb.%s{}", msg.Name))
	w.facBuilder.WriteString("\n\t})")
}

func (w *codecWriter) WriteFooter() {
	w.constBuilder.WriteString("\n)")

	w.facBuilder.WriteString("\n}")
}

func (w *codecWriter) Save() error {
	path := fmt.Sprintf("/%s/codec.go", w.Svc().Name)
	return w.save(path, w.constBuilder.String()+w.facBuilder.String())
}

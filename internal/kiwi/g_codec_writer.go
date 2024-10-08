package kiwi

import (
	"fmt"
	"github.com/15mga/kiwi/util"
	"strings"
)

func newGCodeWriter() IWriter {
	return &gCodecWriter{}
}

type gCodecWriter struct {
	baseWriter
}

func (w *gCodecWriter) Reset() {
	w.SetDirty(true)
}

func (w *gCodecWriter) Save() error {
	headBd := strings.Builder{}
	headBd.WriteString("package codec")
	headBd.WriteString("\n\nimport (")
	headBd.WriteString("\n\t\"github.com/15mga/kiwi\"")
	headBd.WriteString("\n\t\"github.com/15mga/kiwi/util\"")
	headBd.WriteString(fmt.Sprintf("\n\t\"%s/internal/common\"", w.Module()))
	headBd.WriteString(fmt.Sprintf("\n\t\"%s/proto/pb\"", w.Module()))

	contentBd := strings.Builder{}
	contentBd.WriteString("\n\nfunc BindPool() {")
	for _, svc := range w.Builder().svcSlc {
		if svc.IsCommonSvc() {
			continue
		}

		svcName := util.ToBigHump(svc.Name)
		for _, msg := range svc.MsgSlc {
			if msg.Type != EMsgPus &&
				msg.Type != EMsgReq &&
				msg.Type != EMsgRes &&
				msg.Type != EMsgNtc {
				continue
			}
			contentBd.WriteString(fmt.Sprintf("\n\tkiwi.Codec().BindPool(common.%s, %s, func() util.IMsg {",
				svcName, msg.MsgName))
			contentBd.WriteString(fmt.Sprintf("\n\t\treturn &pb.%s{}", msg.MsgName))
			contentBd.WriteString("\n\t})")
		}
	}
	headBd.WriteString("\n)")
	contentBd.WriteString("\n}")

	return w.save("/codec/fac.go", headBd.String()+contentBd.String())
}

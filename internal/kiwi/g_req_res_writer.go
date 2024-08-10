package kiwi

import (
	"fmt"
	"github.com/15mga/kiwi/util"
	"strings"
)

func newGReqResWriter() IWriter {
	return &gReqResWriter{}
}

type gReqResWriter struct {
	baseWriter
}

func (w *gReqResWriter) Reset() {
	w.SetDirty(true)
}

func (w *gReqResWriter) Save() error {
	headBd := strings.Builder{}
	headBd.WriteString("package codec")
	headBd.WriteString("\n\nimport (")
	headBd.WriteString("\n\t\"github.com/15mga/kiwi\"")
	headBd.WriteString(fmt.Sprintf("\n\t\"%s/internal/common\"", w.Module()))

	contentBd := strings.Builder{}
	contentBd.WriteString("\n\nfunc BindReqToRes() {")
	for _, svc := range w.Builder().svcSlc {
		if svc.IsCommonSvc() {
			continue
		}

		for _, msg := range svc.MsgSlc {
			if msg.Type != EMsgReq {
				continue
			}
			reqName := msg.MsgName
			resName := reqToRes(reqName)
			_, ok := svc.Res[resName]
			if !ok {
				continue
			}
			contentBd.WriteString(fmt.Sprintf("\n\tkiwi.Codec().BindReqToRes(common.%s, %s, %s)",
				util.ToBigHump(svc.Name), reqName, resName))
		}
	}
	headBd.WriteString("\n)")
	contentBd.WriteString("\n}")

	return w.save("/codec/req_res.go", headBd.String()+contentBd.String())
}

package kiwi

import (
	"fmt"
	"github.com/15mga/kiwi/util"
	"strings"
)

func NewSvcWriter() IWriter {
	return &svcWriter{}
}

type svcWriter struct {
	baseWriter
	svcBuilder *strings.Builder
	msgSlc     []*Msg
}

func (w *svcWriter) Reset() {
	w.svcBuilder = &strings.Builder{}
	w.msgSlc = nil
	w.SetDirty(true)
}

func (w *svcWriter) WriteMsg(idx int, msg *Msg) error {
	w.msgSlc = append(w.msgSlc, msg)
	return nil
}

func (w *svcWriter) hasSchema() bool {
	for _, msg := range w.msgSlc {
		if msg.Type == EMsgSch {
			return true
		}
	}
	return false
}

func (w *svcWriter) Save() error {
	l := len(w.msgSlc)
	if l == 0 {
		return nil
	}

	svcName := w.svc.Name
	w.svcBuilder.WriteString(fmt.Sprintf("package %s", svcName))
	w.svcBuilder.WriteString("\n\nimport (")
	w.svcBuilder.WriteString(fmt.Sprintf("\n\t\"%s/internal/common\"", w.Builder().module))
	w.svcBuilder.WriteString("\n\t\"github.com/15mga/kiwi\"")
	w.svcBuilder.WriteString("\n\t\"github.com/15mga/kiwi/core\"")
	w.svcBuilder.WriteString("\n)")
	w.svcBuilder.WriteString("\n\nvar (")
	w.svcBuilder.WriteString("\n\t_svc *Svc")
	w.svcBuilder.WriteString("\n)")
	w.svcBuilder.WriteString("\n\nfunc New(ver string) kiwi.IService {")
	w.svcBuilder.WriteString("\n\t_svc = &Svc{")
	w.svcBuilder.WriteString("\n\t\tsvc: svc{")
	w.svcBuilder.WriteString(fmt.Sprintf("\n\t\t\tService: core.NewService(common.%s, ver),", util.ToBigHump(svcName)))
	w.svcBuilder.WriteString("\n\t\t}}")
	w.svcBuilder.WriteString("\n\treturn _svc")
	w.svcBuilder.WriteString("\n}")
	w.svcBuilder.WriteString("\n\ntype Svc struct {")
	w.svcBuilder.WriteString("\n\tsvc")
	w.svcBuilder.WriteString("\n}")
	w.svcBuilder.WriteString("\n\ntype svc struct {")
	w.svcBuilder.WriteString("\n\tcore.Service")
	w.svcBuilder.WriteString("\n}")
	w.svcBuilder.WriteString("\n\nfunc (s *svc) Start() {")
	if w.hasSchema() {
		w.svcBuilder.WriteString("\n\tinitColl()")
	}
	w.svcBuilder.WriteString("\n\tregisterReq()")
	w.svcBuilder.WriteString("\n}")
	w.svcBuilder.WriteString("\n\nfunc (s *svc) AfterStart() {")
	if len(w.svc.WatchNtc) > 0 {
		w.svcBuilder.WriteString("\n\twatchNtc()")
	}
	w.svcBuilder.WriteString("\n}")

	fp := fmt.Sprintf("%s/svc_gen.go", svcName)
	return w.save(fp, w.svcBuilder.String())
}

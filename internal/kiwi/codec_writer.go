package kiwi

import (
	"fmt"
	"strings"
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
	w.constBuilder.WriteString(fmt.Sprintf("package codec"))
	w.constBuilder.WriteString("\n\nimport (")
	w.constBuilder.WriteString("\n\n\t\"github.com/15mga/kiwi\"")
	w.constBuilder.WriteString("\n)")
	w.constBuilder.WriteString("\n\nconst (")
}

func (w *codecWriter) WriteMsg(idx int, msg *Msg) error {
	w.SetDirty(true)
	if msg.Type != EMsgPus &&
		msg.Type != EMsgReq &&
		msg.Type != EMsgRes &&
		msg.Type != EMsgNtc {
		return nil
	}
	w.constBuilder.WriteString(fmt.Sprintf("\n%s", msg.Msg.Comments.Leading.String()))
	w.constBuilder.WriteString(fmt.Sprintf("\t%s kiwi.TCode = %d", msg.MsgName, msg.MethodCode))
	return nil
}

func (w *codecWriter) WriteFooter() {
	w.constBuilder.WriteString("\n)")
}

func (w *codecWriter) Save() error {
	path := fmt.Sprintf("/codec/%s.go", w.Svc().Name)
	return w.save(path, w.constBuilder.String()+w.facBuilder.String())
}

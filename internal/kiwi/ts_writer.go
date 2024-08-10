package kiwi

import (
	"fmt"
	"strings"
)

func NewTsWriter() IWriter {
	return &tsWriter{}
}

type tsWriter struct {
	baseWriter
	builder *strings.Builder
}

func (w *tsWriter) Reset() {
	w.builder = &strings.Builder{}
}

func (w *tsWriter) WriteMsg(idx int, msg *Msg) error {
	if msg.Type != EMsgReq && msg.Type != EMsgRes && msg.Type != EMsgPus {
		return nil
	}
	w.SetDirty(true)
	w.builder.WriteString(fmt.Sprintf("\n\nexport class %s }", msg.MsgName))
	//for _, field := range msg.Msg.Fields {
	//field.
	//}
	w.builder.WriteString("}")
	return nil
}

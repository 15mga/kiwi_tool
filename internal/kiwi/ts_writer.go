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

func (w *tsWriter) WriteMsg(idx int, msg *Msg) {
	if msg.Type != EMsgReq && msg.Type != EMsgRes && msg.Type != EMsgPus {
		return
	}
	w.SetDirty(true)
	w.builder.WriteString(fmt.Sprintf("\n\nexport class %s }", msg.Name))
	//for _, field := range msg.Msg.Fields {
	//field.
	//}
	w.builder.WriteString("}")
}

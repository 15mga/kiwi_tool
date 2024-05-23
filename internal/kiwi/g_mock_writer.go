package kiwi

import (
	"fmt"
	"strings"
)

func NewGMockWriter() IWriter {
	return &gMockWriter{}
}

type gMockWriter struct {
	baseWriter
	builder *strings.Builder
	svcSlc  []*svc
}

func (w *gMockWriter) Reset() {
	w.builder = &strings.Builder{}
	w.svcSlc = nil
}

func (w *gMockWriter) SetSvc(svc *svc) {
	w.dirty = true
	w.svcSlc = append(w.svcSlc, svc)
}

func (w *gMockWriter) Save() error {
	w.builder.WriteString("package mock")
	w.builder.WriteString("\n\nimport (")
	w.builder.WriteString("\n\"github.com/15mga/kiwi/mock\"")
	for _, svc := range w.svcSlc {
		w.builder.WriteString(fmt.Sprintf("\n\t\"%s/internal/mock/%s\"", w.Module(), svc.Name))
	}
	w.builder.WriteString("\n)")

	w.builder.WriteString("\n\nfunc initCodec() {")
	for _, svc := range w.svcSlc {
		w.builder.WriteString(fmt.Sprintf("\n\t%s.Init()", svc.Name))
	}
	w.builder.WriteString("\n}")

	w.builder.WriteString("\n\nfunc initClient(client *mock.Client) {")
	for _, svc := range w.svcSlc {
		w.builder.WriteString(fmt.Sprintf("\n\t%s.InitClient(client)", svc.Name))
	}
	w.builder.WriteString("\n}")

	path := fmt.Sprintf("mock/init.go")
	return w.save(path, w.builder.String())
}

package kiwi

import (
	"fmt"
	"strings"

	"github.com/15mga/kiwi/util"
)

func NewReqResWriter() IWriter {
	return &reqResWriter{}
}

type reqResWriter struct {
	baseWriter
	builder *strings.Builder
}

func (w *reqResWriter) Reset() {
	w.builder = &strings.Builder{}
	w.dirty = true
}

func (w *reqResWriter) WriteHeader() {
	w.builder.WriteString("package " + w.Svc().Name)
	w.builder.WriteString("\n\nimport (")
	w.builder.WriteString(fmt.Sprintf("\n\t\"%s/internal/common\"", w.Module()))
	w.builder.WriteString("\n\n\t\"github.com/15mga/kiwi\"")
	w.builder.WriteString("\n)")
	w.builder.WriteString("\n\nfunc BindReqToRes() {")
}

func (w *reqResWriter) WriteMsg(idx int, msg *Msg) {
	if msg.Type != EMsgReq {
		return
	}
	reqName := msg.Name
	resName := reqToRes(reqName)
	_, ok := w.svc.Res[resName]
	if !ok {
		fmt.Printf("not exist res: %s\n", resName)
		return
	}
	w.write(reqName, resName)
}

func (w *reqResWriter) write(req, res string) {
	w.builder.WriteString(fmt.Sprintf("\n\tkiwi.Codec().BindReqToRes(common.%s, %s, %s)",
		util.ToBigHump(w.Svc().Name), req, res))
}

func (w *reqResWriter) WriteFooter() {
	w.builder.WriteString("\n}")
}

func (w *reqResWriter) Save() error {
	path := fmt.Sprintf("%s/req_res.go", w.Svc().Name)
	return w.save(path, w.builder.String())
}

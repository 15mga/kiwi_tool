package kiwi

import (
	"fmt"
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/util"
	tool "github.com/15mga/kiwi_tool"
	"google.golang.org/protobuf/proto"
	"strings"
)

func NewRoleWriter() IWriter {
	return &roleWriter{}
}

type roleWriter struct {
	baseWriter
	roleBuilder *strings.Builder
}

func (w *roleWriter) Reset() {
	w.roleBuilder = &strings.Builder{}
}

func (w *roleWriter) WriteHeader() {
	w.roleBuilder.WriteString(fmt.Sprintf("package %s", w.Svc().Name))
	w.roleBuilder.WriteString("\n\nimport (")
	w.roleBuilder.WriteString(fmt.Sprintf("\n\t\"%s/internal/common\"", w.Module()))
	w.roleBuilder.WriteString("\n\t\"github.com/15mga/kiwi\"")
	w.roleBuilder.WriteString("\n\t\"github.com/15mga/kiwi/util\"")
	w.roleBuilder.WriteString("\n)")
	w.roleBuilder.WriteString("\n\n\tvar MsgRole = map[kiwi.TSvcMethod]int64 {")
}

func (w *roleWriter) WriteFooter() {
	w.roleBuilder.WriteString("\n}")
}

func (w *roleWriter) WriteMsg(idx int, msg *Msg) error {
	if msg.Type != EMsgReq {
		return nil
	}
	roleSlc := proto.GetExtension(msg.Msg.Desc.Options(), tool.E_Role).([]string)
	if len(roleSlc) == 0 {
		return nil
	}
	slc := make([]string, 0, len(roleSlc))
	for _, role := range roleSlc {
		bigRole := "common.R" + util.ToBigHump(role)
		slc = append(slc, bigRole)
	}
	w.SetDirty(true)
	w.roleBuilder.WriteString(fmt.Sprintf("\n\t%d: util.GenMask(%s),",
		kiwi.MergeSvcMethod(msg.Svc.Id, msg.MethodCode), strings.Join(slc, ", ")))
	return nil
}

func (w *roleWriter) Save() error {
	path := fmt.Sprintf("%s/role.go", w.Svc().Name)
	return w.save(path, w.roleBuilder.String())
}

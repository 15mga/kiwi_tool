package kiwi

import (
	"fmt"
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/util"
	tool "github.com/15mga/kiwi_tool"
	"google.golang.org/protobuf/proto"
	"strings"
)

func NewGRoleWriter() IWriter {
	return &gRoleWriter{}
}

type gRoleWriter struct {
	baseWriter
	roleNames        map[string]struct{}
	headBuilder      *strings.Builder
	footBuilder      *strings.Builder
	roleToStrBuilder *strings.Builder
	strToRoleBuilder *strings.Builder
	roleBuilder      *strings.Builder
}

func (w *gRoleWriter) Reset() {
	w.roleNames = make(map[string]struct{})
	w.headBuilder = &strings.Builder{}
	w.footBuilder = &strings.Builder{}
	w.roleToStrBuilder = &strings.Builder{}
	w.strToRoleBuilder = &strings.Builder{}
	w.roleBuilder = &strings.Builder{}
}

func (w *gRoleWriter) WriteMsg(idx int, msg *Msg) {
	if msg.Type != EMsgReq {
		return
	}
	roleSlc := proto.GetExtension(msg.Msg.Desc.Options(), tool.E_Role).([]string)
	slc := make([]string, 0, len(roleSlc))
	for _, role := range roleSlc {
		w.SetDirty(true)
		_, ok := w.roleNames[role]
		bigRole := "R" + util.ToBigHump(role)
		if !ok {
			w.roleNames[role] = struct{}{}
			w.roleToStrBuilder.WriteString(fmt.Sprintf("\n\t\tcase %s:", bigRole))
			w.roleToStrBuilder.WriteString(fmt.Sprintf("\n\t\treturn \"%s\"", role))
			w.strToRoleBuilder.WriteString(fmt.Sprintf("\n\t\tcase \"%s\": ", role))
			w.strToRoleBuilder.WriteString(fmt.Sprintf("\n\t\t\treturn %s", bigRole))
		}
		slc = append(slc, bigRole)
	}
	if len(slc) > 0 {
		w.roleBuilder.WriteString(fmt.Sprintf("\n\t%d: {%s},",
			kiwi.MergeSvcCode(msg.Svc.Id, msg.Code), strings.Join(slc, ", ")))
	}
}

func (w *gRoleWriter) WriteHeader() {
	w.headBuilder.WriteString("package common")
	w.headBuilder.WriteString("\n\nimport (")
	w.headBuilder.WriteString("\n\t\"github.com/15mga/kiwi\"")
	w.headBuilder.WriteString("\n)")
	w.roleToStrBuilder.WriteString("\n\n\tfunc RoleToStr(role int64) string {")
	w.roleToStrBuilder.WriteString("\n\t\tswitch role {")
	w.strToRoleBuilder.WriteString("\n\n\tfunc StrToRole(role string) int64 {")
	w.strToRoleBuilder.WriteString("\n\t\tswitch role {")
	w.roleBuilder.WriteString("\n\n\tvar MsgRole = map[kiwi.TSvcCode][]int64 {")
}

func (w *gRoleWriter) WriteFooter() {
	w.roleToStrBuilder.WriteString("\n\tdefault:")
	w.roleToStrBuilder.WriteString("\n\t\treturn \"\"")
	w.roleToStrBuilder.WriteString("\n\t}")
	w.roleToStrBuilder.WriteString("\n}")
	w.strToRoleBuilder.WriteString("\n\tdefault:")
	w.strToRoleBuilder.WriteString("\n\t\treturn 0")
	w.strToRoleBuilder.WriteString("\n\t}")
	w.strToRoleBuilder.WriteString("\n}")
	w.roleBuilder.WriteString("\n}")
}

func (w *gRoleWriter) Save() error {
	return w.save("/common/role_gen.go",
		w.headBuilder.String()+
			w.roleToStrBuilder.String()+
			w.strToRoleBuilder.String()+
			w.footBuilder.String()+
			w.roleBuilder.String())
}

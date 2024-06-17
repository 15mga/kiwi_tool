package kiwi

import (
	"fmt"
	"github.com/15mga/kiwi"
	tool "github.com/15mga/kiwi_tool"
	"google.golang.org/protobuf/proto"
	"strings"
)

func NewGCsWriter() IWriter {
	return &gCsWriter{}
}

type gCsWriter struct {
	baseWriter
	header           *strings.Builder
	footer           *strings.Builder
	typeToCodeHeader *strings.Builder
	typeToCodeFooter *strings.Builder
	codeToTypeHeader *strings.Builder
	codeToTypeFooter *strings.Builder
	reqResHeader     *strings.Builder
	reqResFooter     *strings.Builder
}

func (w *gCsWriter) Reset() {
	w.header = &strings.Builder{}
	w.footer = &strings.Builder{}
	w.typeToCodeHeader = &strings.Builder{}
	w.typeToCodeFooter = &strings.Builder{}
	w.codeToTypeHeader = &strings.Builder{}
	w.codeToTypeFooter = &strings.Builder{}
	w.reqResHeader = &strings.Builder{}
	w.reqResFooter = &strings.Builder{}
	w.SetDirty(true)
}

func (w *gCsWriter) WriteHeader() {
	w.header.WriteString("using System;")
	w.header.WriteString("\nusing System.Collections.Generic;")
	w.header.WriteString("\nusing Google.Protobuf;")
	w.header.WriteString("\n\nnamespace Pb")
	w.header.WriteString("\n{")
	w.header.WriteString("\n\tpublic static class Code")
	w.header.WriteString("\n\t{")
	w.typeToCodeHeader.WriteString("\n\t\tpublic static readonly Dictionary<Type, ushort> MsgTypeToCode = new()")
	w.typeToCodeHeader.WriteString("\n\t\t{")
	w.codeToTypeHeader.WriteString("\n\n\t\tpublic static readonly Dictionary<ushort, Func<byte[], IMessage>> CodeToMsg = new()")
	w.codeToTypeHeader.WriteString("\n\t\t{")
	w.reqResHeader.WriteString("\n\n\t\tpublic static readonly Dictionary<ushort, ushort> ReqToRes = new()")
	w.reqResHeader.WriteString("\n\t\t{")
}

func (w *gCsWriter) WriteFooter() {
	w.typeToCodeFooter.WriteString("\n\t\t};")
	w.codeToTypeFooter.WriteString("\n\t\t};")
	w.reqResFooter.WriteString("\n\t\t};")
	w.footer.WriteString("\n\t}")
	w.footer.WriteString("\n}")
}

func (w *gCsWriter) isPlayerMsg(msg *Msg) bool {
	roleSlc := proto.GetExtension(msg.Msg.Desc.Options(), tool.E_Role).([]string)
	for _, role := range roleSlc {
		if w.Builder().isPlayerRole(role) {
			return true
		}
	}
	return false
}

func (w *gCsWriter) WriteMsg(idx int, msg *Msg) error {
	switch msg.Type {
	case EMsgReq:
		if !w.isPlayerMsg(msg) {
			return nil
		}
		reqCode := kiwi.MergeSvcCode(msg.Svc.Id, msg.Code)
		w.typeToCodeHeader.WriteString(fmt.Sprintf("\n\t\t\t{typeof(%s), %d},",
			msg.Name, reqCode))
	case EMsgRes:
		resName := msg.Name
		reqName := resToReq(resName)
		reqMsg, ok := w.Builder().msgMap[reqName]
		if !ok || !w.isPlayerMsg(reqMsg) {
			return nil
		}
		resCode := kiwi.MergeSvcCode(msg.Svc.Id, msg.Code)
		w.codeToTypeHeader.WriteString(fmt.Sprintf("\n\t\t\t{%d, %s.Parser.ParseFrom},",
			resCode, msg.Name))
		_, ok = w.svc.Req[reqName]
		if ok {
			req := w.svc.Req[reqName]
			reqCode := kiwi.MergeSvcCode(req.Svc.Id, req.Code)
			w.reqResHeader.WriteString(fmt.Sprintf("\n\t\t\t{%d, %d},",
				reqCode, resCode))
		}
	case EMsgPus:
		ntcCode := kiwi.MergeSvcCode(msg.Svc.Id, msg.Code)
		w.typeToCodeHeader.WriteString(fmt.Sprintf("\n\t\t\t{typeof(%s), %d},",
			msg.Name, ntcCode))
		w.codeToTypeHeader.WriteString(fmt.Sprintf("\n\t\t\t{%d, %s.Parser.ParseFrom},",
			kiwi.MergeSvcCode(msg.Svc.Id, msg.Code), msg.Name))
	}
	return nil
}

func (w *gCsWriter) Save() error {
	data := w.header.String() +
		w.typeToCodeHeader.String() +
		w.typeToCodeFooter.String() +
		w.codeToTypeHeader.String() +
		w.codeToTypeFooter.String() +
		w.reqResHeader.String() +
		w.reqResFooter.String() +
		w.footer.String()
	return w.saveCustom("proto/cs/Code.cs", data)
}

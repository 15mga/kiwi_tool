package kiwi

import (
	"fmt"
	"github.com/15mga/kiwi"
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
		if w.Builder().playerRole == role {
			return true
		}
	}
	return false
}

func (w *gCsWriter) WriteMsg(idx int, msg *Msg) {
	switch msg.Type {
	case EMsgReq:
		reqName := msg.Name
		resName := reqToRes(reqName)
		if !w.isPlayerMsg(msg) {
			resMsg, ok := _NameToMsg[resName]
			if !ok {
				return
			}
			if !w.isPlayerMsg(resMsg) {
				return
			}
		}
		reqCode := kiwi.MergeSvcCode(msg.Svc.Id, msg.Code)
		w.typeToCodeHeader.WriteString(fmt.Sprintf("\n\t\t\t{typeof(%s), %d},",
			msg.Name, reqCode))
		_, ok := w.svc.Res[resName]
		if ok {
			res := w.svc.Res[resName]
			resCode := kiwi.MergeSvcCode(res.Svc.Id, res.Code)
			w.reqResHeader.WriteString(fmt.Sprintf("\n\t\t\t{%d, %d},",
				reqCode, resCode))
		}
	case EMsgRes:
		resName := msg.Name
		if !w.isPlayerMsg(msg) {
			reqName := resToReq(resName)
			reqMsg, ok := _NameToMsg[reqName]
			if !ok {
				return
			}
			if !w.isPlayerMsg(reqMsg) {
				return
			}
		}
		w.codeToTypeHeader.WriteString(fmt.Sprintf("\n\t\t\t{%d, %s.Parser.ParseFrom},",
			kiwi.MergeSvcCode(msg.Svc.Id, msg.Code), msg.Name))
	case EMsgNtc:
		if !w.isPlayerMsg(msg) {
			return
		}
		ntcCode := kiwi.MergeSvcCode(msg.Svc.Id, msg.Code)
		w.typeToCodeHeader.WriteString(fmt.Sprintf("\n\t\t\t{typeof(%s), %d},",
			msg.Name, ntcCode))
		w.codeToTypeHeader.WriteString(fmt.Sprintf("\n\t\t\t{%d, %s.Parser.ParseFrom},",
			kiwi.MergeSvcCode(msg.Svc.Id, msg.Code), msg.Name))
	}
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

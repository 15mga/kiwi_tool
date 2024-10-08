package kiwi

import (
	"github.com/15mga/kiwi"
	tool "github.com/15mga/kiwi_tool"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"strings"
)

type (
	EMsg uint8
)

const (
	EMsgNil EMsg = iota
	EMsgPus
	EMsgReq
	EMsgRes
	EMsgNtc
	EMsgSch
)

type Msg struct {
	Type       EMsg
	MsgName    string
	MethodName string
	MethodCode kiwi.TMethod
	Svc        *svc
	Msg        *protogen.Message
	Worker     *tool.Worker
}

func (m *Msg) Copy(msg *Msg) {
	msg.Type = m.Type
	msg.MsgName = m.MsgName
	msg.MethodName = m.MethodName
	msg.MethodCode = m.MethodCode
	msg.Msg = m.Msg
	msg.Worker = m.Worker
}

func (m *Msg) GetWorker() *tool.Worker {
	if m.Worker != nil {
		return m.Worker
	}
	if m.Svc.Worker != nil {
		return m.Svc.Worker
	}
	return &tool.Worker{
		Mode: tool.EWorker_Self,
	}
}

func getEMsg(msg *protogen.Message) EMsg {
	msgFullName := msg.GoIdent.GoName
	switch {
	case strings.HasSuffix(msgFullName, "Pus"):
		return EMsgPus
	case strings.HasSuffix(msgFullName, "Req"):
		return EMsgReq
	case strings.HasSuffix(msgFullName, "Res"):
		return EMsgRes
	case strings.HasSuffix(msgFullName, "Ntc"):
		return EMsgNtc
	case isSchema(msg):
		return EMsgSch
	default:
		return EMsgNil
	}
}

func NewMsg(typ EMsg, msg *protogen.Message) *Msg {
	msgFullName := msg.GoIdent.GoName
	return &Msg{
		Type:       typ,
		MsgName:    msgFullName,
		MethodName: msgFullName[:len(msgFullName)-3],
		MethodCode: kiwi.TMethod(proto.GetExtension(msg.Desc.Options(), tool.E_Method).(int32)),
		Msg:        msg,
		Worker:     proto.GetExtension(msg.Desc.Options(), tool.E_Worker).(*tool.Worker),
	}
}

func reqToRes(req string) string {
	if strings.HasSuffix(req, "Req") {
		return req[:len(req)-3] + "Res"
	}
	return req
}

func resToReq(res string) string {
	if strings.HasSuffix(res, "Res") {
		return res[:len(res)-3] + "Req"
	}
	return res
}

package kiwi

import (
	"github.com/15mga/kiwi"
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
	Type   EMsg
	Name   string
	Method string
	Code   kiwi.TCode
	Svc    *Svc
	Msg    *protogen.Message
	Worker *tool.Worker
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

func NewMsg(svc *Svc, typ EMsg, msg *protogen.Message) *Msg {
	msgFullName := msg.GoIdent.GoName
	return &Msg{
		Type:   typ,
		Name:   msgFullName,
		Method: msgFullName[:len(msgFullName)-3],
		Svc:    svc,
		Code:   kiwi.TCode(proto.GetExtension(msg.Desc.Options(), tool.E_Code).(int32)),
		Msg:    msg,
		Worker: proto.GetExtension(msg.Desc.Options(), tool.E_Worker).(*tool.Worker),
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

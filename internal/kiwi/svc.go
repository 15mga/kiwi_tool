package kiwi

import (
	"errors"
	"fmt"
	"github.com/15mga/kiwi"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
)

var (
	_NameToSvc = make(map[string]*Svc)
	_NameToMsg = make(map[string]*Msg)
	_SvcSlc    []*Svc
)

type Svc struct {
	Id        kiwi.TSvc
	Name      string
	CodeToMsg map[kiwi.TCode]*Msg
	MsgSlc    []*Msg
	Files     []*protogen.File
	Pus       map[string]*Msg
	Req       map[string]*Msg
	Res       map[string]*Msg
	Ntc       map[string]*Msg
	Sch       map[string]*Msg
	Msg       map[string]*Msg
	WatchNtc  []*tool.Ntc
	Worker    *tool.Worker
}

func (s *Svc) AddFile(file *protogen.File) error {
	s.Files = append(s.Files, file)

	for _, m := range file.Messages {
		t := getEMsg(m)
		msg := NewMsg(s, t, m)
		if msg.Type != EMsgNil && msg.Type != EMsgSch {
			msg1, ok := s.CodeToMsg[msg.Code]
			if ok {
				return errors.New(fmt.Sprintf("%s svc, %s and %s had same code %d",
					s.Name, msg1.Name, msg.Name, msg.Code))
			}
			s.CodeToMsg[msg.Code] = msg
		}
		s.MsgSlc = append(s.MsgSlc, msg)
		_NameToMsg[msg.Name] = msg

		switch msg.Type {
		case EMsgNil:
			s.Msg[msg.Name] = msg
		case EMsgPus:
			s.Pus[msg.Name] = msg
		case EMsgReq:
			s.Req[msg.Name] = msg
		case EMsgRes:
			s.Res[msg.Name] = msg
		case EMsgNtc:
			s.Res[msg.Name] = msg
		case EMsgSch:
			s.Sch[msg.Name] = msg
		}
	}
	return nil
}

func addSvc(file *protogen.File) error {
	extSvc := proto.GetExtension(file.Desc.Options(), tool.E_Svc).(*tool.Svc)
	if extSvc == nil {
		return nil
	}
	svcName := extSvc.Name
	if svcName == "" {
		return nil
	}
	svcId := kiwi.TSvc(extSvc.Id)
	svc, ok := _NameToSvc[svcName]
	if ok {
		if svcId > 0 {
			if svc.Id > 0 {
				if svc.Id != svcId {
					return errors.New(fmt.Sprintf("svc %s had id %d and %d", svcName, svc.Id, svcId))
				}
			} else {
				svc.Id = svcId
			}
		}
		if extSvc.Worker != nil {
			if svc.Worker != nil {
				if svc.Worker.Mode != extSvc.Worker.Mode {
					return errors.New(fmt.Sprintf("svc %s had worker %d and %d", svcName, svc.Worker.Mode, extSvc.Worker.Mode))
				}
				if svc.Worker.Key != extSvc.Worker.Key {
					return errors.New(fmt.Sprintf("svc %s had worker key %s and %s", svcName, svc.Worker.Key, extSvc.Worker.Key))
				}
				if svc.Worker.Origin != extSvc.Worker.Origin {
					return errors.New(fmt.Sprintf("svc %s had worker origin %d and %d", svcName, svc.Worker.Origin, extSvc.Worker.Origin))
				}
			} else {
				svc.Worker = extSvc.Worker
			}
		}
	} else {
		svc = &Svc{
			Name:      svcName,
			Id:        svcId,
			CodeToMsg: make(map[kiwi.TCode]*Msg),
			Pus:       make(map[string]*Msg),
			Req:       make(map[string]*Msg),
			Res:       make(map[string]*Msg),
			Ntc:       make(map[string]*Msg),
			Sch:       make(map[string]*Msg),
			Msg:       make(map[string]*Msg),
			Worker:    extSvc.Worker,
		}
		_SvcSlc = append(_SvcSlc, svc)
		_NameToSvc[svcName] = svc
	}
	svc.WatchNtc = append(svc.WatchNtc, extSvc.Ntc...)
	return svc.AddFile(file)
}

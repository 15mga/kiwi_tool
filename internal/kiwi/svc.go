package kiwi

import (
	"errors"
	"fmt"
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/util"
	tool "github.com/15mga/kiwi_tool"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
)

func newSvc(name string) *svc {
	return &svc{
		Name:      name,
		CodeToMsg: make(map[kiwi.TMethod]*Msg),
		Pus:       make(map[string]*Msg),
		Req:       make(map[string]*Msg),
		Res:       make(map[string]*Msg),
		Ntc:       make(map[string]*Msg),
		Sch:       make(map[string]*Msg),
		Msg:       make(map[string]*Msg),
	}
}

type svc struct {
	Id           kiwi.TSvc
	Name         string
	Worker       *tool.Worker
	CodeToMsg    map[kiwi.TMethod]*Msg
	MsgSlc       []*Msg
	Pus          map[string]*Msg
	Req          map[string]*Msg
	Res          map[string]*Msg
	Ntc          map[string]*Msg
	Sch          map[string]*Msg
	Msg          map[string]*Msg
	WatchNtc     []*tool.Ntc
	Fail         []*tool.Fail
	Dependencies []string
	Common       []string //通用服务的借口，不是具体服务
}

func (s *svc) hasRecycleDependencies(svcMap map[string]*svc, name string, p ...string) (bool, string) {
	for _, dep := range s.Dependencies {
		if dep == name {
			return true, util.StringsJoin("->", append(p, s.Name, name)...)
		}
		ds, ok := svcMap[dep]
		if !ok {
			continue
		}
		exist, path := ds.hasRecycleDependencies(svcMap, name, append(p, s.Name)...)
		if exist {
			return true, path
		}
	}
	return false, ""
}

func (s *svc) IsCommonSvc() bool {
	return len(s.Common) > 0
}

func (s *svc) AddFile(file *protogen.File) error {
	extSvc := proto.GetExtension(file.Desc.Options(), tool.E_Svc).(*tool.Svc)
	svcId := kiwi.TSvc(extSvc.Id)
	if svcId != 0 {
		if s.Id != 0 {
			if s.Id != svcId {
				return errors.New(fmt.Sprintf("svc %s had id %d and %d", s.Name, s.Id, svcId))
			}
		} else {
			s.Id = svcId
		}
	}
	if extSvc.Worker != nil {
		if s.Worker != nil {
			if s.Worker.Mode != extSvc.Worker.Mode {
				return errors.New(fmt.Sprintf("svc %s had worker %d and %d", s.Name, s.Worker.Mode, extSvc.Worker.Mode))
			}
			if s.Worker.Key != extSvc.Worker.Key {
				return errors.New(fmt.Sprintf("svc %s had worker key %s and %s", s.Name, s.Worker.Key, extSvc.Worker.Key))
			}
			if s.Worker.Origin != extSvc.Worker.Origin {
				return errors.New(fmt.Sprintf("svc %s had worker origin %d and %d", s.Name, s.Worker.Origin, extSvc.Worker.Origin))
			}
		} else {
			s.Worker = extSvc.Worker
		}
	}
	if extSvc.Common != nil {
		s.Common = append(s.Common, extSvc.Common...)
	}
	s.WatchNtc = append(s.WatchNtc, extSvc.Ntc...)
	s.Fail = append(s.Fail, extSvc.Fail...)
	s.Dependencies = append(s.Dependencies, extSvc.Dependencies...)

	for _, m := range file.Messages {
		t := getEMsg(m)
		msg := NewMsg(t, m)
		err := s.AddMsg(msg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *svc) AddMsg(msg *Msg) error {
	if msg.Type != EMsgNil && msg.Type != EMsgSch {
		msg1, ok := s.CodeToMsg[msg.MethodCode]
		if ok {
			return errors.New(fmt.Sprintf("%s and %s had same code %d",
				msg1.MsgName, msg.MsgName, msg.MethodCode))
		}
		s.CodeToMsg[msg.MethodCode] = msg
	}
	s.MsgSlc = append(s.MsgSlc, msg)
	msg.Svc = s

	switch msg.Type {
	case EMsgNil:
		s.Msg[msg.MsgName] = msg
	case EMsgPus:
		s.Pus[msg.MsgName] = msg
	case EMsgReq:
		s.Req[msg.MsgName] = msg
	case EMsgRes:
		s.Res[msg.MsgName] = msg
	case EMsgNtc:
		s.Res[msg.MsgName] = msg
	case EMsgSch:
		s.Sch[msg.MsgName] = msg
	}
	return nil
}

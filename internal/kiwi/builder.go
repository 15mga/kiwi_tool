package kiwi

import (
	tool "github.com/15mga/kiwi_tool"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"sort"
)

func newBuilder(plugin *protogen.Plugin, module, db string, playerRoles map[string]struct{}, clientMap map[string]struct{}) *builder {
	b := &builder{
		plugin:      plugin,
		module:      module,
		playerRoles: playerRoles,
		svcMap:      make(map[string]*svc),
	}
	b.addGlobalWriters(
		newGCodeWriter(),
		newGFailCodeWriter(),
		NewGMockWriter(),
		NewGNtcWriter(),
		newGReqResWriter(),
		NewGRoleWriter(),
		NewGSvcWriter(),
	)
	b.addWriters(
		NewMockWriter(),
		NewCodecWriter(),
		NewReqWriter(),
		NewSvcWriter(),
		NewReqPrcWriter(),
		NewModelWriter(),
	)
	_, ok := clientMap["cs"]
	if ok {
		b.addGlobalWriters(NewGCsWriter())
	}
	switch db {
	case "mgo":
		b.addWriters(NewMgoWriter())
	}
	return b
}

type builder struct {
	plugin        *protogen.Plugin
	module        string
	globalWriters []IWriter
	writers       []IWriter
	playerRoles   map[string]struct{}
	svcMap        map[string]*svc
	svcSlc        []*svc
	commonSvcSlc  []*svc
}

func (b *builder) isPlayerRole(role string) bool {
	_, ok := b.playerRoles[role]
	return ok
}

func (b *builder) addGlobalWriters(writers ...IWriter) {
	for _, w := range writers {
		w.setBuilder(b)
	}
	b.globalWriters = append(b.globalWriters, writers...)
}

func (b *builder) addWriters(writers ...IWriter) {
	for _, w := range writers {
		w.setBuilder(b)
	}
	b.writers = append(b.writers, writers...)
}

func (b *builder) build() error {
	for _, file := range b.plugin.Files {
		err := b.prcFile(file)
		if err != nil {
			return err
		}
	}
	sort.Slice(b.svcSlc, func(i, j int) bool {
		return b.svcSlc[i].Id < b.svcSlc[j].Id
	})
	svcSlc := make([]*svc, 0, len(b.svcSlc))
	commonSvcSlc := make([]*svc, 0, len(b.svcSlc))
	for _, svc := range b.svcSlc {
		if svc.Common == nil {
			svcSlc = append(svcSlc, svc)
		} else {
			commonSvcSlc = append(commonSvcSlc, svc)
		}
	}
	for _, svc := range commonSvcSlc {
		for _, svcName := range svc.Common {
			if s, ok := b.svcMap[svcName]; ok {
				for _, msg := range svc.MsgSlc {
					var m Msg
					msg.Copy(&m)
					err := s.AddMsg(&m)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return b.write(svcSlc)
}

func (b *builder) prcFile(file *protogen.File) error {
	extSvc := proto.GetExtension(file.Desc.Options(), tool.E_Svc).(*tool.Svc)
	if extSvc == nil {
		return nil
	}
	svcName := extSvc.Name
	if svcName == "" {
		return nil
	}
	svc, ok := b.svcMap[svcName]
	if !ok {
		svc = newSvc(svcName)
		b.svcMap[svcName] = svc
		b.svcSlc = append(b.svcSlc, svc)
	}
	return svc.AddFile(file)
}

func (b *builder) write(svcSlc []*svc) error {
	for _, writer := range b.globalWriters {
		writer.Reset()
		writer.WriteHeader()
	}
	for _, svc := range svcSlc {
		msgSlc := svc.MsgSlc
		if len(msgSlc) == 0 {
			continue
		}
		sort.Slice(msgSlc, func(i, j int) bool {
			return msgSlc[i].MethodCode < msgSlc[j].MethodCode
		})
		for _, writer := range b.globalWriters {
			writer.SetSvc(svc)
			for i, m := range msgSlc {
				err := writer.WriteMsg(i, m)
				if err != nil {
					return err
				}
			}
		}
		for _, writer := range b.writers {
			writer.Reset()
			writer.SetSvc(svc)
			writer.WriteHeader()
			for i, m := range msgSlc {
				err := writer.WriteMsg(i, m)
				if err != nil {
					return err
				}
			}

			if !writer.Dirty() {
				continue
			}
			writer.SetDirty(false)

			writer.WriteFooter()
			err := writer.Save()
			if err != nil {
				return err
			}
		}
	}

	for _, writer := range b.globalWriters {
		if !writer.Dirty() {
			continue
		}
		writer.SetDirty(false)
		writer.WriteFooter()
		err := writer.Save()
		if err != nil {
			return err
		}
	}
	return nil
}

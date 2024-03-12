package kiwi

import (
	"google.golang.org/protobuf/compiler/protogen"
	"sort"
)

func newBuilder(plugin *protogen.Plugin, module, db string, playerRole string) *builder {
	b := &builder{
		plugin:     plugin,
		module:     module,
		playerRole: playerRole,
	}
	b.addGlobalWriters(
		NewGCsWriter(),
		NewGNtcWriter(),
		NewGRoleWriter(),
		NewGSvcWriter(),
	)
	b.addWriters(
		NewCodecWriter(),
		NewFailCodeWriter(),
		//NewPusWriter(),
		NewReqWriter(),
		NewReqResWriter(),
		NewSvcWriter(),
		NewPusReqPrcWriter(),
	)
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
	playerRole    string
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
	sort.Slice(_SvcSlc, func(i, j int) bool {
		return _SvcSlc[i].Id < _SvcSlc[j].Id
	})
	for _, writer := range b.globalWriters {
		writer.Reset()
		writer.WriteHeader()
	}
	for _, svc := range _SvcSlc {
		msgSlc := svc.MsgSlc
		if len(msgSlc) == 0 {
			continue
		}
		sort.Slice(msgSlc, func(i, j int) bool {
			return msgSlc[i].Code < msgSlc[j].Code
		})
		for _, writer := range b.globalWriters {
			writer.SetSvc(svc)
			for i, m := range msgSlc {
				writer.WriteMsg(i, m)
			}
		}
		for _, writer := range b.writers {
			writer.Reset()
			writer.SetSvc(svc)
			writer.WriteHeader()
			for i, m := range msgSlc {
				writer.WriteMsg(i, m)
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

package kiwi

import (
	"fmt"
	"github.com/15mga/kiwi/util"
	tool "github.com/15mga/kiwi_tool"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"strings"
)

func NewModelWriter() IWriter {
	return &modelWriter{}
}

type modelWriter struct {
	baseWriter
	headBuilder   *strings.Builder
	facBuilder    *strings.Builder
	structBuilder *strings.Builder
	msgSlc        []*Msg
}

func (w *modelWriter) Reset() {
	w.headBuilder = &strings.Builder{}
	w.facBuilder = &strings.Builder{}
	w.structBuilder = &strings.Builder{}
}

func (w *modelWriter) WriteHeader() {
	w.headBuilder.WriteString("package " + w.Svc().Name)
	w.headBuilder.WriteString("\n\nimport (")
	w.headBuilder.WriteString("\n\t\"github.com/15mga/kiwi/util/mgo\"")
	w.headBuilder.WriteString(fmt.Sprintf("\n\t\"%s/proto/pb\"", w.Module()))

	w.facBuilder.WriteString("\n\nvar (")
	w.facBuilder.WriteString("\n\t_SchemaFac = map[string]func() mgo.IModel{")

}

func (w *modelWriter) WriteFooter() {
	w.facBuilder.WriteString("\n\t}")
	w.facBuilder.WriteString("\n)")
	w.headBuilder.WriteString("\n)")
}

func (w *modelWriter) getFieldName(msg *Msg, field string) string {
	if util.ToBigHump(msg.Svc.Name) == msg.MsgName {
		return field
	}
	return msg.MsgName + field
}

func pbFieldTypeToGo(field *protogen.Field) string {
	switch field.Desc.Kind() {
	case protoreflect.BytesKind:
		return "[]byte"
	case protoreflect.EnumKind:
		return "pb." + field.Enum.GoIdent.GoName
	default:
		return field.Desc.Kind().String()
	}
}

func (w *modelWriter) writeFiledCost(writer *strings.Builder, field *protogen.Field, obj string) (importUnsafe bool) {
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		writer.WriteString(fmt.Sprintf("\n\tcost += 1 //%s %s", field.GoName, field.Desc.Kind()))
	case protoreflect.Int32Kind,
		protoreflect.Sint32Kind,
		protoreflect.Sfixed32Kind,
		protoreflect.Fixed32Kind,
		protoreflect.FloatKind,
		protoreflect.EnumKind:
		writer.WriteString(fmt.Sprintf("\n\tcost += 4 //%s %s", field.GoName, field.Desc.Kind()))
	case protoreflect.Int64Kind,
		protoreflect.Sint64Kind,
		protoreflect.Sfixed64Kind,
		protoreflect.Fixed64Kind,
		protoreflect.DoubleKind:
		writer.WriteString(fmt.Sprintf("\n\tcost += 8 //%s %s", field.GoName, field.Desc.Kind()))
	case protoreflect.BytesKind,
		protoreflect.StringKind:
		writer.WriteString(fmt.Sprintf("\n\tcost += int64(len(%s.%s))", obj, field.GoName))
	case protoreflect.MessageKind:
		for _, f := range field.Message.Fields {
			w.writeFiledCost(writer, f, obj+"."+field.GoName)
		}
	default:
		importUnsafe = true
		writer.WriteString(fmt.Sprintf("\n\tcost += int64(unsafe.Sizeof(%s.%s))", obj, field.GoName))
	}
	return
}

func (w *modelWriter) writeFieldItemsCost(writer *strings.Builder, field *protogen.Field, obj string) (importUnsafe bool) {
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		writer.WriteString(fmt.Sprintf("\n\tcost += int64(len(%s.%s))", obj, field.GoName))
	case protoreflect.Int32Kind,
		protoreflect.Sint32Kind,
		protoreflect.Sfixed32Kind,
		protoreflect.Fixed32Kind,
		protoreflect.FloatKind,
		protoreflect.EnumKind:
		writer.WriteString(fmt.Sprintf("\n\tcost += 4 * int64(len(%s.%s))", obj, field.GoName))
	case protoreflect.Int64Kind,
		protoreflect.Sint64Kind,
		protoreflect.Sfixed64Kind,
		protoreflect.Fixed64Kind,
		protoreflect.DoubleKind:
		writer.WriteString(fmt.Sprintf("\n\tcost += 8 * int64(len(%s.%s))", obj, field.GoName))
	case protoreflect.BytesKind,
		protoreflect.StringKind:
		writer.WriteString(fmt.Sprintf("\n\tfor _, item := range %s.%s {", obj, field.GoName))
		writer.WriteString("\n\t\tcost += int64(len(item))")
		writer.WriteString("\n}")
	case protoreflect.MessageKind:
		writer.WriteString(fmt.Sprintf("\n\tfor _, item := range %s.%s {", obj, field.GoName))
		for _, f := range field.Message.Fields {
			w.writeFiledCost(writer, f, "item")
		}
		writer.WriteString("\n}")
	default:
		importUnsafe = true
		writer.WriteString(fmt.Sprintf("\n\tcost += int64(unsafe.Sizeof(%s.%s))", obj, field.GoName))
	}
	return
}

func (w *modelWriter) WriteMsg(idx int, msg *Msg) error {
	if msg.Type != EMsgSch {
		return nil
	}

	w.facBuilder.WriteString(fmt.Sprintf("\n\t\tSchema%s: New%s,", msg.MsgName, msg.MsgName))
	storeBuilder := strings.Builder{}
	storeBuilder.WriteString(fmt.Sprintf("\n\nfunc store%s(m *%s) {", msg.MsgName, msg.MsgName))
	storeBuilder.WriteString("\n\tmgo.Set(m)")

	getBuilder := strings.Builder{}

	importBson := false
	ok := false
	bigSvcName := util.ToBigHump(w.svc.Name)
	for _, field := range msg.Msg.Fields {
		cache := proto.GetExtension(field.Desc.Options(), tool.E_Cache).(bool)
		if !cache {
			continue
		}
		if !ok {
			ok = true
			w.SetDirty(true)
		}

		//key := ""
		//switch field.Desc.Kind() {
		//case protoreflect.StringKind:
		//	key = fmt.Sprintf("m.Schema() + \":%s:\" + m.%s", field.GoName, field.GoName)
		//case protoreflect.Int32Kind,
		//	protoreflect.Sint32Kind,
		//	protoreflect.Sfixed32Kind,
		//	protoreflect.Int64Kind,
		//	protoreflect.Sint64Kind,
		//	protoreflect.Sfixed64Kind:
		//	key = "fmt.Sprintf(\"%s:%s:%d\", m.Schema(), " + fmt.Sprintf("%s, m.%s)", field.GoName, field.GoName)
		//default:
		//	continue
		//}

		getBuilder.WriteString(fmt.Sprintf("\n\n\tfunc Get%sWith%s(%s string) *%s {", msg.MsgName, field.GoName, field.Desc.Name(), msg.MsgName))
		getBuilder.WriteString(fmt.Sprintf("\n\tm, ok := mgo.Get[*%s](%s)", msg.MsgName, field.Desc.Name()))
		getBuilder.WriteString("\n\tif ok {")
		getBuilder.WriteString("\n\t\treturn m")
		getBuilder.WriteString("\n\t}")
		getBuilder.WriteString(fmt.Sprintf("\n\tm = _SchemaFac[Schema%s]().(*%s)", msg.MsgName, msg.MsgName))
		if field.GoName == "Id" {
			getBuilder.WriteString("\n\tm.Load(id)")
		} else {
			importBson = true
			if msg.MsgName == bigSvcName {
				getBuilder.WriteString(fmt.Sprintf("\n\tm.LoadWithFilter(bson.M{%s:%s})", field.GoName, field.Desc.Name()))
			} else {
				getBuilder.WriteString(fmt.Sprintf("\n\tm.LoadWithFilter(bson.M{%s%s:%s})", msg.MsgName, field.GoName, field.Desc.Name()))
			}
		}
		getBuilder.WriteString(fmt.Sprintf("\n\tstore%s(m)", msg.MsgName))
		getBuilder.WriteString("\n\treturn m")
		getBuilder.WriteString("\n}")
	}

	storeBuilder.WriteString("\n}")

	w.structBuilder.WriteString(storeBuilder.String())
	w.structBuilder.WriteString(getBuilder.String())

	w.structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc Load%s(filter any) *%s {", msg.MsgName, msg.MsgName))
	w.structBuilder.WriteString(fmt.Sprintf("\n\tm := _SchemaFac[Schema%s]().(*%s)", msg.MsgName, msg.MsgName))
	w.structBuilder.WriteString("\n\tm.LoadWithFilter(filter)")
	w.structBuilder.WriteString(fmt.Sprintf("\n\tstore%s(m)", msg.MsgName))
	w.structBuilder.WriteString("\n\treturn m")
	w.structBuilder.WriteString("\n}")

	w.structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc New%s() mgo.IModel {", msg.MsgName))
	w.structBuilder.WriteString(fmt.Sprintf("\n\tm := &%s{", msg.MsgName))
	w.structBuilder.WriteString(fmt.Sprintf("\n\t%s:&pb.%s{},", msg.MsgName, msg.MsgName))
	w.structBuilder.WriteString("\n\t}")
	w.structBuilder.WriteString(fmt.Sprintf("\n\tm.Model = mgo.NewModel(Schema%s, %d, m.GetVal)", msg.MsgName, len(msg.Msg.Fields)))
	w.structBuilder.WriteString("\n\treturn m")
	w.structBuilder.WriteString("\n}")

	w.structBuilder.WriteString(fmt.Sprintf("\n\ntype %s struct {", msg.MsgName))
	w.structBuilder.WriteString(fmt.Sprintf("\n\t*pb.%s", msg.MsgName))
	w.structBuilder.WriteString("\n\t*mgo.Model")
	w.structBuilder.WriteString("\n}")

	costBuilder := &strings.Builder{}
	costBuilder.WriteString(fmt.Sprintf("\n\nfunc (this *%s) Cost() int64 {", msg.MsgName))
	costBuilder.WriteString("\n\tvar cost int64 = 0")

	importUnsafe := false
	for _, field := range msg.Msg.Fields {
		if field.Desc.IsList() {
			if w.writeFieldItemsCost(costBuilder, field, "this") {
				importUnsafe = true
			}
		} else {
			if w.writeFiledCost(costBuilder, field, "this") {
				importUnsafe = true
			}
		}
	}
	if importUnsafe {
		w.headBuilder.WriteString("\n\t\"unsafe\"")
	}
	if importBson {
		w.headBuilder.WriteString("\n\t\"go.mongodb.org/mongo-driver/bson\"")
	}

	getterBuilder := &strings.Builder{}
	getterBuilder.WriteString(fmt.Sprintf("\n\nfunc (this *%s) GetVal(key string) any {", msg.MsgName))
	getterBuilder.WriteString("\n\tswitch key {")

	mapBuilder := &strings.Builder{}

	for _, field := range msg.Msg.Fields {
		if field.GoName == "Id" {
			continue
		}
		getterBuilder.WriteString(fmt.Sprintf("\n\tcase %s:", w.getFieldName(msg, field.GoName)))
		getterBuilder.WriteString(fmt.Sprintf("\n\t\treturn this.%s", field.GoName))
		if field.Desc.IsList() {
			switch field.Desc.Kind() {
			case protoreflect.BytesKind:
				w.structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) Set%s(val [][]byte) {", msg.MsgName, field.GoName))
				w.structBuilder.WriteString(fmt.Sprintf("\n\tthis.%s = val", field.GoName))
				w.structBuilder.WriteString(fmt.Sprintf("\n\tthis.SetDirty(%s)", w.getFieldName(msg, field.GoName)))
				w.structBuilder.WriteString("\n}")
			case protoreflect.EnumKind:
				w.structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) Set%s(val []pb.%s) {", msg.MsgName, field.GoName, field.Enum.GoIdent.GoName))
				w.structBuilder.WriteString(fmt.Sprintf("\n\tthis.%s = val", field.GoName))
				w.structBuilder.WriteString(fmt.Sprintf("\n\tthis.SetDirty(%s)", w.getFieldName(msg, field.GoName)))
				w.structBuilder.WriteString("\n}")
			default:
				ts := field.Desc.Kind().String()
				if field.Desc.Kind() == protoreflect.MessageKind {
					ts = fmt.Sprintf("*pb.%s", field.Message.Desc.Name())
				}
				w.structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) Set%s(val []%s) {", msg.MsgName, field.GoName, ts))
				w.structBuilder.WriteString(fmt.Sprintf("\n\tthis.%s = val", field.GoName))
				w.structBuilder.WriteString(fmt.Sprintf("\n\tthis.SetDirty(%s)", w.getFieldName(msg, field.GoName)))
				w.structBuilder.WriteString("\n}")

				//push
				w.structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) Push%s(items ...%s) {", msg.MsgName, field.GoName, ts))
				w.structBuilder.WriteString(fmt.Sprintf("\n\tthis.%s = append(this.%s, items...)", field.GoName, field.GoName))
				w.structBuilder.WriteString(fmt.Sprintf("\n\tthis.SetDirty(%s)", w.getFieldName(msg, field.GoName)))
				w.structBuilder.WriteString("\n}")
				//addToSet
				w.structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) AddToSet%s(items ...%s) {", msg.MsgName, field.GoName, ts))
				w.structBuilder.WriteString("\n\tfor _, item := range items {")
				w.structBuilder.WriteString(fmt.Sprintf("\n\t\tfor _, v := range this.%s {", field.GoName))
				w.structBuilder.WriteString("\n\t\t\tif v == item {")
				w.structBuilder.WriteString("\n\t\t\t\treturn")
				w.structBuilder.WriteString("\n\t\t\t}")
				w.structBuilder.WriteString("\n\t\t}")
				w.structBuilder.WriteString(fmt.Sprintf("\n\t\tthis.%s = append(this.%s, item)", field.GoName, field.GoName))
				w.structBuilder.WriteString("\n\t}")
				w.structBuilder.WriteString(fmt.Sprintf("\n\tthis.SetDirty(%s)", w.getFieldName(msg, field.GoName)))
				w.structBuilder.WriteString("\n}")
				//pull
				w.structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) Pull%s(items ...%s) {", msg.MsgName, field.GoName, ts))
				w.structBuilder.WriteString(fmt.Sprintf("\n\tif this.%s == nil || len(this.%s) == 0 {", field.GoName, field.GoName))
				w.structBuilder.WriteString("\n\t\treturn")
				w.structBuilder.WriteString("\n\t}")
				w.structBuilder.WriteString("\n\tdirty := false")
				w.structBuilder.WriteString("\n\tfor _, item := range items {")
				w.structBuilder.WriteString(fmt.Sprintf("\n\tfor i, v := range this.%s {", field.GoName))
				w.structBuilder.WriteString("\n\t\tif v == item {")
				w.structBuilder.WriteString(fmt.Sprintf("\n\t\t\tthis.%s = append(this.%s[:i], this.%s[i+1:]...)", field.GoName, field.GoName, field.GoName))
				w.structBuilder.WriteString("\n\t\t\tdirty = true")
				w.structBuilder.WriteString("\n\t\t\tbreak")
				w.structBuilder.WriteString("\n\t\t}")
				w.structBuilder.WriteString("\n\t\t}")
				w.structBuilder.WriteString("\n\t}")
				w.structBuilder.WriteString("\n\tif dirty {")
				w.structBuilder.WriteString(fmt.Sprintf("\n\t\tthis.SetDirty(%s)", w.getFieldName(msg, field.GoName)))
				w.structBuilder.WriteString("\n\t}")
				w.structBuilder.WriteString("\n}")
			}
		} else {
			switch field.Desc.Kind() {
			case protoreflect.BytesKind:
				w.structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) Set%s(val []byte) {", msg.MsgName, field.GoName))
			case protoreflect.EnumKind:
				w.structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) Set%s(val pb.%s) {", msg.MsgName, field.GoName, field.Enum.GoIdent.GoName))
			case protoreflect.MessageKind:
				w.structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) Set%s(val *pb.%s) {", msg.MsgName, field.GoName, field.Message.Desc.Name()))
			default:
				w.structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) Set%s(val %s) {", msg.MsgName, field.GoName, field.Desc.Kind()))
			}
			w.structBuilder.WriteString(fmt.Sprintf("\n\tthis.%s = val", field.GoName))
			w.structBuilder.WriteString(fmt.Sprintf("\n\tthis.SetDirty(%s)", w.getFieldName(msg, field.GoName)))
			w.structBuilder.WriteString("\n}")
		}
	}
	costBuilder.WriteString("\n\treturn cost")
	costBuilder.WriteString("\n}")
	getterBuilder.WriteString("\n\t default:")
	getterBuilder.WriteString("\n\t\t return nil")
	getterBuilder.WriteString("\n\t}")
	getterBuilder.WriteString("\n}")
	w.structBuilder.WriteString(costBuilder.String())
	w.structBuilder.WriteString(getterBuilder.String())
	w.structBuilder.WriteString(mapBuilder.String())

	return nil
}

func (w *modelWriter) Save() error {
	path := fmt.Sprintf("%s/model.go", w.Svc().Name)
	return w.save(path, w.headBuilder.String()+
		w.facBuilder.String()+
		w.structBuilder.String())
}

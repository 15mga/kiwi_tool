package kiwi

import (
	"fmt"
	"github.com/15mga/kiwi/util"
	tool "github.com/15mga/kiwi_tool"
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
	structBuilder *strings.Builder
	msgSlc        []*Msg
}

func (w *modelWriter) Reset() {
	w.headBuilder = &strings.Builder{}
	w.structBuilder = &strings.Builder{}
}

func (w *modelWriter) WriteHeader() {
	w.headBuilder.WriteString("package " + w.Svc().Name)
	w.headBuilder.WriteString("\n\nimport (")
	w.headBuilder.WriteString("\n\t\"github.com/15mga/kiwi/util\"")
	w.headBuilder.WriteString("\n\t\"github.com/15mga/kiwi/util/mgo\"")
	w.headBuilder.WriteString("\n\t\"go.mongodb.org/mongo-driver/bson\"")
	w.headBuilder.WriteString("\n\t\"go.mongodb.org/mongo-driver/mongo\"")
	w.headBuilder.WriteString(fmt.Sprintf("\n\t\"%s/proto/pb\"", w.Module()))
	w.headBuilder.WriteString("\n)")
}

func (w *modelWriter) WriteFooter() {

}

func (w *modelWriter) getFieldName(msg *Msg, field string) string {
	if util.ToBigHump(msg.Svc.Name) == msg.Name {
		return field
	}
	return msg.Name + field
}

func (w *modelWriter) WriteMsg(idx int, msg *Msg) error {
	if msg.Type != EMsgSch {
		return nil
	}
	if !proto.GetExtension(msg.Msg.Desc.Options(), tool.E_Cache).(bool) {
		return nil
	}
	w.SetDirty(true)
	w.structBuilder.WriteString(fmt.Sprintf("\n\n\t func New%s() *%s {", msg.Name, msg.Name))
	w.structBuilder.WriteString(fmt.Sprintf("\n\tm := &%s{", msg.Name))
	w.structBuilder.WriteString(fmt.Sprintf("\n\t%s:&pb.%s{},", msg.Name, msg.Name))
	w.structBuilder.WriteString("\n\t}")
	w.structBuilder.WriteString(fmt.Sprintf("\n\tm.Model = util.NewModel(Schema%s, %d, m.GetVal)", msg.Name, len(msg.Msg.Fields)))
	w.structBuilder.WriteString("\n\treturn m")
	w.structBuilder.WriteString("\n}")

	w.structBuilder.WriteString(fmt.Sprintf("\n\ntype %s struct {", msg.Name))
	w.structBuilder.WriteString(fmt.Sprintf("\n\t*pb.%s", msg.Name))
	w.structBuilder.WriteString("\n\t*util.Model")
	w.structBuilder.WriteString("\n}")

	loadBuilder := &strings.Builder{}
	loadBuilder.WriteString(fmt.Sprintf("\n\nfunc (this *%s) LoadWithId(id string) error {", msg.Name))
	loadBuilder.WriteString(fmt.Sprintf("\n\treturn mgo.FindOne(Schema%s, bson.M{\"_id\":id}, &this.%s)", msg.Name, msg.Name))
	loadBuilder.WriteString("\n}")
	loadBuilder.WriteString(fmt.Sprintf("\n\nfunc (this *%s) Load(filter any) error {", msg.Name))
	loadBuilder.WriteString(fmt.Sprintf("\n\treturn mgo.FindOne(Schema%s, filter, &this.%s)", msg.Name, msg.Name))
	loadBuilder.WriteString("\n}")
	loadBuilder.WriteString(fmt.Sprintf("\n\nfunc (this *%s) UpdateDb() (*mongo.UpdateResult, error) {", msg.Name))
	loadBuilder.WriteString("\n\tupdate := bson.M{}")
	loadBuilder.WriteString("\n\tthis.GenUpdate(update)")
	loadBuilder.WriteString(fmt.Sprintf("\n\treturn mgo.UpdateOne(Schema%s, bson.M{\"_id\":this.Id}, bson.M{\"$set\": update})", msg.Name))
	loadBuilder.WriteString("\n}")

	getterBuilder := &strings.Builder{}
	getterBuilder.WriteString(fmt.Sprintf("\n\nfunc (this *%s) GetVal(key string) any {", msg.Name))
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
				w.structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) Set%s(val [][]byte) {", msg.Name, field.GoName))
				w.structBuilder.WriteString(fmt.Sprintf("\n\tthis.%s = val", field.GoName))
				w.structBuilder.WriteString(fmt.Sprintf("\n\tthis.SetDirty(%s)", w.getFieldName(msg, field.GoName)))
				w.structBuilder.WriteString("\n}")
			case protoreflect.EnumKind:
				w.structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) Set%s(val []pb.%s) {", msg.Name, field.GoName, field.Enum.GoIdent.GoName))
				w.structBuilder.WriteString(fmt.Sprintf("\n\tthis.%s = val", field.GoName))
				w.structBuilder.WriteString(fmt.Sprintf("\n\tthis.SetDirty(%s)", w.getFieldName(msg, field.GoName)))
				w.structBuilder.WriteString("\n}")
			default:
				w.structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) Set%s(val []%s) {", msg.Name, field.GoName, field.Desc.Kind()))
				w.structBuilder.WriteString(fmt.Sprintf("\n\tthis.%s = val", field.GoName))
				w.structBuilder.WriteString(fmt.Sprintf("\n\tthis.SetDirty(%s)", w.getFieldName(msg, field.GoName)))
				w.structBuilder.WriteString("\n}")

				//push
				w.structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) Push%s(items ...%s) {", msg.Name, field.GoName, field.Desc.Kind()))
				w.structBuilder.WriteString(fmt.Sprintf("\n\tthis.%s = append(this.%s, items...)", field.GoName, field.GoName))
				w.structBuilder.WriteString(fmt.Sprintf("\n\tthis.SetDirty(%s)", w.getFieldName(msg, field.GoName)))
				w.structBuilder.WriteString("\n}")
				//addToSet
				w.structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) AddToSet%s(items ...%s) {", msg.Name, field.GoName, field.Desc.Kind()))
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
				w.structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) Pull%s(items ...%s) {", msg.Name, field.GoName, field.Desc.Kind()))
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
				w.structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) %sSet(val []byte) {", msg.Name, field.GoName))
			case protoreflect.EnumKind:
				w.structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) %sSet(val pb.%s) {", msg.Name, field.GoName, field.Enum.GoIdent.GoName))
			default:
				w.structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) %sSet(val %s) {", msg.Name, field.GoName, field.Desc.Kind()))
			}
			w.structBuilder.WriteString(fmt.Sprintf("\n\tthis.%s = val", field.GoName))
			w.structBuilder.WriteString(fmt.Sprintf("\n\tthis.SetDirty(%s)", w.getFieldName(msg, field.GoName)))
			w.structBuilder.WriteString("\n}")
		}
	}
	getterBuilder.WriteString("\n\t default:")
	getterBuilder.WriteString("\n\t\t return nil")
	getterBuilder.WriteString("\n\t}")
	getterBuilder.WriteString("\n}")
	w.structBuilder.WriteString(loadBuilder.String())
	w.structBuilder.WriteString(getterBuilder.String())
	w.structBuilder.WriteString(mapBuilder.String())

	return nil
}

func (w *modelWriter) Save() error {
	path := fmt.Sprintf("%s/model.go", w.Svc().Name)
	return w.save(path, w.headBuilder.String()+
		w.structBuilder.String())
}

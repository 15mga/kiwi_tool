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
	w.headBuilder.WriteString("\n\t\"github.com/15mga/kiwi/ds\"")
	w.headBuilder.WriteString(fmt.Sprintf("\n\t\"%s/proto/pb\"", w.Module()))
	w.headBuilder.WriteString("\n)")

	w.facBuilder.WriteString("\n\nvar (")
	w.facBuilder.WriteString("\n\t_SchemaFac = map[string]func() mgo.IModel{")

}

func (w *modelWriter) WriteFooter() {
	w.facBuilder.WriteString("\n\t}")
	w.facBuilder.WriteString("\n)")
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

	w.facBuilder.WriteString(fmt.Sprintf("\n\t\tSchema%s: New%s,", msg.Name, msg.Name))

	idxBuilder := strings.Builder{}
	idxBuilder.WriteString("\n\nvar (")
	idxBuilder.WriteString(fmt.Sprintf("\n\t\t_%sSet = ds.NewKSet[string, *%s](1024, func(model *%s) string {", msg.Name, msg.Name, msg.Name))
	idxBuilder.WriteString("\n\t\treturn model.Id")
	idxBuilder.WriteString("\n\t})")

	modelSetBuilder := strings.Builder{}
	modelSetBuilder.WriteString(fmt.Sprintf("\n\n\tfunc add%s(m *%s) {", msg.Name, msg.Name))
	modelSetBuilder.WriteString(fmt.Sprintf("\n\t_%sSet.Add(m)", msg.Name))

	slc := proto.GetExtension(msg.Msg.Desc.Options(), tool.E_Idx).([]*tool.Idx)
	for _, item := range slc {
		l := len(item.Fields)
		if l == 0 {
			continue
		}
		name := item.Fields[0].Name
		if l > 1 {
			for _, field := range item.Fields[1:] {
				name += "_" + field.Name
			}
		}
		idxBuilder.WriteString(fmt.Sprintf("\n\t\t_%sSet = ds.NewKSet[string, *%s](1024, func(model *%s) string {", name, msg.Name, msg.Name))
		idxBuilder.WriteString("\n\t})")
	}

	idxBuilder.WriteString("\n)")
	modelSetBuilder.WriteString("\n}")

	w.structBuilder.WriteString(idxBuilder.String())
	w.structBuilder.WriteString(modelSetBuilder.String())

	w.structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc Get%s(id string) *%s {", msg.Name, msg.Name))
	w.structBuilder.WriteString(fmt.Sprintf("\n\tm, ok := _%sSet.Get(id)", msg.Name))
	w.structBuilder.WriteString("\n\tif ok {")
	w.structBuilder.WriteString("\n\t\treturn m")
	w.structBuilder.WriteString("\n\t}")
	w.structBuilder.WriteString(fmt.Sprintf("\n\tm = _SchemaFac[Schema%s]().(*%s)", msg.Name, msg.Name))
	w.structBuilder.WriteString("\n\tm.Load(id)")
	w.structBuilder.WriteString(fmt.Sprintf("\n\tadd%s(m)", msg.Name))
	w.structBuilder.WriteString("\n\treturn m")
	w.structBuilder.WriteString("\n}")

	w.structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc Load%s(filter any) *%s {", msg.Name, msg.Name))
	w.structBuilder.WriteString(fmt.Sprintf("\n\tm := _SchemaFac[Schema%s]().(*%s)", msg.Name, msg.Name))
	w.structBuilder.WriteString("\n\tm.LoadWithFilter(filter)")
	w.structBuilder.WriteString(fmt.Sprintf("\n\t_%sSet.Add(m)", msg.Name))
	w.structBuilder.WriteString("\n\treturn m")
	w.structBuilder.WriteString("\n}")

	w.structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc New%s() mgo.IModel {", msg.Name))
	w.structBuilder.WriteString(fmt.Sprintf("\n\tm := &%s{", msg.Name))
	w.structBuilder.WriteString(fmt.Sprintf("\n\t%s:&pb.%s{},", msg.Name, msg.Name))
	w.structBuilder.WriteString("\n\t}")
	w.structBuilder.WriteString(fmt.Sprintf("\n\tm.Model = mgo.NewModel(Schema%s, %d, m.GetVal)", msg.Name, len(msg.Msg.Fields)))
	w.structBuilder.WriteString("\n\treturn m")
	w.structBuilder.WriteString("\n}")

	w.structBuilder.WriteString(fmt.Sprintf("\n\ntype %s struct {", msg.Name))
	w.structBuilder.WriteString(fmt.Sprintf("\n\t*pb.%s", msg.Name))
	w.structBuilder.WriteString("\n\t*mgo.Model")
	w.structBuilder.WriteString("\n}")

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
				w.structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) Set%s(val []byte) {", msg.Name, field.GoName))
			case protoreflect.EnumKind:
				w.structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) Set%s(val pb.%s) {", msg.Name, field.GoName, field.Enum.GoIdent.GoName))
			default:
				w.structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) Set%s(val %s) {", msg.Name, field.GoName, field.Desc.Kind()))
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

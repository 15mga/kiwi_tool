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
	headBuilder     *strings.Builder
	initBuilder     *strings.Builder
	modelFacBuilder *strings.Builder
	evictBuilder    *strings.Builder
	msgBuilder      *strings.Builder
	msgSlc          []*Msg
}

func (w *modelWriter) Reset() {
	w.headBuilder = &strings.Builder{}
	w.initBuilder = &strings.Builder{}
	w.modelFacBuilder = &strings.Builder{}
	w.evictBuilder = &strings.Builder{}
	w.msgBuilder = &strings.Builder{}
}

func (w *modelWriter) WriteHeader() {
	w.headBuilder.WriteString("package " + w.Svc().Name)
	w.headBuilder.WriteString("\n\nimport (")
	w.headBuilder.WriteString("\n\t\"github.com/15mga/kiwi/util/mgo\"")
	w.headBuilder.WriteString(fmt.Sprintf("\n\t\"%s/proto/pb\"", w.Module()))

	w.initBuilder.WriteString("\n\nfunc initModels() {")
	w.initBuilder.WriteString("\n\tinitModelFac()")
	w.initBuilder.WriteString("\n\tinitEvict()")
	w.initBuilder.WriteString("\n}")

	w.modelFacBuilder.WriteString("\n\nvar _ModelFac map[string]func() mgo.IModel")
	w.modelFacBuilder.WriteString("\n\nfunc initModelFac() {")
	w.modelFacBuilder.WriteString("\n\t_ModelFac = map[string]func() mgo.IModel{")

	w.evictBuilder.WriteString("\n\nfunc initEvict() {")
}

func (w *modelWriter) WriteFooter() {
	w.headBuilder.WriteString("\n)")
	w.modelFacBuilder.WriteString("\n\t}")
	w.modelFacBuilder.WriteString("\n}")
	w.evictBuilder.WriteString("\n}")
}

func (w *modelWriter) getFieldName(msg *Msg, field string) string {
	if util.ToBigHump(msg.Svc.Name) == msg.MsgName {
		return field
	}
	return msg.MsgName + field
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

func (w *modelWriter) hasCache(msg *Msg) bool {
	hasId := false
	hasCache := false
	for _, field := range msg.Msg.Fields {
		if field.GoName == "Id" {
			if hasCache {
				return true
			}
			hasId = true
			continue
		}
		cache := proto.GetExtension(field.Desc.Options(), tool.E_Cache).(bool)
		if !cache {
			if hasId {
				return true
			}
			hasCache = true
			continue
		}
	}
	return false
}

func (w *modelWriter) WriteMsg(idx int, msg *Msg) error {
	if msg.Type != EMsgSch {
		return nil
	}

	if !w.hasCache(msg) {
		return nil
	}
	w.SetDirty(true)

	w.modelFacBuilder.WriteString(fmt.Sprintf("\n\t\tSchema%s: New%s,", msg.MsgName, msg.MsgName))

	mapBuilder := strings.Builder{}
	mapBuilder.WriteString("\n\nvar (")
	mapBuilder.WriteString(fmt.Sprintf("\n\t_%sIdMap = make(map[string]struct{})", msg.MsgName))

	storeAllBuilder := &strings.Builder{}
	storeAllBuilder.WriteString(fmt.Sprintf("\n\nfunc StoreAll%ss() {", msg.MsgName))
	storeAllBuilder.WriteString(fmt.Sprintf("\n\tfor id := range _%sIdMap {", msg.MsgName))
	storeAllBuilder.WriteString(fmt.Sprintf("\n\t\tm, ok := mgo.Get[*%s](Schema%s, id)", msg.MsgName, msg.MsgName))
	storeAllBuilder.WriteString("\n\t\tif !ok {")
	storeAllBuilder.WriteString("\n\t\t\tcontinue")
	storeAllBuilder.WriteString("\n\t\t}")
	storeAllBuilder.WriteString("\n\t\tm.Store()")
	storeAllBuilder.WriteString("\n\t}")
	storeAllBuilder.WriteString("\n}")

	setBuilder := strings.Builder{}
	setBuilder.WriteString(fmt.Sprintf("\n\nfunc Set%s(m *%s) {", msg.MsgName, msg.MsgName))
	setBuilder.WriteString("\n\tmgo.Set(m)")

	delBuilder := strings.Builder{}
	delBuilder.WriteString(fmt.Sprintf("\n\nfunc Del%s(id string) {", msg.MsgName))
	delBuilder.WriteString(fmt.Sprintf("\n\tm, ok := mgo.Get[*%s](Schema%s, id)", msg.MsgName, msg.MsgName))
	delBuilder.WriteString("\n\tif !ok {")
	delBuilder.WriteString("\n\t\treturn")
	delBuilder.WriteString("\n\t}")
	delBuilder.WriteString("\n\t_ = m.Store()")
	delBuilder.WriteString(fmt.Sprintf("\n\tmgo.Del(Schema%s, id)", msg.MsgName))
	delBuilder.WriteString(fmt.Sprintf("\n\tdel%sMap(m)", msg.MsgName))
	delBuilder.WriteString("\n}")

	evictBuilder := strings.Builder{}
	evictBuilder.WriteString(fmt.Sprintf("\n\nfunc on%sEvict(model mgo.IModel) {", msg.MsgName))
	evictBuilder.WriteString(fmt.Sprintf("\n\tm := model.(*%s)", msg.MsgName))
	evictBuilder.WriteString("\n\t_ = m.Store()")
	evictBuilder.WriteString(fmt.Sprintf("\n\tdel%sMap(m)", msg.MsgName))
	evictBuilder.WriteString("\n}")

	delMapBuilder := strings.Builder{}
	delMapBuilder.WriteString(fmt.Sprintf("\n\nfunc del%sMap(m *%s) {", msg.MsgName, msg.MsgName))
	delMapBuilder.WriteString(fmt.Sprintf("\n\tdelete(_%sIdMap, m.GetId())", msg.MsgName))

	w.evictBuilder.WriteString(fmt.Sprintf("\n\tmgo.BindEvict(Schema%s, on%sEvict)", msg.MsgName, msg.MsgName))

	newFnBuilder := strings.Builder{}
	newFnBuilder.WriteString(fmt.Sprintf("\n\nfunc New%s() mgo.IModel {", msg.MsgName))
	newFnBuilder.WriteString(fmt.Sprintf("\n\tm := &%s{", msg.MsgName))
	newFnBuilder.WriteString(fmt.Sprintf("\n\t%s:&pb.%s{},", msg.MsgName, msg.MsgName))
	newFnBuilder.WriteString("\n\t}")
	newFnBuilder.WriteString(fmt.Sprintf("\n\tm.Model = mgo.NewModel(Schema%s, %d, m.GetVal)", msg.MsgName, len(msg.Msg.Fields)))
	newFnBuilder.WriteString("\n\treturn m")
	newFnBuilder.WriteString("\n}")

	newFnBuilder.WriteString(fmt.Sprintf("\n\nfunc Insert%s(data *pb.%s) (*%s, error) {", msg.MsgName, msg.MsgName, msg.MsgName))
	newFnBuilder.WriteString("\n\tif data.Id == \"\" {")
	newFnBuilder.WriteString("\n\t\treturn nil, mgo.ErrNoId")
	newFnBuilder.WriteString("\n\t}")
	newFnBuilder.WriteString(fmt.Sprintf("\n\t_, e := mgo.InsertOne(Schema%s, data)", msg.MsgName))
	newFnBuilder.WriteString("\n\tif e != nil {")
	newFnBuilder.WriteString("\n\t\treturn nil, e")
	newFnBuilder.WriteString("\n\t}")
	newFnBuilder.WriteString(fmt.Sprintf("\n\tm := New%sWithData(data)", msg.MsgName))
	newFnBuilder.WriteString(fmt.Sprintf("\n\tSet%s(m)", msg.MsgName))
	newFnBuilder.WriteString("\n\treturn m, nil")
	newFnBuilder.WriteString("\n}")

	newFnBuilder.WriteString(fmt.Sprintf("\n\n\tfunc New%sWithData(data *pb.%s) *%s {", msg.MsgName, msg.MsgName, msg.MsgName))
	newFnBuilder.WriteString(fmt.Sprintf("\n\tm := &%s{", msg.MsgName))
	newFnBuilder.WriteString(fmt.Sprintf("\n\t%s: data,", msg.MsgName))
	newFnBuilder.WriteString("\n\t}")
	newFnBuilder.WriteString(fmt.Sprintf("\n\tm.Model = mgo.NewModel(Schema%s, %d, m.GetVal)", msg.MsgName, len(msg.Msg.Fields)))
	newFnBuilder.WriteString("\n\tm.SetDirty(")

	structBuilder := strings.Builder{}
	structBuilder.WriteString(fmt.Sprintf("\n\ntype %s struct {", msg.MsgName))
	structBuilder.WriteString(fmt.Sprintf("\n\t*pb.%s", msg.MsgName))
	structBuilder.WriteString("\n\t*mgo.Model")
	structBuilder.WriteString("\n}")

	structGetterBuilder := &strings.Builder{}
	structGetterBuilder.WriteString(fmt.Sprintf("\n\nfunc (this *%s) GetVal(key string) any {", msg.MsgName))
	structGetterBuilder.WriteString("\n\tswitch key {")

	structCostBuilder := &strings.Builder{}
	structCostBuilder.WriteString(fmt.Sprintf("\n\nfunc (this *%s) Cost() int64 {", msg.MsgName))
	structCostBuilder.WriteString("\n\tvar cost int64 = 0")

	getBuilder := strings.Builder{}
	importBson := false
	importUnsafe := false
	bigSvcName := util.ToBigHump(w.svc.Name)
	for _, field := range msg.Msg.Fields {
		if field.GoName == "Id" {
			getBuilder.WriteString(fmt.Sprintf("\n\n\tfunc Get%sWith%s(%s string) (*%s, error)  {", msg.MsgName, field.GoName, field.Desc.Name(), msg.MsgName))
			getBuilder.WriteString(fmt.Sprintf("\n\tm, ok := mgo.Get[*%s](Schema%s, %s)", msg.MsgName, msg.MsgName, field.Desc.Name()))
			getBuilder.WriteString("\n\tif ok {")
			getBuilder.WriteString("\n\t\treturn m, nil")
			getBuilder.WriteString("\n\t}")
			getBuilder.WriteString(fmt.Sprintf("\n\tm = _ModelFac[Schema%s]().(*%s)", msg.MsgName, msg.MsgName))
			getBuilder.WriteString(fmt.Sprintf("\n\terr := m.Load(id, m.%s)", msg.MsgName))
			getBuilder.WriteString("\n\tif err != nil {")
			getBuilder.WriteString("\n\t\treturn nil, err")
			getBuilder.WriteString("\n\t}")
			getBuilder.WriteString(fmt.Sprintf("\n\tSet%s(m)", msg.MsgName))
			getBuilder.WriteString("\n\treturn m, nil")
			getBuilder.WriteString("\n}")
			continue
		}
		if msg.MsgName == bigSvcName {
			newFnBuilder.WriteString(fmt.Sprintf("\n\t\t%s,", field.GoName))
		} else {
			newFnBuilder.WriteString(fmt.Sprintf("\n\t\t%s%s,", msg.MsgName, field.GoName))
		}
		cache := proto.GetExtension(field.Desc.Options(), tool.E_Cache).(bool)
		if cache {
			if field.GoName != "Id" {
				switch field.Desc.Kind() {
				case protoreflect.StringKind:
					mapBuilder.WriteString(fmt.Sprintf("\n\t_%s%sToId = make(map[string]string)", msg.MsgName, field.GoName))
				case protoreflect.Int64Kind,
					protoreflect.Sint64Kind,
					protoreflect.Fixed64Kind:
					mapBuilder.WriteString(fmt.Sprintf("\n\t_%s%sToId = make(map[string]int64)", msg.MsgName, field.GoName))
				}

				setBuilder.WriteString(fmt.Sprintf("\n\t_%s%sToId[m.%s] = m.Id", msg.MsgName, field.GoName, field.GoName))

				delMapBuilder.WriteString(fmt.Sprintf("\n\tdelete(_%s%sToId, m.%s)", msg.MsgName, field.GoName, field.GoName))
			}

			getBuilder.WriteString(fmt.Sprintf("\n\n\tfunc Get%sWith%s(%s string) (*%s, error) {", msg.MsgName, field.GoName, field.Desc.Name(), msg.MsgName))
			getBuilder.WriteString(fmt.Sprintf("\n\tid, ok := _%s%sToId[%s]", msg.MsgName, field.GoName, field.Desc.Name()))
			getBuilder.WriteString("\n\tif ok {")
			getBuilder.WriteString(fmt.Sprintf("\n\t\tm, ok := mgo.Get[*%s](Schema%s, id)", msg.MsgName, msg.MsgName))
			getBuilder.WriteString("\n\t\tif ok {")
			getBuilder.WriteString("\n\t\t\treturn m, nil")
			getBuilder.WriteString("\n\t\t}")
			getBuilder.WriteString("\n\t}")
			getBuilder.WriteString(fmt.Sprintf("\n\tm := _ModelFac[Schema%s]().(*%s)", msg.MsgName, msg.MsgName))
			importBson = true
			if msg.MsgName == bigSvcName {
				getBuilder.WriteString(fmt.Sprintf("\n\terr := m.LoadWithFilter(bson.M{%s:%s})", field.GoName, field.Desc.Name()))
			} else {
				getBuilder.WriteString(fmt.Sprintf("\n\terr := m.LoadWithFilter(bson.M{%s%s:%s})", msg.MsgName, field.GoName, field.Desc.Name()))
			}
			getBuilder.WriteString("\n\tif err != nil {")
			getBuilder.WriteString("\n\t\treturn nil, err")
			getBuilder.WriteString("\n\t}")
			getBuilder.WriteString(fmt.Sprintf("\n\tSet%s(m)", msg.MsgName))
			getBuilder.WriteString("\n\treturn m, nil")
			getBuilder.WriteString("\n}")
		}

		if field.Desc.IsList() {
			if w.writeFieldItemsCost(structCostBuilder, field, "this") {
				importUnsafe = true
			}
			switch field.Desc.Kind() {
			case protoreflect.BytesKind:
				structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) Set%s(val [][]byte) {", msg.MsgName, field.GoName))
				structBuilder.WriteString(fmt.Sprintf("\n\tthis.%s = val", field.GoName))
				structBuilder.WriteString(fmt.Sprintf("\n\tthis.SetDirty(%s)", w.getFieldName(msg, field.GoName)))
				structBuilder.WriteString("\n}")
			case protoreflect.EnumKind:
				structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) Set%s(val []pb.%s) {", msg.MsgName, field.GoName, field.Enum.GoIdent.GoName))
				structBuilder.WriteString(fmt.Sprintf("\n\tthis.%s = val", field.GoName))
				structBuilder.WriteString(fmt.Sprintf("\n\tthis.SetDirty(%s)", w.getFieldName(msg, field.GoName)))
				structBuilder.WriteString("\n}")
			default:
				ts := field.Desc.Kind().String()
				if field.Desc.Kind() == protoreflect.MessageKind {
					ts = fmt.Sprintf("*pb.%s", field.Message.Desc.Name())
				}
				structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) Set%s(val []%s) {", msg.MsgName, field.GoName, ts))
				structBuilder.WriteString(fmt.Sprintf("\n\tthis.%s = val", field.GoName))
				structBuilder.WriteString(fmt.Sprintf("\n\tthis.SetDirty(%s)", w.getFieldName(msg, field.GoName)))
				structBuilder.WriteString("\n}")

				//push
				structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) Push%s(items ...%s) {", msg.MsgName, field.GoName, ts))
				structBuilder.WriteString(fmt.Sprintf("\n\tthis.%s = append(this.%s, items...)", field.GoName, field.GoName))
				structBuilder.WriteString(fmt.Sprintf("\n\tthis.SetDirty(%s)", w.getFieldName(msg, field.GoName)))
				structBuilder.WriteString("\n}")
				//addToSet
				structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) AddToSet%s(items ...%s) {", msg.MsgName, field.GoName, ts))
				structBuilder.WriteString("\n\tfor _, item := range items {")
				structBuilder.WriteString(fmt.Sprintf("\n\t\tfor _, v := range this.%s {", field.GoName))
				structBuilder.WriteString("\n\t\t\tif v == item {")
				structBuilder.WriteString("\n\t\t\t\treturn")
				structBuilder.WriteString("\n\t\t\t}")
				structBuilder.WriteString("\n\t\t}")
				structBuilder.WriteString(fmt.Sprintf("\n\t\tthis.%s = append(this.%s, item)", field.GoName, field.GoName))
				structBuilder.WriteString("\n\t}")
				structBuilder.WriteString(fmt.Sprintf("\n\tthis.SetDirty(%s)", w.getFieldName(msg, field.GoName)))
				structBuilder.WriteString("\n}")
				//pull
				structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) Pull%s(items ...%s) {", msg.MsgName, field.GoName, ts))
				structBuilder.WriteString(fmt.Sprintf("\n\tif this.%s == nil || len(this.%s) == 0 {", field.GoName, field.GoName))
				structBuilder.WriteString("\n\t\treturn")
				structBuilder.WriteString("\n\t}")
				structBuilder.WriteString("\n\tdirty := false")
				structBuilder.WriteString("\n\tfor _, item := range items {")
				structBuilder.WriteString(fmt.Sprintf("\n\tfor i, v := range this.%s {", field.GoName))
				structBuilder.WriteString("\n\t\tif v == item {")
				structBuilder.WriteString(fmt.Sprintf("\n\t\t\tthis.%s = append(this.%s[:i], this.%s[i+1:]...)", field.GoName, field.GoName, field.GoName))
				structBuilder.WriteString("\n\t\t\tdirty = true")
				structBuilder.WriteString("\n\t\t\tbreak")
				structBuilder.WriteString("\n\t\t}")
				structBuilder.WriteString("\n\t\t}")
				structBuilder.WriteString("\n\t}")
				structBuilder.WriteString("\n\tif dirty {")
				structBuilder.WriteString(fmt.Sprintf("\n\t\tthis.SetDirty(%s)", w.getFieldName(msg, field.GoName)))
				structBuilder.WriteString("\n\t}")
				structBuilder.WriteString("\n}")
			}
		} else {
			if w.writeFiledCost(structCostBuilder, field, "this") {
				importUnsafe = true
			}
			switch field.Desc.Kind() {
			case protoreflect.BytesKind:
				structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) Set%s(val []byte) {", msg.MsgName, field.GoName))
			case protoreflect.EnumKind:
				structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) Set%s(val pb.%s) {", msg.MsgName, field.GoName, field.Enum.GoIdent.GoName))
			case protoreflect.MessageKind:
				structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) Set%s(val *pb.%s) {", msg.MsgName, field.GoName, field.Message.Desc.Name()))
			default:
				structBuilder.WriteString(fmt.Sprintf("\n\n\tfunc (this *%s) Set%s(val %s) {", msg.MsgName, field.GoName, field.Desc.Kind()))
			}
			structBuilder.WriteString(fmt.Sprintf("\n\tthis.%s = val", field.GoName))
			structBuilder.WriteString(fmt.Sprintf("\n\tthis.SetDirty(%s)", w.getFieldName(msg, field.GoName)))
			structBuilder.WriteString("\n}")
		}
	}

	if importUnsafe {
		w.headBuilder.WriteString("\n\t\"unsafe\"")
	}
	if importBson {
		w.headBuilder.WriteString("\n\t\"go.mongodb.org/mongo-driver/bson\"")
	}

	mapBuilder.WriteString("\n)")
	delMapBuilder.WriteString("\n}")
	newFnBuilder.WriteString("\n\t)")
	newFnBuilder.WriteString("\n\treturn m")
	newFnBuilder.WriteString("\n}")
	setBuilder.WriteString("\n}")
	structCostBuilder.WriteString("\n\treturn cost")
	structCostBuilder.WriteString("\n}")
	structGetterBuilder.WriteString("\n\t default:")
	structGetterBuilder.WriteString("\n\t\t return nil")
	structGetterBuilder.WriteString("\n\t}")
	structGetterBuilder.WriteString("\n}")

	w.msgBuilder.WriteString(mapBuilder.String() +
		storeAllBuilder.String() +
		setBuilder.String() +
		delBuilder.String() +
		evictBuilder.String() +
		delMapBuilder.String() +
		getBuilder.String() +
		structBuilder.String() +
		newFnBuilder.String() +
		structGetterBuilder.String() +
		structCostBuilder.String())

	return nil
}

func (w *modelWriter) Save() error {
	path := fmt.Sprintf("%s/model.go", w.Svc().Name)
	return w.save(path, w.headBuilder.String()+
		w.initBuilder.String()+
		w.modelFacBuilder.String()+
		w.evictBuilder.String()+
		w.msgBuilder.String())
}

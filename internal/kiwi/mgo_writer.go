package kiwi

import (
	"fmt"
	"github.com/15mga/kiwi/util"
	tool "github.com/15mga/kiwi_tool"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"sort"
	"strings"
)

func NewMgoWriter() IWriter {
	return &mgoWriter{}
}

type mgoWriter struct {
	baseWriter
	importBuilder *strings.Builder
	schemaBuilder *strings.Builder
	initBuilder   *strings.Builder
	fieldBuilder  *strings.Builder
	idxBuilder    *strings.Builder
	importBson    bool
	importMgo     bool
	importOptions bool
	fields        map[*Msg][]*protogen.Field
}

func (w *mgoWriter) Reset() {
	w.importBuilder = &strings.Builder{}
	w.schemaBuilder = &strings.Builder{}
	w.initBuilder = &strings.Builder{}
	w.fieldBuilder = &strings.Builder{}
	w.idxBuilder = &strings.Builder{}
	w.fields = make(map[*Msg][]*protogen.Field)
	w.importBson = false
	w.importMgo = false
	w.importOptions = false
}

func (w *mgoWriter) WriteHeader() {
	w.importBuilder.WriteString("package " + w.Svc().Name)
	w.importBuilder.WriteString("\n\nimport (")
	w.schemaBuilder.WriteString("\n\nconst (")
	w.initBuilder.WriteString("\n\nfunc initColl() {")
}

type mgoFieldData struct {
	msg    *Msg
	fields []*protogen.Field
}

func (w *mgoWriter) WriteFooter() {
	w.fieldBuilder.WriteString("\n\nconst (")
	fieldDataSlc := make([]*mgoFieldData, 0, len(w.fields))
	for msg, fields := range w.fields {
		fieldDataSlc = append(fieldDataSlc, &mgoFieldData{
			msg:    msg,
			fields: fields,
		})
	}
	sort.Slice(fieldDataSlc, func(i, j int) bool {
		return fieldDataSlc[i].msg.MsgName < fieldDataSlc[j].msg.MsgName
	})
	for _, f := range fieldDataSlc {
		msg, fields := f.msg, f.fields
		msgName := msg.MsgName
		if msg.Type == EMsgSch {
			sort.Slice(fields, func(i, j int) bool {
				if fields[i].GoName == "Id" {
					return true
				} else if fields[j].GoName == "Id" {
					return false
				}
				return fields[i].GoName < fields[j].GoName
			})
			if util.ToBigHump(msg.Svc.Name) == msgName {
				for _, field := range fields {
					comments := field.Comments.Leading.String()
					if comments != "" {
						w.fieldBuilder.WriteString(fmt.Sprintf("\n\t%s", strings.Trim(comments, "\n")))
					}
					if field.GoName == "Id" {
						w.fieldBuilder.WriteString(fmt.Sprintf("\n\tId = \"_id\""))
					} else {
						w.fieldBuilder.WriteString(fmt.Sprintf("\n\t%s = \"%s\"",
							field.GoName, util.ToUnderline(field.GoName)))
					}
					comments = field.Comments.Trailing.String()
					if comments != "" {
						w.fieldBuilder.WriteString(fmt.Sprintf("%s", strings.Trim(comments, "\n")))
					}
				}
			} else {
				for _, field := range fields {
					comments := field.Comments.Leading.String()
					if comments != "" {
						w.fieldBuilder.WriteString(fmt.Sprintf("\n\t%s", strings.Trim(comments, "\n")))
					}
					if field.GoName == "Id" {
						w.fieldBuilder.WriteString(fmt.Sprintf("\n\t%sId = \"_id\"", msgName))
					} else {
						w.fieldBuilder.WriteString(fmt.Sprintf("\n\t%s%s = \"%s\"", msgName,
							field.GoName, util.ToUnderline(field.GoName)))
					}
					comments = field.Comments.Trailing.String()
					if comments != "" {
						w.fieldBuilder.WriteString(fmt.Sprintf("%s", strings.Trim(comments, "\n")))
					}
				}
			}
		} else {
			sort.Slice(fields, func(i, j int) bool {
				return fields[i].GoName < fields[j].GoName
			})

			for _, field := range fields {
				comments := field.Comments.Leading.String()
				if comments != "" {
					w.fieldBuilder.WriteString(fmt.Sprintf("\n\t%s", strings.Trim(comments, "\n")))
				}
				w.fieldBuilder.WriteString(fmt.Sprintf("\n\t%s%s = \"%s\"", msgName,
					field.GoName, util.ToUnderline(field.GoName)))
				comments = field.Comments.Trailing.String()
				if comments != "" {
					w.fieldBuilder.WriteString(fmt.Sprintf("%s", strings.Trim(comments, "\n")))
				}
			}
		}
		w.fieldBuilder.WriteString("\n")
	}
	w.fieldBuilder.WriteString(")")
	w.importBuilder.WriteString("\n)")
	w.schemaBuilder.WriteString("\n)")
	w.initBuilder.WriteString("\n}")
}

func (w *mgoWriter) WriteMsg(idx int, msg *Msg) error {
	if msg.Type != EMsgNil && msg.Type != EMsgSch {
		return nil
	}
	w.SetDirty(true)
	w.addFields(msg)
	ok := isSchema(msg.Msg)
	if ok {
		w.writeImportMgo()
		w.writeSchema(msg)
		w.writeIdx(msg)
	}
	return nil
}

func (w *mgoWriter) Save() error {
	path := fmt.Sprintf("%s/mgo.go", w.Svc().Name)
	return w.save(path,
		w.importBuilder.String()+
			w.schemaBuilder.String()+
			w.fieldBuilder.String()+
			w.initBuilder.String()+
			w.idxBuilder.String(),
	)
}

func isSchema(msg *protogen.Message) bool {
	return proto.GetExtension(msg.Desc.Options(), tool.E_Schema).(bool)
}

func (w *mgoWriter) writeSchema(msg *Msg) {
	msgName := msg.Msg.GoIdent.GoName
	w.schemaBuilder.WriteString(fmt.Sprintf("\n\tSchema%s = \"%s\"", msgName, util.ToUnderline(msgName)))
	w.initBuilder.WriteString(fmt.Sprintf("\n\tmgo.InitColl(Schema%s, %sIdx)", msgName, msgName))
}

func (w *mgoWriter) addFields(msg *Msg) {
	slc := make([]*protogen.Field, 0, len(msg.Msg.Fields))
	for _, f := range msg.Msg.Fields {
		slc = append(slc, f)
	}
	w.fields[msg] = slc
}

func (w *mgoWriter) writeIdx(msg *Msg) {
	msgName := msg.Msg.GoIdent.GoName
	w.idxBuilder.WriteString(fmt.Sprintf("\n\nfunc %sIdx() []mongo.IndexModel {", msgName))
	w.idxBuilder.WriteString("\n\treturn []mongo.IndexModel{")

	for _, field := range msg.Msg.Fields {
		if field.GoName == "Id" {
			continue
		}
		cache := proto.GetExtension(field.Desc.Options(), tool.E_Cache).(bool)
		if !cache {
			continue
		}
		w.writeImportBson()
		w.writeImportOptions()

		w.idxBuilder.WriteString("\n\t\t{")
		w.idxBuilder.WriteString("\n\t\t\tKeys: bson.D{")
		w.idxBuilder.WriteString(fmt.Sprintf("\n\t\t\t{\"%s\", 1},", util.ToUnderline(field.GoName)))
		w.idxBuilder.WriteString("\n\t\t\t},")
		w.idxBuilder.WriteString("\n\t\t\tOptions: options.Index().SetUnique(true),")
		w.idxBuilder.WriteString("\n\t\t},")
	}

	idxSlc := proto.GetExtension(msg.Msg.Desc.Options(), tool.E_Idx).([]*tool.Idx)
	for _, idx := range idxSlc {
		if len(idx.Fields) == 0 {
			continue
		}
		w.writeImportBson()
		w.idxBuilder.WriteString("\n\t\t{")
		w.idxBuilder.WriteString("\n\t\t\tKeys: bson.D{")
		for _, f := range idx.Fields {
			switch f.Type {
			case tool.EIdx_Asc:
				w.idxBuilder.WriteString(fmt.Sprintf("\n\t\t\t{\"%s\", 1},", f.Name))
			case tool.EIdx_Desc:
				w.idxBuilder.WriteString(fmt.Sprintf("\n\t\t\t{\"%s\", -1},", f.Name))
			case tool.EIdx_Text:
				w.idxBuilder.WriteString(fmt.Sprintf("\n\t\t\t{\"%s\", \"text\"},", f.Name))
			case tool.EIdx_TwoDSphere:
				w.idxBuilder.WriteString(fmt.Sprintf("\n\t\t\t{\"%s\", \"2dsphere\"},", f.Name))
			}
		}
		w.idxBuilder.WriteString("\n\t\t\t},")

		optBuilder := &strings.Builder{}
		writeOpt := false
		optBuilder.WriteString("\n\t\t\tOptions: options.Index()")
		if idx.Name != "" {
			writeOpt = true
			w.writeImportOptions()
			optBuilder.WriteString(fmt.Sprintf(".\n\t\t\t\tSetName(\"%s\")", idx.Name))
		}
		if idx.Unique {
			writeOpt = true
			w.writeImportOptions()
			optBuilder.WriteString(".\n\t\t\t\tSetUnique(true)")
		}
		if idx.Ttl > 0 {
			writeOpt = true
			w.writeImportOptions()
			optBuilder.WriteString(fmt.Sprintf(".\n\t\t\t\tSetExpireAfterSeconds(%d)", idx.Ttl))
		}
		if idx.Sparse {
			writeOpt = true
			w.writeImportOptions()
			optBuilder.WriteString(".\n\t\t\t\tSetSparse(true)")
		}
		if writeOpt {
			w.idxBuilder.WriteString(optBuilder.String())
			w.idxBuilder.WriteString(",")
		}

		w.idxBuilder.WriteString("\n\t\t},")
	}

	w.idxBuilder.WriteString("\n\t}")
	w.idxBuilder.WriteString("\n}")
}

func (w *mgoWriter) writeImportBson() {
	if w.importBson {
		return
	}
	w.importBson = true
	w.importBuilder.WriteString("\n\"go.mongodb.org/mongo-driver/bson\"")
}

func (w *mgoWriter) writeImportOptions() {
	if w.importOptions {
		return
	}
	w.importOptions = true
	w.importBuilder.WriteString("\n\"go.mongodb.org/mongo-driver/mongo/options\"")
}

func (w *mgoWriter) writeImportMgo() {
	if w.importMgo {
		return
	}
	w.importMgo = true
	w.importBuilder.WriteString("\n\"github.com/15mga/kiwi/util/mgo\"")
	w.importBuilder.WriteString("\n\"go.mongodb.org/mongo-driver/mongo\"")
}

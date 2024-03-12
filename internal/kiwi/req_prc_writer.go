package kiwi

import (
	"fmt"
	tool "github.com/15mga/kiwi_tool"
	"strings"

	"github.com/15mga/kiwi/util"
)

func NewReqPrcWriter() IWriter {
	return &pusReqPrcWriter{}
}

type pusReqPrcWriter struct {
	baseWriter
	headBuilder *strings.Builder
	svcBuilder  *strings.Builder
	public      map[string]struct{}
}

func (w *pusReqPrcWriter) Reset() {
	w.public = make(map[string]struct{})
	w.headBuilder = &strings.Builder{}
	w.svcBuilder = &strings.Builder{}
}

func (w *pusReqPrcWriter) WriteHeader() {
	w.headBuilder.WriteString("package " + w.Svc().Name)
	w.headBuilder.WriteString("\n\nimport (")
	w.headBuilder.WriteString("\n\t\"github.com/15mga/kiwi\"")
	w.headBuilder.WriteString(fmt.Sprintf("\n\t\"%s/internal/common\"", w.Module()))

	w.svcBuilder.WriteString("\n\nfunc (s *svc) registerReq() {")
}

func (w *pusReqPrcWriter) WriteMsg(idx int, msg *Msg) {
	svcStr := "_svc"
	svcBuilder := w.svcBuilder
	onStr := fmt.Sprintf("%s.%s", svcStr, HandlerPrefix)
	writeUtil := false
	writeHead := false
	bigSvcName := util.ToBigHump(msg.Svc.Name)
	switch msg.Type {
	case EMsgReq:
		writeHead = true
		name := msg.Name
		svcBuilder.WriteString(fmt.Sprintf("\n\tkiwi.Router().BindReq(common.%s, %s, func(req kiwi.IRcvRequest){",
			util.ToBigHump(msg.Svc.Name), name))
		worker := msg.GetWorker()
		switch worker.Mode {
		case tool.EWorker_Go:
			svcBuilder.WriteString(fmt.Sprintf("\n\t\tcore.GoPrcReq[*pb.%s](req, %s%s)",
				name, onStr, msg.Method))
		case tool.EWorker_Active:
			switch worker.Origin {
			case tool.EOrigin_Head:
				writeUtil = true
				svcBuilder.WriteString(fmt.Sprintf("\n\t\tkey, _ := util.MGet[string](req.Head(), \"%s\")", worker.Key))
				svcBuilder.WriteString(fmt.Sprintf("\n\t\tcore.ActivePrcReq[*pb.%s](req, key, %s%s)",
					name, onStr, msg.Method))
			case tool.EOrigin_Pkt:
				svcBuilder.WriteString(fmt.Sprintf("\n\t\tkey := req.Msg().(*pb.%s).%s", name, util.ToBigHump(worker.Key)))
				svcBuilder.WriteString(fmt.Sprintf("\n\t\tcore.ActivePrcReq[*pb.%s](req, key, %s%s)",
					name, onStr, msg.Method))
			case tool.EOrigin_Service:
				svcBuilder.WriteString(fmt.Sprintf("\n\t\tcore.ActivePrcReq[*pb.%s](req, common.S%s, %s%s)",
					name, bigSvcName, onStr, msg.Method))
			}
		case tool.EWorker_Share:
			switch worker.Origin {
			case tool.EOrigin_Head:
				writeUtil = true
				svcBuilder.WriteString(fmt.Sprintf("\n\t\tkey, _ := util.MGet[string](req.Head(), \"%s\")", worker.Key))
				svcBuilder.WriteString(fmt.Sprintf("\n\t\tcore.SharePrcReq[*pb.%s](req,  key, %s%s)",
					name, onStr, msg.Method))
			case tool.EOrigin_Pkt:
				svcBuilder.WriteString(fmt.Sprintf("\n\t\tkey := req.Msg().(*pb.%s).%s", name, util.ToBigHump(worker.Key)))
				svcBuilder.WriteString(fmt.Sprintf("\n\t\tcore.SharePrcReq[*pb.%s](req,  key, %s%s)",
					name, onStr, msg.Method))
			case tool.EOrigin_Service:
				svcBuilder.WriteString(fmt.Sprintf("\n\t\tcore.SharePrcReq[*pb.%s](req, common.S%s, %s%s)",
					name, bigSvcName, onStr, msg.Method))
			}
		case tool.EWorker_Global:
			svcBuilder.WriteString(fmt.Sprintf("\n\t\tcore.GlobalPrcReq[*pb.%s](req, %s%s)",
				name, onStr, msg.Method))
		case tool.EWorker_Self:
			svcBuilder.WriteString(fmt.Sprintf("\n\t\tcore.SelfPrcReq[*pb.%s](req, %s%s)",
				name, onStr, msg.Method))
		}
		svcBuilder.WriteString("\n\t})")
	default:
		return
	}
	if writeHead {
		w.headBuilder.WriteString("\n\t\"github.com/15mga/kiwi/core\"")
		w.headBuilder.WriteString(fmt.Sprintf("\n\t\"%s/proto/pb\"", w.Module()))
	}
	if writeUtil {
		w.headBuilder.WriteString("\n\t\"github.com/15mga/kiwi/util\"")
	}
	w.SetDirty(true)
}

func (w *pusReqPrcWriter) WriteFooter() {
	w.headBuilder.WriteString("\n)")
	w.writeSvcFoot(w.svcBuilder)
}

func (w *pusReqPrcWriter) writeSvcFoot(builder *strings.Builder) {
	builder.WriteString("\n}")
}

func (w *pusReqPrcWriter) Save() error {
	path := fmt.Sprintf("%s/req_prc.go", w.Svc().Name)
	return w.save(path, w.headBuilder.String()+w.svcBuilder.String())
}

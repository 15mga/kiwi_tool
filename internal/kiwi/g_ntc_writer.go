package kiwi

import (
	"fmt"
	"github.com/15mga/kiwi/util"
	tool "github.com/15mga/kiwi_tool"
	"strings"
)

func NewGNtcWriter() IWriter {
	return &gNtcWriter{}
}

type ntcWatcher struct {
	watcher string
	worker  *tool.NtcItem
}

type gNtcWriter struct {
	baseWriter
	watcherToNtc map[string][]*tool.Ntc
	ntcToWatcher map[string][]*ntcWatcher
}

func (w *gNtcWriter) Reset() {
	w.watcherToNtc = make(map[string][]*tool.Ntc)
	w.ntcToWatcher = make(map[string][]*ntcWatcher)
}

func (w *gNtcWriter) SetSvc(svc *svc) {
	w.baseWriter.SetSvc(svc)
	svcName := svc.Name
	w.SetDirty(true)
	for _, ntc := range svc.WatchNtc {
		w.watcherToNtc[svcName] = append(w.watcherToNtc[svcName], ntc)
		for _, worker := range ntc.Items {
			w.ntcToWatcher[worker.Code] = append(w.ntcToWatcher[worker.Code], &ntcWatcher{
				watcher: svcName,
				worker:  worker,
			})
		}
	}
}

func (w *gNtcWriter) Save() error {
	for svc, slc := range w.watcherToNtc {
		bigSvcName := util.ToBigHump(svc)
		ntcBuilder := &strings.Builder{}
		ntcBuilder.WriteString("package " + svc)
		ntcBuilder.WriteString("\n\nimport (")
		ntcBuilder.WriteString(fmt.Sprintf("\n\t\"%s/proto/pb\"", w.Module()))
		ntcBuilder.WriteString("\n\t\"github.com/15mga/kiwi\"")
		ntcBuilder.WriteString("\n\t\"github.com/15mga/kiwi/util\"")
		ntcBuilder.WriteString("\n)")

		headBuilder := &strings.Builder{}
		headBuilder.WriteString("package " + svc)
		watchBuilder := &strings.Builder{}
		watchBuilder.WriteString("\n\nfunc watchNtc() {")
		writeImport := false
		writeUtil := false
		writeCommon := false
		for _, ntc := range slc {
			for _, worker := range ntc.Items {
				c := worker.Code
				ntcBuilder.WriteString(fmt.Sprintf("\n\nfunc (s *svc) %s%s(pkt kiwi.IRcvNotice, ntc *pb.%s) {",
					HandlerPrefix, c, c))
				ntcBuilder.WriteString("\n\tpkt.Err2(util.EcNotImplement, util.M{\"ntc\": ntc})")
				ntcBuilder.WriteString("\n}")

				watchBuilder.WriteString(fmt.Sprintf("\n\t_svc.WatchNotice(&pb.%s{}, func(ntc kiwi.IRcvNotice) {",
					c))
				switch worker.Mode {
				case tool.EWorker_Go:
					watchBuilder.WriteString(fmt.Sprintf("\n\t\tcore.GoPrcNtc[*pb.%s](ntc, _svc.%s%s)",
						c, HandlerPrefix, c))
				case tool.EWorker_Active:
					switch worker.Origin {
					case tool.EOrigin_Head:
						writeUtil = true
						watchBuilder.WriteString(fmt.Sprintf("\n\t\tkey, ok := util.MGet[string](ntc.Head(), \"%s\")", worker.Key))
						watchBuilder.WriteString("\n\t\tif ok {")
						watchBuilder.WriteString(fmt.Sprintf("\n\t\tcore.ActivePrcNtc[*pb.%s](ntc, key, _svc.%s%s)",
							c, HandlerPrefix, c))
						watchBuilder.WriteString("\n\t\t}")
					case tool.EOrigin_Pkt:
						watchBuilder.WriteString(fmt.Sprintf("\n\t\tkey := ntc.Msg().(*pb.%s).%s", c, util.ToBigHump(worker.Key)))
						watchBuilder.WriteString("\n\t\tif key != \"\" {")
						watchBuilder.WriteString(fmt.Sprintf("\n\t\tcore.ActivePrcNtc[*pb.%s](ntc, key, _svc.%s%s)",
							c, HandlerPrefix, c))
						watchBuilder.WriteString("\n\t\t}")
					case tool.EOrigin_Service:
						watchBuilder.WriteString(fmt.Sprintf("\n\t\tcore.ActivePrcNtc[*pb.%s](ntc, common.S%s, _svc.%s%s)",
							c, bigSvcName, HandlerPrefix, c))
						writeCommon = true
					}
				case tool.EWorker_Share:
					switch worker.Origin {
					case tool.EOrigin_Head:
						writeUtil = true
						watchBuilder.WriteString(fmt.Sprintf("\n\t\tkey, ok := util.MGet[string](ntc.Data(), \"%s\")", worker.Key))
						watchBuilder.WriteString("\n\t\tif ok {")
						watchBuilder.WriteString(fmt.Sprintf("\n\t\tcore.SharePrcNtc[*pb.%s](ntc, key, _svc.%s%s)",
							c, HandlerPrefix, c))
						watchBuilder.WriteString("\n\t\t}")
					case tool.EOrigin_Pkt:
						watchBuilder.WriteString(fmt.Sprintf("\n\t\tkey := ntc.Msg().(*pb.%s).%s", c, util.ToBigHump(worker.Key)))
						watchBuilder.WriteString("\n\t\tif key != \"\" {")
						watchBuilder.WriteString(fmt.Sprintf("\n\t\tcore.SharePrcNtc[*pb.%s](ntc, key, _svc.%s%s)",
							c, HandlerPrefix, c))
						watchBuilder.WriteString("\n\t\t}")
					case tool.EOrigin_Service:
						watchBuilder.WriteString(fmt.Sprintf("\n\t\tcore.SharePrcNtc[*pb.%s](ntc, common.S%s, _svc.%s%s)",
							c, bigSvcName, HandlerPrefix, c))
						writeCommon = true
					}
				case tool.EWorker_Global:
					watchBuilder.WriteString(fmt.Sprintf("\n\t\tcore.GlobalPrcNtc[*pb.%s](ntc, _svc.%s%s)",
						c, HandlerPrefix, c))
				case tool.EWorker_Self:
					watchBuilder.WriteString(fmt.Sprintf("\n\t\tcore.SelfPrcNtc[*pb.%s](ntc, _svc.%s%s)",
						c, HandlerPrefix, c))
				}
				watchBuilder.WriteString("\n\t})")
				writeImport = true
			}
		}

		if writeImport {
			headBuilder.WriteString("\n\nimport (")
			headBuilder.WriteString(fmt.Sprintf("\n\t\"%s/proto/pb\"", w.Module()))
			headBuilder.WriteString("\n\t\"github.com/15mga/kiwi\"")
			headBuilder.WriteString("\n\t\"github.com/15mga/kiwi/core\"")
			if writeUtil {
				headBuilder.WriteString("\n\t\"github.com/15mga/kiwi/util\"")
			}
			if writeCommon {
				headBuilder.WriteString(fmt.Sprintf("\n\t\"%s/internal/common\"", w.Module()))
			}
			headBuilder.WriteString("\n)")
		}

		watchBuilder.WriteString("\n}")

		if len(slc) > 0 {
			path := fmt.Sprintf("%s/ntc_gen.go", svc)
			err := w.save(path, ntcBuilder.String())
			if err != nil {
				return err
			}
		}

		path := fmt.Sprintf("%s/ntc_prc.go", svc)
		err := w.save(path, headBuilder.String()+watchBuilder.String())
		if err != nil {
			return err
		}
	}
	return nil
}

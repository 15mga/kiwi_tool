package kiwi

import (
	"fmt"
	"strings"

	"github.com/15mga/kiwi/util"
)

func NewGSvcWriter() IWriter {
	return &gSvcWriter{}
}

type gSvcWriter struct {
	baseWriter
}

func (w *gSvcWriter) Save() error {
	constBuilder := &strings.Builder{}
	nameBuilder := &strings.Builder{}
	svcToNameBuilder := &strings.Builder{}
	nameToSvcBuilder := &strings.Builder{}

	constBuilder.WriteString("package common")
	constBuilder.WriteString("\n\nimport (")
	constBuilder.WriteString("\n\"github.com/15mga/kiwi\"")
	constBuilder.WriteString("\n)")
	constBuilder.WriteString("\n\nconst (")

	nameBuilder.WriteString("\n\nconst (")

	svcToNameBuilder.WriteString("\n\nvar SvcToName = map[kiwi.TSvc]string{")

	nameToSvcBuilder.WriteString("\n\nvar NameToSvc = map[string]kiwi.TSvc{")

	for _, svc := range w.Builder().svcSlc {
		svcName := svc.Name
		bigSvcName := util.ToBigHump(svcName)
		constBuilder.WriteString(fmt.Sprintf("\n\t%s kiwi.TSvc = %d", bigSvcName, svc.Id))
		nameBuilder.WriteString(fmt.Sprintf("\n\tS%s = \"%s\"", bigSvcName, svcName))
		svcToNameBuilder.WriteString(fmt.Sprintf("\n\t\t%s : S%s,", bigSvcName, bigSvcName))
		nameToSvcBuilder.WriteString(fmt.Sprintf("\n\t\tS%s : %s,", bigSvcName, bigSvcName))
	}

	constBuilder.WriteString("\n)")
	nameBuilder.WriteString("\n)")
	svcToNameBuilder.WriteString("\n\t}")
	nameToSvcBuilder.WriteString("\n\t}")

	return w.save("/common/svc.go", constBuilder.String()+
		nameBuilder.String()+
		svcToNameBuilder.String()+
		nameToSvcBuilder.String())
}

func (w *gSvcWriter) SetSvc(svc *svc) {
	w.SetDirty(true)
}

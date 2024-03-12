package kiwi

import (
	"flag"
	"google.golang.org/protobuf/compiler/protogen"
	"strings"
)

func Run() {
	protogen.Options{}.Run(func(plugin *protogen.Plugin) error {
		params := plugin.Request.GetParameter()
		var flags flag.FlagSet
		module := flags.String("m", "game", "module name")
		roleStr := flags.String("r", "player", "player role")
		db := flags.String("db", "mgo", "database")
		slc := strings.Split(params, ",")
		err := flags.Parse(slc)
		if err != nil {
			return err
		}
		for _, f := range plugin.Files {
			if !f.Generate {
				continue
			}
			err := addSvc(f)
			if err != nil {
				return err
			}
		}
		roles := strings.Split(*roleStr, "_")
		for i, role := range roles {
			roles[i] = strings.TrimSpace(role)
		}
		return newBuilder(plugin, *module, *db, *roleStr).build()
	})
}

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
		roles := strings.Split(*roleStr, "_")
		roleMap := make(map[string]struct{}, len(roles))
		for _, role := range roles {
			roleMap[strings.TrimSpace(role)] = struct{}{}
		}
		return newBuilder(plugin, *module, *db, roleMap).build()
	})
}

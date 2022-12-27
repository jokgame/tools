package main

import (
	"github.com/gopherd/log"
	"github.com/midlang/mid/src/mid/build"

	"github.com/jokgame/tools/autoconf/xlsx"
)

func main() {
	log.SetLevel(log.LevelWarn)
	plugin, config, builder, err := build.ParseFlags()
	if err != nil {
		panic(err)
	}
	err = generate(builder, plugin, config)
	if err != nil {
		panic(err)
	}
}

func generate(builder *build.Builder, plugin build.Plugin, config build.PluginRuntimeConfig) (err error) {
	pkgs := builder.Packages
	for _, pkg := range pkgs {
		if err := xlsx.GenerateXlsx(plugin, config, pkg); err != nil {
			return err
		}
	}
	return nil
}

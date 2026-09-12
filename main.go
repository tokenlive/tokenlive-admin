package main

import (
	"os"

	"github.com/tokenlive/tokenlive-admin/cmd"
	"github.com/urfave/cli/v2"
)

// Usage: go build -ldflags "-X main.VERSION=x.x.x -X main.BUILD_KIND=release"
var VERSION = "dev"
var BUILD_KIND = "dev"

// @title tokenlive-admin
// @version v1.0.0
// @description An admin control center for tokelvie.
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @schemes http https
// @basePath /
func main() {
	app := cli.NewApp()
	app.Name = "tokenlive-admin"
	app.Version = VERSION
	app.Usage = "An admin control center for tokelvie."
	app.Commands = []*cli.Command{
		cmd.StartCmd(VERSION, BUILD_KIND),
		cmd.StopCmd(),
		cmd.VersionCmd(VERSION),
	}
	err := app.Run(os.Args)
	if err != nil {
		println(err.Error())
		panic(err)
	}
}

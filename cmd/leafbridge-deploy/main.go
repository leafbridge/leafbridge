package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kong"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var cli struct {
		Deploy  DeployCmd  `kong:"cmd,help='Deploys a particular software package.'"`
		Show    ShowCmd    `kong:"cmd,help='Shows information about a deployment.'"`
		Version VersionCmd `kong:"cmd,help='Display leafbridge-deploy version information.'"`
	}

	parser := kong.Must(&cli,
		kong.Description("Deploys software to computers."),
		kong.BindTo(ctx, (*context.Context)(nil)),
		kong.UsageOnError())

	app, parseErr := parser.Parse(os.Args[1:])
	parser.FatalIfErrorf(parseErr)

	appErr := app.Run()
	app.FatalIfErrorf(appErr)
}

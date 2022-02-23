package main

import (
	"os"

	gotip "github.com/ForestEckhardt/gotip"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/cargo"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/draft"
	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

func main() {
	packit.Run(
		gotip.Detect(),
		gotip.Build(
			draft.NewPlanner(),
			postal.NewService(cargo.NewTransport()),
			pexec.NewExecutable("go"),
			pexec.NewExecutable("gotip"),
			chronos.DefaultClock,
			scribe.NewEmitter(os.Stdout),
		),
	)
}

package gotip

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/fs"
	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

//go:generate faux --interface EntryResolver --output fakes/entry_resolver.go
type EntryResolver interface {
	MergeLayerTypes(name string, entries []packit.BuildpackPlanEntry) (launch, build bool)
}

//go:generate faux --interface DependencyManager --output fakes/dependency_manager.go
type DependencyManager interface {
	Resolve(path, id, version, stack string) (postal.Dependency, error)
	Deliver(dependency postal.Dependency, cnbPath, layerPath, platformPath string) error
}

//go:generate faux --interface Executable --output fakes/executable.go
type Executable interface {
	Execute(execution pexec.Execution) error
}

func Build(
	entryResolver EntryResolver,
	dependencyManager DependencyManager,
	goExecutable Executable,
	gotipExecutable Executable,
	clock chronos.Clock,
	logs scribe.Emitter,
) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logs.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		// Cache this layer to make rebuilds slightly faster
		tempGoLayer, err := context.Layers.Get("temp-go")
		if err != nil {
			return packit.BuildResult{}, err
		}

		tempGoLayer, err = tempGoLayer.Reset()
		if err != nil {
			return packit.BuildResult{}, err
		}

		dependency, err := dependencyManager.Resolve(filepath.Join(context.CNBPath, "buildpack.toml"), "go", "default", context.Stack)
		if err != nil {
			return packit.BuildResult{}, err
		}

		logs.Process("Executing build process")

		logs.Subprocess("Installing Go %s", dependency.Version)
		duration, err := clock.Measure(func() error {
			return dependencyManager.Deliver(dependency, context.CNBPath, tempGoLayer.Path, context.Platform.Path)
		})
		if err != nil {
			return packit.BuildResult{}, err
		}
		logs.Action("Completed in %s", duration.Round(time.Millisecond))
		logs.Break()

		env := os.Environ()

		tempPath := fmt.Sprintf("%s:%s", filepath.Join(tempGoLayer.Path, "bin"), os.Getenv("PATH"))

		tempGopath, err := os.MkdirTemp(tempGoLayer.Path, "temp-gopath")
		if err != nil {
			return packit.BuildResult{}, err
		}

		env = setOrOverride(env, "PATH", tempPath)
		env = setOrOverride(env, "GOPATH", tempGopath)

		logs.Process("Installing gotip")

		args := []string{"install", "golang.org/dl/gotip@latest"}

		logs.Subprocess("Running go %s", strings.Join(args, " "))
		buffer := bytes.NewBuffer(nil)
		duration, err = clock.Measure(func() error {
			return goExecutable.Execute(pexec.Execution{
				Args:   args,
				Dir:    context.WorkingDir,
				Env:    env,
				Stdout: buffer,
				Stderr: buffer,
			})
		})
		if err != nil {
			logs.Action("Failed after %s", duration.Round(time.Millisecond))
			logs.Detail(buffer.String())

			return packit.BuildResult{}, fmt.Errorf("failed to execute 'go install': %w", err)
		}

		logs.Action("Completed in %s", duration.Round(time.Millisecond))
		logs.Break()

		logs.Process("Running gotip")

		args = []string{"download"}

		tempPath = fmt.Sprintf("%s:%s", filepath.Join(tempGopath, "bin"), tempPath)

		tempHome, err := os.MkdirTemp(tempGoLayer.Path, "temp")
		if err != nil {
			return packit.BuildResult{}, err
		}

		env = setOrOverride(env, "PATH", tempPath)
		env = setOrOverride(env, "HOME", tempHome)

		logs.Subprocess("Running gotip %s", strings.Join(args, " "))
		buffer = bytes.NewBuffer(nil)
		duration, err = clock.Measure(func() error {
			return gotipExecutable.Execute(pexec.Execution{
				Args:   args,
				Dir:    context.WorkingDir,
				Env:    env,
				Stdout: buffer,
				Stderr: buffer,
			})
		})
		if err != nil {
			logs.Action("Failed after %s", duration.Round(time.Millisecond))
			logs.Detail(buffer.String())

			return packit.BuildResult{}, fmt.Errorf("failed to execute 'gotip download': %w", err)
		}

		logs.Action("Completed in %s", duration.Round(time.Millisecond))
		logs.Break()

		goLayer, err := context.Layers.Get("go")
		if err != nil {
			return packit.BuildResult{}, err
		}

		goLayer, err = goLayer.Reset()
		if err != nil {
			return packit.BuildResult{}, err
		}

		launch, build := entryResolver.MergeLayerTypes("go", context.Plan.Entries)
		goLayer.Launch, goLayer.Build, goLayer.Cache = launch, build, build

		dirEntries, err := os.ReadDir(filepath.Join(tempHome, "sdk", "gotip"))
		if err != nil {
			return packit.BuildResult{}, err
		}

		for _, e := range dirEntries {
			if strings.HasPrefix(e.Name(), ".") {
				continue
			}
			err = fs.Move(filepath.Join(tempHome, "sdk", "gotip", e.Name()), filepath.Join(goLayer.Path, e.Name()))
			if err != nil {
				return packit.BuildResult{}, err
			}
		}

		return packit.BuildResult{
			Layers: []packit.Layer{goLayer},
		}, nil
	}
}

func setOrOverride(env []string, name, val string) []string {
	var set bool
	for i, e := range env {
		splitEnv := strings.Split(e, "=")
		if splitEnv[0] == name {
			env[i] = fmt.Sprintf("%s=%s", name, val)
			set = true
			break
		}
	}

	if !set {
		env = append(env, fmt.Sprintf("%s=%s", name, val))
	}

	return env
}

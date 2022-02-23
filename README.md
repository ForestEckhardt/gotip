# Gotip buildpack

:rotating_light::rotating_light::rotating_light::warning: WARNING THIS REPO IS EXPIREMENTAL AND HIGHLY UNTESTED USE AT YOUR OWN RISK:warning::rotating_light::rotating_light::rotating_light:

This buildpack installs a version of Go using [`gotip`](https://pkg.go.dev/golang.org/dl/gotip).

## Usage

To use this buildpacks it must first be packaged. Run the following command:
```bash
./scripts/package.sh --version 1.2.3
```

The output of the command will indicate where the built artifacts are on the
filesystem. From there, either artifact (the tar ball or the buildpackage) can
be used with the `--buildpack` flag in `pack`

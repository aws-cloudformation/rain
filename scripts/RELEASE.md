# Release procedure

[ ] - Increment the version number in `internal/config/version.go`
[ ] - Run `go get -u ./...` to update all dependencies
[ ] - Update `docs/README.tmpl` with any new features
[ ] - Generate cached schemas: `./scripts/cache-schemas.sh internal/aws/cfn/schemas`
[ ] - Generate docs: `go generate ./...`
[ ] - Run integ tests: `./scripts/integ.sh`
[ ] - Create a PR and merge it
[ ] - Create a new release and tag with the same name, starting with v, like `v1.8.5`
[ ] - Auto generate release notes, then edit them to clean them up
[ ] - Check the Actions tab to make sure the release scripts worked
[ ] - Announce on Discord and social media if there are significant new features


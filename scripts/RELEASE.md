# Release procedure

1. Increment the version number in `internal/config/version.go`
2. Update `docs/README.tmpl` with any new features
3. Generate cached schemas

    `./scripts/cache-schemas.sh internal/aws/cfn/schemas`

4. Generate docs

    `go generate ./...`

5. Run integ tests

    `./scripts/integ.sh`
6. Create a PR and merge it
7. Create a new release and tag with the same name, starting with v, like `v1.8.5`
8. Auto generate release notes, then edit them to clean them up
9. Check the Actions tab to make sure the release scripts worked
10. Announce on Discord and social media if there are significant new features


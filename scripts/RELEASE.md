# Release procedure

1. Increment the version number in `internal/config/version.go`
2. Update `docs/README.tmpl` with any new features
3. Generate cached schemas

    `./scripts/cache-schemas.sh internal/aws/cfn/schemas`

4. Generate Pkl classes

    `./scripts/pklgen.sh`

5. Generate docs

    `go generate ./...`

6. Run integ tests

    `./scripts/integ.sh`
7. Create a PR and merge it
8. Create a new release and tag with the same name, starting with v, like `v1.8.5`
9. Auto generate release notes, then edit them to clean them up
10. Check the Actions tab to make sure the release scripts worked
11. Announce on Discord and social media if there are significant new features


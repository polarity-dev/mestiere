# Development

Requires Go 1.27.3 or later and `git` on your `PATH`.

```sh
go test -race ./...      # unit and integration tests, using real git repositories
go vet ./...
goreleaser check         # validate the release configuration
```

Package layout and design decisions are in [Architecture](architecture.md).

## Releases

Releases are cut by pushing a `vX.Y.Z` tag. One workflow runs gofmt, vet, the tests and `registry lint`, then GoReleaser builds the archives and checksums, publishes the GitHub release and updates the cask in [polarity-dev/homebrew-tap](https://github.com/polarity-dev/homebrew-tap).

The workflow pushes to the tap with a GitHub App token, because `GITHUB_TOKEN` can't write to other repositories. It needs the `RELEASE_APP_CLIENT_ID` and `RELEASE_APP_PRIVATE_KEY` secrets, for an App installed on `polarity-dev` with contents write on `mestiere` and `homebrew-tap`.

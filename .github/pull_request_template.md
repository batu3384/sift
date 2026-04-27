## Summary

- What changed?
- Why was it needed?

## Validation

- [ ] `go test ./...`
- [ ] `go build -o ./sift ./cmd/sift`
- [ ] `make smoke`
- [ ] `./hack/security_check.sh`
- [ ] `make package-manifests TAG=v0.0.0-ci DIST_DIR=./.tmp/package-dist OUT_DIR=./.tmp/manifests` when packaging scripts or metadata changed
- [ ] `make release-preflight TAG=v0.0.0-ci DIST_DIR=./.tmp/package-dist MANIFEST_DIR=./.tmp/manifests` when packaging scripts or metadata changed
- [ ] Remote CI/security impact reviewed when `.github/workflows/**` changed

## Risk Notes

- User-facing behavior changes:
- Platform-specific caveats:
- Follow-up work:

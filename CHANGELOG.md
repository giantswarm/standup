# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [2.2.0] - 2021-01-18

### Added

- `jq` installed in Docker image.

## [2.1.0] - 2020-11-30

### Changed

- Wait for `external-dns` on AWS and Azure in `wait` task.

## [2.0.0] - 2020-11-18

### Changed

- `--kubeconfig` flag now contains a directory with separate kubeconfig files for each provider

### Removed

- Removed `--kubeconfig` flag from `create cluster` command.

## [1.1.1] - 2020-11-16

- Improve test release name generation to work with alpha / beta releases.

## [1.1.0] - 2020-10-21

### Changed

- Split the `create` command up into a `create cluster` and `create release` command.

## [1.0.0] - 2020-10-14

### Added

- Added `wait` command to wait for various components of a test cluster to be ready.
- Added `--unshallow` flag when calling `git fetch`
- Added `--release` flag to `cleanup` that specifies the release to delete. Defaults to the release of the cluster
passed via `--cluster`.

### Fixed

- Added logic for handling cluster creation errors that fail to create a cluster.
- Fixed `nil` dereference bug in `wait` command.
- Adjusted `git diff` arguments to correctly identify target files in a PR.
- Get only PR files by performing `git diff` against the merge base

### Changed

- Modified `gsctl` execution to use the binary from the current `$PATH`.
- Use `gsctl` version 0.24.0.
- Let `gsctl` write the kubeconfig directly.
- Modified to be used in tenant clusters against external control planes.
- `create` writes release ID to filesystem.
- `cleanup` tries to clean up the release passed via `--release` if cluster does not exist.
- Parse `gsctl` command output when it fails internally.
- Update `gsctl` to `0.24.4`.
- Update `kubectl` to `0.18.9`.

### Removed

- Removed `--wait` flag from `create` command.
- Removed unused `test` command.

[Unreleased]: https://github.com/giantswarm/standup/compare/v2.2.0...HEAD
[2.2.0]: https://github.com/giantswarm/standup/compare/v2.1.0...v2.2.0
[2.1.0]: https://github.com/giantswarm/standup/compare/v2.0.0...v2.1.0
[2.0.0]: https://github.com/giantswarm/standup/compare/v1.1.1...v2.0.0
[1.1.1]: https://github.com/giantswarm/standup/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/giantswarm/standup/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/giantswarm/standup/releases/tag/v1.0.0

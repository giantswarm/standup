# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Added optional flags allowing an OwnerReference to be set on test releases allowing automatic cleanup when ProwJobs 
  are garbage collected.
- Added `wait` command to wait for various components of a test cluster to be ready.

### Fixed

- Added logic for handling cluster creation errors that fail to create a cluster.

### Changed

- Modified `gsctl` execution to use the binary from the current `$PATH`.

### Removed

- Removed `--wait` flag from `create` command.
- Removed unused `test` command.

[Unreleased]: https://github.com/giantswarm/standup/tree/master

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.5] - 2026-02-05

### Added
- Unit tests for check-fstab-mounts

### Fixed
- Skip fstab entries with `null` as mountpoint
- Skip fstab entries with `noauto` mount option

## [0.1.3]

### Added
- Bonsai asset configuration (.bonsai.yml)

## [0.1.2] - 2024-01-01

### Changed
- Updated .goreleaser.yml configuration

## [0.1.1] - 2024-01-01

### Changed
- Updated release.yml workflow

## [0.1.0] - 2024-01-01

### Added
- Initial release
- Golang replacement for sensu-plugins-disk-checks
- check-disk-usage command
- check-fstab-mounts command

### Security
- Bumped github.com/golang-jwt/jwt/v4 from 4.4.2 to 4.5.2
- Bumped google.golang.org/grpc from 1.53.0 to 1.56.3
- Bumped google.golang.org/protobuf from 1.28.1 to 1.33.0
- Bumped github.com/sirupsen/logrus from 1.9.0 to 1.9.1
- Bumped golang.org/x/net from 0.7.0 to 0.38.0

# Multi-Platform Packaging Plan for gsh

This document outlines the plan to provide gsh packages for high-priority platforms, enabling users to install gsh using their native package managers.

## Table of Contents

1. [Current State](#current-state)
2. [Target Platforms](#target-platforms)
3. [Implementation Phases](#implementation-phases)
4. [Detailed Platform Plans](#detailed-platform-plans)
5. [CI/CD Integration](#cicd-integration)
6. [Implementation Checklist](#implementation-checklist)

---

## Current State

### Existing Packaging

| Platform | Status | Method |
|----------|--------|--------|
| **macOS (Homebrew)** | ✅ Active | GoReleaser auto-generates Formula |
| **Arch Linux (AUR)** | ✅ Active | GoReleaser publishes `gsh` and `gsh-bin` |
| **Nix/NixOS** | ✅ Active | `flake.nix` in repository |
| **GitHub Releases** | ✅ Active | Binary tarballs for Linux/macOS/Windows |

### High Priority Platforms to Add

| Platform | Priority | Reach |
|----------|----------|-------|
| **Ubuntu/Debian (apt)** | 🔴 High | ~40% of Linux servers |
| **Red Hat/Fedora (dnf/rpm)** | 🔴 High | ~25% of enterprise Linux |
| **Windows (winget/scoop/choco)** | 🔴 High | ~70% of desktops |

---

## Target Platforms

1. **Ubuntu/Debian** - `.deb` packages via NFPM + apt repository
2. **Red Hat/Fedora/CentOS** - `.rpm` packages via NFPM + COPR
3. **Windows** - Winget, Scoop, and Chocolatey packages
4. **macOS Homebrew** - ✅ Already implemented
5. **Arch Linux AUR** - ✅ Already implemented

---

## Implementation Phases

### Phase 1: GoReleaser Enhancement

Extend `.goreleaser.yaml` to generate `.deb`, `.rpm`, and Scoop packages natively.

**GoReleaser natively supports:**
- ✅ Homebrew (implemented)
- ✅ AUR (implemented)
- 🔲 NFPM for `.deb` and `.rpm`
- 🔲 Scoop manifests

### Phase 2: Debian/Ubuntu Packages

Create `.deb` packages with apt repository hosting.

### Phase 3: RPM Packages

Create `.rpm` packages with Fedora COPR.

### Phase 4: Windows Packages

Publish to Winget, Scoop, and Chocolatey.

---

## Detailed Platform Plans

### 1. Debian/Ubuntu (.deb) Packages

#### Method: NFPM via GoReleaser + apt repository

**GoReleaser NFPM Configuration:**

```yaml
# Add to .goreleaser.yaml
nfpms:
  - id: packages
    package_name: gsh
    vendor: atinylittleshell
    homepage: https://github.com/atinylittleshell/gsh
    maintainer: atinylittleshell <shell@atinylittleshell.me>
    description: A modern, POSIX-compatible, generative shell
    license: GPL-3.0-or-later
    formats:
      - deb
      - rpm
    bindir: /usr/bin
    section: shells
    priority: optional
    rpm:
      group: System Environment/Shells
      compression: xz
    deb:
      lintian_overrides:
        - statically-linked-binary
    contents:
      - src: ./LICENSE
        dst: /usr/share/licenses/gsh/LICENSE
        file_info:
          mode: 0644
      - src: ./README.md
        dst: /usr/share/doc/gsh/README.md
        file_info:
          mode: 0644
```

**Distribution Options:**

1. **GitHub Releases** - Upload `.deb` to releases (simplest, recommended to start)
2. **Launchpad PPA** - Ubuntu-native, auto-builds for multiple Ubuntu versions
3. **Cloudsmith/packagecloud.io** - Hosted apt repository service

**User Installation:**

```bash
# Direct download from GitHub Releases
wget https://github.com/atinylittleshell/gsh/releases/download/v0.26.0/gsh_0.26.0_amd64.deb
sudo dpkg -i gsh_0.26.0_amd64.deb

# Or with PPA (after setup)
sudo add-apt-repository ppa:atinylittleshell/gsh
sudo apt update
sudo apt install gsh
```

---

### 2. Red Hat/Fedora (.rpm) Packages

#### Method: NFPM via GoReleaser + COPR

The same NFPM configuration above generates both `.deb` and `.rpm`.

**Distribution Options:**

1. **GitHub Releases** - Upload `.rpm` directly (simplest, recommended to start)
2. **Fedora COPR** - Community repository, free, well-integrated

**User Installation:**

```bash
# Direct download from GitHub Releases
sudo rpm -i https://github.com/atinylittleshell/gsh/releases/download/v0.26.0/gsh-0.26.0-1.x86_64.rpm

# Or with COPR (after setup)
sudo dnf copr enable atinylittleshell/gsh
sudo dnf install gsh
```

---

### 3. Windows Packages

#### 3a. Scoop (Recommended for developers)

**GoReleaser Scoop Configuration:**

```yaml
# Add to .goreleaser.yaml
scoops:
  - name: gsh
    homepage: https://github.com/atinylittleshell/gsh
    description: A modern, POSIX-compatible, generative shell
    license: GPL-3.0-or-later
    repository:
      owner: atinylittleshell
      name: scoop-bucket
      token: "{{ .Env.GITHUB_TOKEN }}"
```

**Setup Required:**
- Create a `scoop-bucket` repository at `github.com/atinylittleshell/scoop-bucket`

**User Installation:**
```powershell
scoop bucket add gsh https://github.com/atinylittleshell/scoop-bucket
scoop install gsh
```

#### 3b. Winget (Windows Package Manager)

**Manifest File: `packaging/winget/atinylittleshell.gsh.yaml`**

```yaml
PackageIdentifier: atinylittleshell.gsh
PackageVersion: 0.26.0
PackageLocale: en-US
Publisher: atinylittleshell
PackageName: gsh
License: GPL-3.0-or-later
ShortDescription: A modern, POSIX-compatible, generative shell
PackageUrl: https://github.com/atinylittleshell/gsh
Installers:
  - Architecture: x64
    InstallerType: zip
    InstallerUrl: https://github.com/atinylittleshell/gsh/releases/download/v0.26.0/gsh_Windows_x86_64.zip
    InstallerSha256: <sha256>
ManifestType: singleton
ManifestVersion: 1.4.0
```

**Process:** Submit PR to [microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs)

**User Installation:**
```powershell
winget install atinylittleshell.gsh
```

#### 3c. Chocolatey

**File: `packaging/chocolatey/gsh.nuspec`**

```xml
<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://schemas.microsoft.com/packaging/2015/06/nuspec.xsd">
  <metadata>
    <id>gsh</id>
    <version>0.26.0</version>
    <title>gsh - Generative Shell</title>
    <authors>atinylittleshell</authors>
    <owners>atinylittleshell</owners>
    <licenseUrl>https://github.com/atinylittleshell/gsh/blob/main/LICENSE</licenseUrl>
    <projectUrl>https://github.com/atinylittleshell/gsh</projectUrl>
    <requireLicenseAcceptance>false</requireLicenseAcceptance>
    <description>A modern, POSIX-compatible, generative shell with AI capabilities</description>
    <tags>shell cli terminal ai llm</tags>
  </metadata>
  <files>
    <file src="tools\**" target="tools" />
  </files>
</package>
```

**File: `packaging/chocolatey/tools/chocolateyinstall.ps1`**

```powershell
$ErrorActionPreference = 'Stop'
$toolsDir = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$url64 = 'https://github.com/atinylittleshell/gsh/releases/download/v0.26.0/gsh_Windows_x86_64.zip'

$packageArgs = @{
  packageName   = $env:ChocolateyPackageName
  unzipLocation = $toolsDir
  url64bit      = $url64
  checksum64    = '<sha256>'
  checksumType64= 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
```

**User Installation:**
```powershell
choco install gsh
```

---

## CI/CD Integration

### Enhanced GitHub Actions Workflow

**Update `.github/workflows/release-please.yml`:**

```yaml
name: Release

on:
  push:
    branches:
      - main

permissions:
  contents: write
  pull-requests: write

jobs:
  release-please:
    runs-on: ubuntu-latest
    outputs:
      release_created: ${{ steps.release.outputs.release_created }}
      tag_name: ${{ steps.release.outputs.tag_name }}
    steps:
      - uses: googleapis/release-please-action@v4
        id: release
        with:
          release-type: go

  goreleaser:
    needs: release-please
    if: ${{ needs.release-please.outputs.release_created }}
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version: stable

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          distribution: goreleaser
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          AUR_PRIVATE_KEY: ${{ secrets.AUR_PRIVATE_KEY }}

  chocolatey:
    needs: [release-please, goreleaser]
    if: ${{ needs.release-please.outputs.release_created }}
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4

      - name: Download release assets
        run: |
          $version = "${{ needs.release-please.outputs.tag_name }}".TrimStart('v')
          # Update checksums and version in chocolatey files
          # Pack and push to Chocolatey

      - name: Push to Chocolatey
        run: |
          cd packaging/chocolatey
          choco pack
          choco push gsh.*.nupkg --source https://push.chocolatey.org/ --api-key ${{ secrets.CHOCO_API_KEY }}

  winget:
    needs: [release-please, goreleaser]
    if: ${{ needs.release-please.outputs.release_created }}
    runs-on: ubuntu-latest
    steps:
      - uses: vedantmgoyal9/winget-releaser@main
        with:
          identifier: atinylittleshell.gsh
          version: ${{ needs.release-please.outputs.tag_name }}
          token: ${{ secrets.WINGET_TOKEN }}
```

---

## Complete Updated .goreleaser.yaml

```yaml
version: 2

before:
  hooks:
    - go mod tidy

builds:
  - main: ./cmd/gsh
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - windows
      - darwin
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w -X main.BUILD_VERSION={{.Version}}

source:
  enabled: true

archives:
  - format: tar.gz
    name_template: >-
      {{ .ProjectName }}_
      {{- title .Os }}_
      {{- if eq .Arch "amd64" }}x86_64
      {{- else if eq .Arch "386" }}i386
      {{- else }}{{ .Arch }}{{ end }}
      {{- if .Arm }}v{{ .Arm }}{{ end }}
    format_overrides:
      - goos: windows
        format: zip

# NFPM for .deb and .rpm packages
nfpms:
  - id: packages
    package_name: gsh
    vendor: atinylittleshell
    homepage: https://github.com/atinylittleshell/gsh
    maintainer: atinylittleshell <shell@atinylittleshell.me>
    description: A modern, POSIX-compatible, generative shell
    license: GPL-3.0-or-later
    formats:
      - deb
      - rpm
    bindir: /usr/bin
    section: shells
    priority: optional
    rpm:
      group: System Environment/Shells
      compression: xz
    deb:
      lintian_overrides:
        - statically-linked-binary
    contents:
      - src: ./LICENSE
        dst: /usr/share/licenses/gsh/LICENSE
        file_info:
          mode: 0644
      - src: ./README.md
        dst: /usr/share/doc/gsh/README.md
        file_info:
          mode: 0644

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"

# Homebrew
brews:
  - name: gsh
    homepage: https://github.com/atinylittleshell/gsh
    description: A modern, POSIX-compatible, generative shell
    license: GPL-3.0-or-later
    directory: Formula
    commit_author:
      name: atinylittleshell
      email: shell@atinylittleshell.me
    repository:
      owner: atinylittleshell
      name: gsh
      token: "{{ .Env.GITHUB_TOKEN }}"
    install: |
      bin.install "gsh"
    test: |
      system "#{bin}/gsh", "--version"

# Scoop (Windows)
scoops:
  - name: gsh
    homepage: https://github.com/atinylittleshell/gsh
    description: A modern, POSIX-compatible, generative shell
    license: GPL-3.0-or-later
    repository:
      owner: atinylittleshell
      name: scoop-bucket
      token: "{{ .Env.GITHUB_TOKEN }}"

# AUR binary package
aurs:
  - name: gsh-bin
    homepage: https://github.com/atinylittleshell/gsh
    description: A modern, POSIX-compatible, generative shell
    license: GPL-3.0-or-later
    maintainers:
      - "atinylittleshell <shell@atinylittleshell.me>"
    private_key: "{{ .Env.AUR_PRIVATE_KEY }}"
    git_url: "ssh://aur@aur.archlinux.org/gsh-bin.git"
    commit_author:
      name: atinylittleshell
      email: shell@atinylittleshell.me
    package: |-
      install -Dm755 "./gsh" "${pkgdir}/usr/bin/gsh"
      install -Dm644 "./LICENSE" "${pkgdir}/usr/share/licenses/gsh/LICENSE"

# AUR source package
aur_sources:
  - name: gsh
    homepage: https://github.com/atinylittleshell/gsh
    description: A modern, POSIX-compatible, generative shell
    license: GPL-3.0-or-later
    maintainers:
      - "atinylittleshell <shell@atinylittleshell.me>"
      - "Vitalii Kuzhdin <vitaliikuzhdin@gmail.com>"
    private_key: "{{ .Env.AUR_PRIVATE_KEY }}"
    git_url: "ssh://aur@aur.archlinux.org/gsh.git"
    commit_author:
      name: atinylittleshell
      email: shell@atinylittleshell.me
    prepare: |-
      cd "${srcdir}/${_pkgsrc}"
      go mod download
    build: |-
      cd "${srcdir}/${_pkgsrc}"
      export CGO_ENABLED=0
      export GOFLAGS="-trimpath -mod=readonly -modcacherw"
      go build -ldflags="-X main.BUILD_VERSION=${pkgver}" -o "./bin/${pkgname}" "./cmd/${pkgname}"
    package: |-
      cd "${srcdir}/${_pkgsrc}"
      install -Dsm755 "./bin/${pkgname}" "${pkgdir}/usr/bin/${pkgname}"
      install -Dm644 "./LICENSE" "${pkgdir}/usr/share/licenses/${pkgname}/LICENSE"
      install -Dm644 "./README.md" "${pkgdir}/usr/share/doc/${pkgname}/README.md"
      install -Dm644 "./ROADMAP.md" "${pkgdir}/usr/share/doc/${pkgname}/ROADMAP.md"
      install -Dm644 "./CHANGELOG.md" "${pkgdir}/usr/share/doc/${pkgname}/CHANGELOG.md"
```

---

## Directory Structure

After implementation:

```
gsh/
├── .goreleaser.yaml           # Enhanced with NFPM + Scoop
├── packaging/
│   ├── chocolatey/
│   │   ├── gsh.nuspec
│   │   └── tools/
│   │       ├── chocolateyinstall.ps1
│   │       └── chocolateyuninstall.ps1
│   └── winget/
│       └── atinylittleshell.gsh.yaml
├── Formula/
│   └── gsh.rb                 # Auto-generated by GoReleaser
└── flake.nix                  # Existing Nix flake
```

---

## Implementation Checklist

### Phase 1: GoReleaser Enhancement
- [ ] Add NFPM configuration for .deb and .rpm to `.goreleaser.yaml`
- [ ] Add Scoop configuration to `.goreleaser.yaml`
- [ ] Create `scoop-bucket` repository on GitHub
- [ ] Test local builds with `goreleaser release --snapshot --clean`

### Phase 2: Debian/Ubuntu
- [ ] Verify .deb packages are uploaded to GitHub Releases
- [ ] Test installation on Ubuntu: `sudo dpkg -i gsh_*.deb`
- [ ] (Optional) Set up Launchpad PPA for apt repository
- [ ] Update README with installation instructions

### Phase 3: RPM Packages
- [ ] Verify .rpm packages are uploaded to GitHub Releases
- [ ] Test installation on Fedora: `sudo rpm -i gsh-*.rpm`
- [ ] (Optional) Set up COPR repository
- [ ] Update README with installation instructions

### Phase 4: Windows
- [ ] Create `packaging/chocolatey/` directory with package files
- [ ] Register on chocolatey.org and get API key
- [ ] Add `CHOCO_API_KEY` secret to GitHub
- [ ] Submit initial package to winget-pkgs
- [ ] Add `WINGET_TOKEN` secret to GitHub (PAT with public_repo scope)
- [ ] Test all three installation methods on Windows

### Phase 5: CI/CD
- [ ] Update `.github/workflows/release-please.yml` with new jobs
- [ ] Add required secrets to GitHub repository
- [ ] Test full release pipeline

---

## User Installation Summary

After implementation, users can install gsh using:

```bash
# macOS
brew install atinylittleshell/gsh/gsh

# Ubuntu/Debian
wget https://github.com/atinylittleshell/gsh/releases/latest/download/gsh_amd64.deb
sudo dpkg -i gsh_amd64.deb

# Fedora/RHEL/CentOS
sudo rpm -i https://github.com/atinylittleshell/gsh/releases/latest/download/gsh.x86_64.rpm

# Arch Linux
yay -S gsh          # Source build
yay -S gsh-bin      # Pre-built binary

# NixOS / Nix
nix run github:atinylittleshell/gsh

# Windows (Scoop)
scoop bucket add gsh https://github.com/atinylittleshell/scoop-bucket
scoop install gsh

# Windows (Winget)
winget install atinylittleshell.gsh

# Windows (Chocolatey)
choco install gsh

# Direct download (any platform)
# https://github.com/atinylittleshell/gsh/releases
```

---

## Required Secrets

| Secret | Purpose |
|--------|---------|
| `GITHUB_TOKEN` | Auto-provided, used for releases |
| `AUR_PRIVATE_KEY` | Already configured for AUR publishing |
| `CHOCO_API_KEY` | Chocolatey.org API key |
| `WINGET_TOKEN` | GitHub PAT for winget-pkgs PRs |

$ErrorActionPreference = 'Stop'

$packageName = $env:ChocolateyPackageName
$toolsDir = "$(Split-Path -Parent $MyInvocation.MyCommand.Definition)"
$url64 = 'https://github.com/robottwo/bishop/releases/download/v0.26.0/bish_Windows_x86_64.zip'

$packageArgs = @{
  packageName   = $packageName
  unzipLocation = $toolsDir
  url64bit      = $url64
  checksum64    = 'PLACEHOLDER_CHECKSUM'
  checksumType64= 'sha256'
}

Install-ChocolateyZipPackage @packageArgs

# Create shim for bish.exe
$bishPath = Join-Path $toolsDir 'bish.exe'
if (Test-Path $bishPath) {
  Write-Host "bish installed successfully to $bishPath"
}

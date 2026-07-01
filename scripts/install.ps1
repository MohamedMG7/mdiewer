$ErrorActionPreference = "Stop"

$Repo = if ($env:MDIEWER_REPO) { $env:MDIEWER_REPO } else { "MohamedMG7/mdiewer" }
$Version = if ($env:MDIEWER_VERSION) { $env:MDIEWER_VERSION } else { "latest" }
$InstallDir = if ($env:MDIEWER_INSTALL_DIR) { $env:MDIEWER_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "Programs\mdiewer" }

$arch = switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant()) {
    "x64" { "amd64" }
    "arm64" { "arm64" }
    default { throw "Unsupported Windows architecture: $([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture)" }
}

$asset = "mdiewer-windows-$arch.zip"
if ($Version -eq "latest") {
    $url = "https://github.com/$Repo/releases/latest/download/$asset"
} else {
    $url = "https://github.com/$Repo/releases/download/$Version/$asset"
}

$tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("mdiewer-install-" + [System.Guid]::NewGuid().ToString("N"))
$zip = Join-Path $tmp $asset

New-Item -ItemType Directory -Path $tmp | Out-Null
New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null

try {
    Write-Host "Downloading $url"
    Invoke-WebRequest -Uri $url -OutFile $zip
    Expand-Archive -Path $zip -DestinationPath $tmp -Force

    $exe = Get-ChildItem -Path $tmp -Filter "mdiewer.exe" -Recurse | Select-Object -First 1
    if (-not $exe) {
        throw "mdiewer.exe was not found in $asset"
    }

    Copy-Item -Path $exe.FullName -Destination (Join-Path $InstallDir "mdiewer.exe") -Force

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $paths = @()
    if (-not [string]::IsNullOrWhiteSpace($userPath)) {
        $paths = $userPath.Split(";") | Where-Object { -not [string]::IsNullOrWhiteSpace($_) }
    }

    $alreadyOnPath = $false
    foreach ($path in $paths) {
        if ([string]::Equals($path.Trim(), $InstallDir, [StringComparison]::OrdinalIgnoreCase)) {
            $alreadyOnPath = $true
        }
    }

    if (-not $alreadyOnPath) {
        $newPath = ($paths + $InstallDir) -join ";"
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        $env:Path = $env:Path + ";" + $InstallDir
        Write-Host "Added $InstallDir to your user PATH."
    }

    Write-Host "Installed mdiewer to $InstallDir"
    Write-Host "Open a new terminal, then run: mdiewer --help"
} finally {
    Remove-Item -Path $tmp -Recurse -Force -ErrorAction SilentlyContinue
}

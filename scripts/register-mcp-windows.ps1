<#
.SYNOPSIS
Registra IA_Orquestador MCP en %USERPROFILE%\.mcp.json (VS Code y Visual Studio).

.PARAMETER Url
URL del servidor IA_Orquestador. Ejemplo: http://<CT_IP>:7439/mcp

.PARAMETER Token
API key del servidor (generada en el primer arranque). Ejemplo: mcp_xxx

.EXAMPLE
.\register-mcp-windows.ps1 -Url "http://<CT_IP>:7439/mcp" -Token "mcp_xxx"
.\register-mcp-windows.ps1
#>

param(
    [string]$Url,
    [string]$Token
)

$mcpFile = Join-Path $env:USERPROFILE ".mcp.json"

if (-not $Url) {
    $Url = Read-Host "URL del servidor MCP (ej: http://<CT_IP>:7439/mcp)"
}
if (-not $Token) {
    $Token = Read-Host "API key (ej: mcp_xxx)"
}

# Leer config existente o crear nueva
if (Test-Path $mcpFile) {
    $config = Get-Content $mcpFile -Raw | ConvertFrom-Json
} else {
    $config = [PSCustomObject]@{
        inputs  = @()
        servers = [PSCustomObject]@{}
    }
}

if (-not $config.servers) {
    $config | Add-Member -MemberType NoteProperty -Name "servers" -Value ([PSCustomObject]@{})
}

# Añadir o reemplazar entrada ia-orquestador
$serverEntry = [PSCustomObject]@{
    type    = "http"
    url     = $Url
    headers = [PSCustomObject]@{
        Authorization = "Bearer $Token"
    }
}

if ($config.servers.PSObject.Properties["ia-orquestador"]) {
    $config.servers."ia-orquestador" = $serverEntry
} else {
    $config.servers | Add-Member -MemberType NoteProperty -Name "ia-orquestador" -Value $serverEntry
}

$config | ConvertTo-Json -Depth 10 | Set-Content $mcpFile -Encoding UTF8
Write-Host "OK: ia-orquestador registrado en $mcpFile"
Write-Host "Reinicia VS Code o Visual Studio para aplicar los cambios."
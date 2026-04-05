# Registrar IA_Orquestador MCP en Windows

## 1 — Editar `%USERPROFILE%\.mcp.json`

Tanto **VS Code** como **Visual Studio 2022/2026** leen este mismo archivo:

```
C:\Users\<tu-usuario>\.mcp.json
```

Añade la entrada `ia-orquestador`:

```json
{
  "inputs": [],
  "servers": {
    "ia-orquestador": {
      "type": "http",
      "url": "http://<HOST>:<PUERTO>/mcp",
      "headers": {
        "Authorization": "Bearer mcp_<TU_API_KEY>"
      }
    }
  }
}
```

## 2 — Reiniciar el cliente

Cierra y abre VS Code o Visual Studio para que detecte el nuevo servidor.

## 3 — Verificar

```powershell
$body = '{"jsonrpc":"2.0","id":1,"method":"ping","params":{}}'
Invoke-RestMethod -Uri "http://<HOST>:<PUERTO>/mcp" -Method POST `
  -ContentType "application/json" -Body $body
# → result: @{status=pong}
```

---

> **Nota:** para registrar o listar skills usa la API REST:
>
> ```powershell
> # Listar skills
> Invoke-RestMethod -Uri "http://<HOST>:<PUERTO>/api/v1/skills" `
>   -Headers @{"X-Api-Key" = "mcp_<TU_API_KEY>"}
> ```

Ver [README.md](../README.md) para documentación completa.
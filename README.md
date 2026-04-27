# IA_Orquestador

> Servidor de orquestación MCP (Model Context Protocol) escrito en Go. Registra skills/tools dinámicamente, gestiona sesiones de agentes IA y se integra con **IA_Recuerdo** para memoria persistente.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org)
[![MCP](https://img.shields.io/badge/MCP-2024--11--05-blueviolet)](https://spec.modelcontextprotocol.io/)

---

## ¿Qué hace?

**IA_Orquestador** expone un servidor JSON-RPC 2.0 compatible con MCP que actúa como capa de infraestructura IA:

- Registra y ejecuta **skills** (herramientas de IA) dinámicamente
- Soporta transporte **STDIO** (agentes locales) y **HTTP + SSE + WebSocket** (agentes remotos)
- Persiste estado en **PostgreSQL**
- Se integra con **IA_Recuerdo** para memoria persistente entre sesiones
- Expone **API REST de administración** (`/api/v1/skills`)
- Autenticación por **API key** con bootstrap automático en primer arranque

---

## Arquitectura

```
Cliente IA (Claude / Cursor / VS Code Copilot)
         │
         ├── STDIO  (agentes locales, IDE)
         └── HTTP + SSE + WebSocket  (agentes remotos)
                    │
         ┌──────────▼───────────┐
         │    IA_Orquestador    │
         │                      │
         │  JSON-RPC dispatcher │◄─── POST /mcp/jsonrpc
         │  Skill Registry      │──►  Proceso externo (bash/sh)
         │  Session Manager     │
         │  Auth (API Key)      │◄─── X-Api-Key header
          │     PostgreSQL       │
         │  Metrics /metrics    │──►  Prometheus
         └──────────┬───────────┘
                    │ HTTP
         ┌──────────▼───────────┐
         │     IA_Recuerdo     │  :7438  (memoria persistente)
         └──────────────────────┘
```

---

## Estructura del código

```
code/ia-orquestador/
├── cmd/orchestrator/       # main.go — flags, bootstrap, wiring
├── internal/
│   ├── transport/          # STDIO, HTTP+SSE, WebSocket
│   ├── jsonrpc/            # Dispatcher JSON-RPC 2.0
│   ├── skills/             # Registro dinámico de skills
│   ├── db/                 # PostgreSQL
│   ├── auth/               # API key (SHA-256, bootstrap automático)
│   ├── executor/           # Ejecución de skills externos
│   ├── admin/              # API REST /api/v1/skills
│   └── metrics/            # Endpoint /metrics (Prometheus)
├── pkg/
│   ├── types/              # Tipos MCP
│   └── errors/             # Códigos de error MCP
├── skills/
│   ├── dotnet/             # 18 skills .NET (Blazor, MAUI, Clean Arch, etc.)
│   ├── sdd/                # 9 skills SDD (Spec-Driven Development)
│   └── echo-skill/         # Skill de ejemplo
├── deploy/
│   ├── systemd/            # Unit file para Linux / Proxmox CT
│   └── kubernetes/         # Manifiestos K8s
├── configs/                # config.example.yaml
├── migrations/             # Migraciones de DB
└── scripts/                # register-skill.sh, test-e2e.sh
```

---

## Build

```bash
cd code/ia-orquestador

make build          # Go, desarrollo
make build-linux    # Linux amd64 (cross-compile)
make build-postgres # Linux amd64, PostgreSQL
```

---

## Inicio rápido (local)

```bash
cd code/ia-orquestador

# Modo STDIO — para agentes locales (Claude Code, VS Code, etc.)
make run

# Modo HTTP — para agentes remotos o múltiples clientes
make run-http
# → http://localhost:8080
```

En el **primer arranque** sin API keys registradas, el servidor genera una automáticamente:

```
FIRST RUN — Bootstrap API key (save, shown once only):
mcp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

Guardala — no se muestra de nuevo. Usarla como `X-Api-Key` en las requests.

Para crear claves adicionales:

```bash
./bin/orchestrator -create-token "mi-agente"
```

---

## Configuración

Todos los parámetros se pasan como flags:

| Flag | Default | Descripción |
|------|---------|-------------|
| `-transport` | `stdio` | `stdio` o `http` |
| `-http-addr` | `:8080` | Dirección del servidor HTTP |
| `-db` | `postgres://...` | DSN PostgreSQL |
| `-db-driver` | `postgres` | Solo `postgres` |
| `-db-dsn` | `` | DSN PostgreSQL |
| `-memory-url` | `http://127.0.0.1:7438` | URL de IA_Recuerdo |
| `-project` | `ia-orquestador` | Nombre de proyecto en IA_Recuerdo |
| `-create-token` | `` | Crear API key con nombre dado y salir |
| `-otel-exporter` | `none` | OTel traces: `none` \| `stdout` \| `otlp` \| `both` |
| `-otel-endpoint` | `localhost:4318` | Endpoint OTLP HTTP (para `otlp` o `both`) |
| `-skill-reload-interval` | `0` | Hot-reload de skills; `0` = deshabilitado (ej. `30s`) |

---

## API (JSON-RPC 2.0)

Endpoint: `POST /mcp/jsonrpc`

### Inicializar sesión

```bash
curl -X POST http://localhost:8080/mcp/jsonrpc \
  -H "Content-Type: application/json" \
  -H "X-Api-Key: mcp_xxx" \
  -d '{"jsonrpc":"2.0","id":1,"method":"mcp.initialize","params":{"clientId":"cli","protocolVersion":"2024-11-05","clientCapabilities":{}}}'
```

### Listar skills

```bash
curl -X POST http://localhost:8080/mcp/jsonrpc \
  -H "Content-Type: application/json" \
  -H "X-Api-Key: mcp_xxx" \
  -d '{"jsonrpc":"2.0","id":2,"method":"mcp.tools.list","params":{}}'
```

### Ejecutar un skill

```bash
curl -X POST http://localhost:8080/mcp/jsonrpc \
  -H "Content-Type: application/json" \
  -H "X-Api-Key: mcp_xxx" \
  -d '{"jsonrpc":"2.0","id":3,"method":"mcp.tools.call","params":{"toolId":"echo-skill","sessionId":"demo","input":{"text":"hola"}}}'
```

### Métodos disponibles

| Método | Descripción |
|--------|-------------|
| `mcp.initialize` | Iniciar sesión, obtener `sessionId` |
| `mcp.tools.list` | Listar skills registrados |
| `mcp.tools.call` | Ejecutar un skill (sync / SSE stream) |
| `mcp.tools.status` | Estado de ejecución por `requestId` |
| `mcp.tools.cancel` | Cancelar ejecución |

### Endpoints HTTP

| Endpoint | Descripción |
|----------|-------------|
| `POST /mcp/jsonrpc` | JSON-RPC 2.0 |
| `GET /mcp/stream?session_id=xxx` | SSE stream de eventos |
| `GET /mcp/ws` | WebSocket full-duplex |
| `GET /healthz` | Health check |
| `GET /metrics` | Métricas Prometheus |
| `GET /api/v1/skills` | Lista skills (REST admin) |
| `POST /api/v1/skills` | Registrar skill |

---

## Registrar el MCP en clientes IA

El servidor **IA_Orquestador** puede ser consumido por cualquier IDE o agente IA que soporte MCP. A continuación se documenta cómo registrarlo en **VS Code** y **Visual Studio 2022 / 2026** con GitHub Copilot.

### Obtener/Crear una API Key

El servidor genera automáticamente una **API key de bootstrap** en el primer arranque:

```bash
# Arrancar el servidor
make run-http

# En los logs verás:
# FIRST RUN — Bootstrap API key (save, shown once only):
# mcp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

Para crear claves adicionales para otros clientes:

```bash
./bin/orchestrator -create-token "mi-cliente"
```

> ⚠️ **Importante**: La clave se muestra una sola vez. Guárdala en un lugar seguro.

---

### VS Code y Visual Studio — archivo de configuración único

Tanto **VS Code** como **Visual Studio 2022/2026** con GitHub Copilot leen el archivo:

```
%USERPROFILE%\.mcp.json
```

Es decir: `C:\Users\<tu-usuario>\.mcp.json`

Añade la entrada `ia-orquestador` junto a cualquier otro servidor que ya tengas:

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

Reinicia el cliente (VS Code o Visual Studio) para que cargue la nueva configuración.

---

### Verificar la conexión

```bash
curl -s -X POST http://<HOST>:<PUERTO>/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"ping","params":{}}'

# Respuesta esperada:
# {"jsonrpc":"2.0","id":1,"result":{"status":"pong"}}
```

> **Nota sobre autenticación:** El endpoint `/mcp` no requiere autenticación (el middleware de auth solo protege las rutas `/api/v1/*`). El header `Authorization: Bearer` es aceptado pero ignorado en el endpoint MCP.

---

## Registrar skills

```bash
# Skill individual
scripts/register-skill.sh skills/echo-skill/skill.sh "Echo Skill" "Devuelve el input"

# Registro masivo
scripts/bulk-register-skills.sh skills/dotnet/
```

---

## Skills incluidos

### .NET (18 skills)

`blazor-server` · `blazor-wasm` · `csharp-expert` · `dotnet-architecture` · `minimal-api` · `microservices-dotnet` · `rest-webapi` · `mudblazor` · `maui-expert` · `mvvm-patterns` · `mvc-dotnet` · `onion-architecture` · `solid-principles` · `wpf-expert` · `aspire-dotnet` · `aspx-legacy` · `soap-wcf`

### SDD — Spec-Driven Development (9 skills)

`sdd-init` · `sdd-explore` · `sdd-design` · `sdd-spec` · `sdd-propose` · `sdd-tasks` · `sdd-apply` · `sdd-verify` · `sdd-archive`

---

## Despliegue en Proxmox (systemd)

```bash
# 1. Compilar para Linux
cd code/ia-orquestador && make build-linux

# 2. Copiar al CT
scp bin/orchestrator root@<ct-ip>:/opt/ia-orquestador/bin/

# 3. Crear usuario del servicio
useradd -r -s /sbin/nologin orquestador
mkdir -p /var/lib/ia-orquestador
chown orquestador: /var/lib/ia-orquestador

# 4. Instalar unit file
cp deploy/systemd/ia-orquestador.service /etc/systemd/system/

# 5. Editar la URL de IA_Recuerdo
# ExecStart=... -memory-url=http://<ia-recuerdo-host>:7438

# 6. Arrancar
systemctl daemon-reload
systemctl enable --now ia-orquestador
systemctl status ia-orquestador
```

El servicio escucha en `:7438` por defecto (configurable en el unit file).

---

## Tests

```bash
cd code/ia-orquestador
make test       # Tests unitarios
make test-e2e   # Tests end-to-end
make vet        # go vet
```

---

## Estado

| Componente | Estado |
|------------|--------|
| JSON-RPC 2.0 dispatcher | ✅ |
| STDIO | ✅ |
| HTTP + SSE | ✅ |
| WebSocket | ✅ |
| Skill registry + ejecución | ✅ |
| IA_Recuerdo integration | ✅ |
| PostgreSQL | ✅ |
| PostgreSQL | ✅ build tag `postgres` |
| API REST admin | ✅ |
| Auth API key + Bearer token | ✅ |
| Métricas Prometheus `/metrics` | ✅ |
| OpenTelemetry traces (stdout/OTLP) | ✅ |
| Skill hot-reload (polling DB) | ✅ |

---

## Agradecimientos

Este proyecto usa **IA_Recuerdo** como memoria persistente y se inspira en los patrones SDD de **Alan Buscaglia** y la comunidad **Gentleman Programming**.

| Proyecto | Descripción |
|----------|-------------|
| [Gentle AI](https://github.com/Gentleman-Programming/gentle-ai) ⭐ 1.6k | Stack IA completo para cualquier agente |
| [Gentleman Skills](https://github.com/Gentleman-Programming/Gentleman-Skills) | Skills curados para Claude Code, OpenCode, VS Code |
| [Agent Teams Lite](https://github.com/Gentleman-Programming/agent-teams-lite) | Orquestación SDD, 9 sub-agentes, zero deps |

🎥 [YouTube @GentlemanProgramming](https://www.youtube.com/@GentlemanProgramming) · 🌐 [alan-buscaglia.vercel.app](https://alan-buscaglia.vercel.app/home) · 🔗 [doras.to/gentleman-programming](https://doras.to/gentleman-programming)

> *"Concepts over code. Foundations over frameworks."* — Alan Buscaglia

---

## Licencia

[MIT](LICENSE) © 2026 mdesantis1984

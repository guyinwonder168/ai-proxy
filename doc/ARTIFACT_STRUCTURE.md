# AI Proxy Release Artifact Structure

## Required Artifacts for VPS Deployment

### 1. ai-proxy-release.tar.gz Structure
```
ai-proxy-release.tar.gz
├── ai-proxy                     # Pre-compiled binary (executable)
├── Dockerfile                   # Production Dockerfile with dynamic EXPOSE
├── .env.example                 # Environment template
├── provider-config.yaml         # Provider configuration template
├── README.md                    # Project documentation
├── LICENSE                      # License file
└── INSTALL.md                   # Installation instructions
```

### 2. installer.sh Requirements

The installer script must be delivered as a separate file alongside the tar.gz and implement:

#### a. Interactive Port Validation
- Prompt user for host port (1-6535)
- Validate numeric input range
- Extract ai-proxy-release.tar.gz to `/tmp/ai-proxy` (isolated context)
- Create `/srv/ai-proxy` with strict ownership/permissions (root:root, 755)
- Populate `/srv/ai-proxy` with `.env` and `provider-config.yaml` from extracted artifacts
- Atomically update `PORT` in `/srv/ai-proxy/.env`
- Dynamically modify `EXPOSE` directive in `/tmp/ai-proxy/Dockerfile`

#### b. Docker Build Context Isolation
- Build `ai-proxy:latest` exclusively from `/tmp/ai-proxy` contents
- Prohibit local path references in build context
- Implement build cache optimization
- Use single-stage build for pre-compiled binary deployment

#### c. Production Container Deployment
- Automatic restart policy (unless-stopped)
- Health check with liveness probe
- Host:container port mapping from user input
- Immutable config volume: `/srv/ai-proxy → /app/config:ro`
- Resource constraints (memory/CPU limits)
- Security hardening (non-root, no-new-privileges, etc.)

### 3. Security & Error Handling
- All filesystem operations use absolute paths
- Atomic operations with rollback capability
- Step validation with exit-on-error (set -e)
- Cleanup trap removing /tmp/ai-proxy on exit
- Pre-flight dependency checks (Docker, tar)
- Idempotent execution support

### 4. Dockerfile Requirements
- Build context strictly limited to `/tmp/ai-proxy` contents
- Runtime configuration exclusively sourced from `/app/config`
- `EXPOSE` directive dynamically synchronized with runtime port
- Non-root execution context (ai-proxy user, UID 101)
- Single-stage build optimization
- Pre-compiled binary support (no Go compilation needed)
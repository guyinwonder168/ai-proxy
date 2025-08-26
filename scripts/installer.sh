#!/usr/bin/env bash
# AI Proxy Production VPS Installer (Docker/Podman)
# - Extracts ai-proxy-release.tar.gz
# - Locates build context (folder containing Dockerfile)
# - Interactively asks for port (or reads APP_PORT), validates strictly
# - Patches Dockerfile (LABEL/EXPOSE) + runtime .env to chosen port
# - SELinux-safe read-only config bind (adds :Z automatically)
# - Runs container as non-root, overrides entrypoint to ./ai-proxy (skips UID remap)
# - Adds container-level healthcheck (--health-cmd, --health-*)
# - Waits for HTTP health from host, prints logs on failure, keeps container on success

set -Eeuo pipefail
IFS=$'\n\t'

########################################
# User-tunable knobs (env overrides OK)
########################################
RELEASE_ARCHIVE="${RELEASE_ARCHIVE:-ai-proxy-release.tar.gz}"

# Root-owned config dir (mounted read-only)
CONFIG_DIR="${CONFIG_DIR:-/srv/ai-proxy}"

# Container identity
CONTAINER_NAME="${CONTAINER_NAME:-ai-proxy}"
IMAGE_NAME_LOCAL="${IMAGE_NAME_LOCAL:-ai-proxy:latest}"

# App port & health probe (host + container)
APP_PORT="${APP_PORT:-}"                       # if empty, we’ll prompt
HEALTH_PATH="${HEALTH_PATH:-/ping}"
HEALTH_TIMEOUT_SECS="${HEALTH_TIMEOUT_SECS:-45}"
HEALTH_SUCCESS_STREAK="${HEALTH_SUCCESS_STREAK:-2}"

# Container-level healthcheck (inside the container)
# Choose tool used inside the IMAGE: wget (default) or curl
HEALTHCHECK_TOOL="${HEALTHCHECK_TOOL:-wget}"   # set to "curl" if your image lacks wget
# You can fully override the command if you want. If set, this exact string is used.
# Otherwise we auto-compose based on HEALTHCHECK_TOOL and the resolved URL.
HEALTHCHECK_CMD="${HEALTHCHECK_CMD:-}"

HEALTHCHECK_INTERVAL="${HEALTHCHECK_INTERVAL:-10s}"
HEALTHCHECK_TIMEOUT="${HEALTHCHECK_TIMEOUT:-3s}"
HEALTHCHECK_START_PERIOD="${HEALTHCHECK_START_PERIOD:-5s}"
HEALTHCHECK_RETRIES="${HEALTHCHECK_RETRIES:-3}"

# Logging behavior
LOGS_TAIL_LINES="${LOGS_TAIL_LINES:-200}"
SHOW_LOGS_ON_SUCCESS="${SHOW_LOGS_ON_SUCCESS:-0}"  # 1 to show logs even when healthy
TEARDOWN_ON_SUCCESS="${TEARDOWN_ON_SUCCESS:-0}"    # 1 to stop+rm on success

# Non-root user inside container
APP_UID="${APP_UID:-1001}"
APP_GID="${APP_GID:-1001}"

# Writable volumes (if app uses them)
VAR_VOLUME="${VAR_VOLUME:-ai-proxy-var}"
LOG_VOLUME="${LOG_VOLUME:-ai-proxy-logs}"

########################################
# Internals
########################################
TMP_ROOT=""
BUILD_CTX=""
CONTAINER_RUNTIME=""
CONFIG_MOUNT_SUFFIX=":ro"   # becomes :ro,Z on SELinux systems

log_info()    { echo -e "[INFO] $*"; }
log_warn()    { echo -e "[WARNING] $*"; }
log_error()   { echo -e "[ERROR] $*" >&2; }
die()         { local c=${2:-1}; log_error "$1"; exit "$c"; }

cleanup() {
  # Clean up temporary files and directories
  if [[ -n "${TMP_ROOT:-}" && -d "${TMP_ROOT}" ]]; then
    rm -rf "${TMP_ROOT}"
    log_info "Cleaned up temporary directory: ${TMP_ROOT}"
  fi
}

detect_runtime() {
  if command -v podman >/dev/null 2>&1; then CONTAINER_RUNTIME="podman"
  elif command -v docker >/dev/null 2>&1; then CONTAINER_RUNTIME="docker"
  else die "Neither podman nor docker is installed."
  fi
  log_info "Using container runtime: ${CONTAINER_RUNTIME}"
}

detect_selinux_suffix() {
  CONFIG_MOUNT_SUFFIX=":ro"
  if command -v getenforce >/dev/null 2>&1; then
    local mode
    mode="$(getenforce 2>/dev/null || true)"
    if [[ "${mode}" == "Enforcing" || "${mode}" == "Permissive" ]]; then
      CONFIG_MOUNT_SUFFIX=":ro,Z"
    fi
  fi
}

# shellcheck disable=SC2329  # used throughout; some analyzers miss indirect usages
exists_container(){ "${CONTAINER_RUNTIME}" ps -a --format '{{.Names}}' 2>/dev/null | grep -qx "${CONTAINER_NAME}"; }
# shellcheck disable=SC2329  # used throughout; some analyzers miss indirect usages
is_running(){ "${CONTAINER_RUNTIME}" ps --format '{{.Names}}' 2>/dev/null | grep -qx "${CONTAINER_NAME}"; }

teardown_container(){
  if exists_container; then
    log_warn "Stopping container: ${CONTAINER_NAME}"
    "${CONTAINER_RUNTIME}" stop "${CONTAINER_NAME}" >/dev/null 2>&1 || true
    log_warn "Removing container: ${CONTAINER_NAME}"
    "${CONTAINER_RUNTIME}" rm "${CONTAINER_NAME}"   >/dev/null 2>&1 || true
  fi
}

trap cleanup EXIT

ensure_tools(){
  command -v tar   >/dev/null 2>&1 || die "tar not found"
  if ! command -v curl >/dev/null 2>&1 && ! command -v wget >/dev/null 2>&1; then
    die "Neither curl nor wget found (needed for health probe)."
  fi
  command -v sed   >/dev/null 2>&1 || die "sed not found"
}

extract_release(){
  [[ -f "${RELEASE_ARCHIVE}" ]] || die "Release archive not found: ${RELEASE_ARCHIVE}"
  TMP_ROOT="$(mktemp -d /tmp/ai-proxy.XXXXXX)"
  log_info "Extracting ${RELEASE_ARCHIVE} -> ${TMP_ROOT}"
  tar -xzf "${RELEASE_ARCHIVE}" -C "${TMP_ROOT}"

  # Locate build context with Dockerfile/Containerfile
  if [[ -f "${TMP_ROOT}/Dockerfile" || -f "${TMP_ROOT}/Containerfile" ]]; then
    BUILD_CTX="${TMP_ROOT}"
  else
    mapfile -t dockerfiles < <(find "${TMP_ROOT}" -maxdepth 3 -type f \( -name Dockerfile -o -name Containerfile \) 2>/dev/null || true)
    if [[ ${#dockerfiles[@]} -eq 1 ]]; then
      BUILD_CTX="$(dirname "${dockerfiles[0]}")"
    else
      mapfile -t top_dirs < <(find "${TMP_ROOT}" -mindepth 1 -maxdepth 1 -type d 2>/dev/null || true)
      if [[ ${#top_dirs[@]} -eq 1 && -f "${top_dirs[0]}/Dockerfile" ]]; then
        BUILD_CTX="${top_dirs[0]}"
      else
        log_error "Could not uniquely locate Dockerfile."
        (cd "${TMP_ROOT}" && find . -maxdepth 2 -print)
        die "No Containerfile or Dockerfile found in the extracted archive."
      fi
    fi
  fi
  log_info "Build context resolved to: ${BUILD_CTX}"

  # Ensure executables
  [[ -f "${BUILD_CTX}/ai-proxy" ]] && chmod +x "${BUILD_CTX}/ai-proxy" || true
}

prepare_config_dir(){
  log_info "Ensuring config directory exists: ${CONFIG_DIR}"
  sudo mkdir -p "${CONFIG_DIR}"
  sudo chown root:root "${CONFIG_DIR}"
  sudo chmod 0755 "${CONFIG_DIR}"

  # Copy defaults once if host lacks them
  if [[ -f "${BUILD_CTX}/provider-config.yaml" && ! -f "${CONFIG_DIR}/provider-config.yaml" ]]; then
    sudo cp -n "${BUILD_CTX}/provider-config.yaml" "${CONFIG_DIR}/"
    log_info "Installed default provider-config.yaml to ${CONFIG_DIR}"
  fi
  if [[ -f "${BUILD_CTX}/.env" && ! -f "${CONFIG_DIR}/.env" ]]; then
    sudo cp -n "${BUILD_CTX}/.env" "${CONFIG_DIR}/"
    log_info "Installed default .env to ${CONFIG_DIR}"
  fi

  # Ensure readable perms for non-root in container
  if [[ -f "${CONFIG_DIR}/.env" ]]; then
    sudo chmod 0644 "${CONFIG_DIR}/.env"
  fi
  if [[ -f "${CONFIG_DIR}/provider-config.yaml" ]]; then
    sudo chmod 0644 "${CONFIG_DIR}/provider-config.yaml"
  fi
  sudo chmod 0755 "${CONFIG_DIR}"

  log_info "Config will be mounted read-only; edit BEFORE starting if needed."
}

# --- Interactive port validation with strict range checking
prompt_and_validate_port() {
    log_info "Interactive port configuration..."
    local USER_PORT="" confirm=""
    while true; do
        echo -n "Enter the host port for AI Proxy (1-65535): "
        read -r USER_PORT

        if ! [[ "${USER_PORT}" =~ ^[0-9]+$ ]]; then
            log_error "Port must be a numeric value. Please enter a number between 1 and 65535."
            continue
        fi
        if [[ "${USER_PORT}" -lt 1 || "${USER_PORT}" -gt 65535 ]]; then
            log_error "Port must be between 1 and 65535. Got: ${USER_PORT}"
            continue
        fi
        if [[ "${USER_PORT}" -lt 1024 ]]; then
            log_warn "Port ${USER_PORT} is in the reserved range (< 1024). Ensure you have proper privileges."
            echo -n "Continue with port ${USER_PORT}? (y/N): "
            read -r confirm
            if [[ ! "${confirm}" =~ ^[Yy]$ ]]; then
                continue
            fi
        fi
        # best-effort check if in use
        if command -v ss >/dev/null 2>&1; then
            if ss -ltn 2>/dev/null | awk '{print $4}' | grep -qE "[:.]${USER_PORT}$"; then
                log_warn "Port ${USER_PORT} appears to be in use."
                echo -n "Continue with port ${USER_PORT}? (y/N): "
                read -r confirm
                if [[ ! "${confirm}" =~ ^[Yy]$ ]]; then
                    continue
                fi
            fi
        elif command -v netstat >/dev/null 2>&1; then
            if netstat -tulpn 2>/dev/null | grep -q ":${USER_PORT} "; then
                log_warn "Port ${USER_PORT} appears to be in use."
                echo -n "Continue with port ${USER_PORT}? (y/N): "
                read -r confirm
                if [[ ! "${confirm}" =~ ^[Yy]$ ]]; then
                    continue
                fi
            fi
        fi
        log_info "Port validated: ${USER_PORT}"
        APP_PORT="${USER_PORT}"
        break
    done
}

resolve_port(){
  if [[ -n "${APP_PORT}" ]]; then
    log_info "Using APP_PORT from env: ${APP_PORT}"
    return
  fi
  # Try image label (in case of rebuilds)
  local lbl=""
  if [[ "${CONTAINER_RUNTIME}" == "docker" ]]; then
    lbl="$(docker image inspect "${IMAGE_NAME_LOCAL}" --format '{{ index .Config.Labels "ai-proxy.build.port" }}' 2>/dev/null || true)"
  else
    lbl="$(podman image inspect "${IMAGE_NAME_LOCAL}" --format '{{ index .Labels "ai-proxy.build.port" }}' 2>/dev/null || true)"
  fi
  if [[ -n "${lbl}" && "${lbl}" =~ ^[0-9]+$ ]]; then
    APP_PORT="${lbl}"
    log_info "Using APP_PORT from image label: ${APP_PORT}"
    return
  fi
  if [[ -t 0 ]]; then
    prompt_and_validate_port
  else
    APP_PORT="4000"
    log_warn "Non-interactive shell; defaulting APP_PORT=${APP_PORT}"
  fi
}

# --- Patch the build context files to use the chosen port
patch_build_context_port(){
  local df="${BUILD_CTX}/Dockerfile"
  if [[ -f "${df}" ]]; then
    # Ensure file ends with newline (prevents CMD and LABEL on same line)
    tail -c1 "${df}" | read -r _ || echo >> "${df}"

    # LABEL ai-proxy.build.port="NNNN"
    if grep -qE 'LABEL[[:space:]]+["'"'"']?ai-proxy\.build\.port["'"'"']?=' "${df}"; then
      sed -i -E "s/(LABEL[[:space:]]+\"?ai-proxy\.build\.port\"?=)\"?[0-9]+\"?/\\1\"${APP_PORT}\"/g" "${df}"
    else
      printf '\nLABEL ai-proxy.build.port="%s"\n' "${APP_PORT}" >> "${df}"
    fi

    # EXPOSE NNNN  (replace first EXPOSE, else append)
    if grep -qE '^[[:space:]]*EXPOSE[[:space:]]+[0-9]+' "${df}"; then
      sed -i -E "0,/^[[:space:]]*EXPOSE[[:space:]]+[0-9]+/s//EXPOSE ${APP_PORT}/" "${df}"
    else
      printf 'EXPOSE %s\n' "${APP_PORT}" >> "${df}"
    fi

    log_info "Patched Dockerfile LABEL/EXPOSE to port ${APP_PORT}"
  else
    log_warn "Dockerfile not found at ${df}; skipping Dockerfile patch."
  fi

  # (Optional) Patch provider-config.yaml default port if it has a top-level 'port:'
  local yml="${BUILD_CTX}/provider-config.yaml"
  if [[ -f "${yml}" ]]; then
    if grep -qE '^[[:space:]]*port:[[:space:]]*[0-9]+' "${yml}"; then
      sed -i -E "s/^[[:space:]]*port:[[:space:]]*[0-9]+/port: ${APP_PORT}/" "${yml}"
      log_info "Patched provider-config.yaml default port to ${APP_PORT}"
    fi
  fi
}

# --- Ensure CONFIG_DIR/.env reflects the chosen port (runtime)
patch_runtime_env(){
  sudo mkdir -p "${CONFIG_DIR}"
  sudo touch "${CONFIG_DIR}/.env"
  # Update/append common keys
  if sudo grep -qE '^(PORT|APP_PORT|SERVICE_PORT)=' "${CONFIG_DIR}/.env"; then
    sudo sed -i -E "s/^PORT=.*/PORT=${APP_PORT}/; s/^APP_PORT=.*/APP_PORT=${APP_PORT}/; s/^SERVICE_PORT=.*/SERVICE_PORT=${APP_PORT}/" "${CONFIG_DIR}/.env"
  else
    echo "PORT=${APP_PORT}"     | sudo tee -a "${CONFIG_DIR}/.env" >/dev/null
    echo "APP_PORT=${APP_PORT}" | sudo tee -a "${CONFIG_DIR}/.env" >/dev/null
  fi
  sudo chmod 0644 "${CONFIG_DIR}/.env"
  log_info "Wrote runtime port into ${CONFIG_DIR}/.env"
}

build_image(){
  log_info "Building image ${IMAGE_NAME_LOCAL} from ${BUILD_CTX}"
  pushd "${BUILD_CTX}" >/dev/null
  "${CONTAINER_RUNTIME}" build -t "${IMAGE_NAME_LOCAL}" .
  popd >/dev/null
  log_info "Docker image built successfully: ${IMAGE_NAME_LOCAL}"
}

ensure_volumes(){
  "${CONTAINER_RUNTIME}" volume create "${VAR_VOLUME}" >/dev/null 2>&1 || true
  "${CONTAINER_RUNTIME}" volume create "${LOG_VOLUME}" >/dev/null 2>&1 || true
}

run_container(){
  if exists_container; then
    log_warn "Removing pre-existing container ${CONTAINER_NAME}..."
    teardown_container
  fi
  ensure_volumes

  log_info "Deploying production container..."

  # Build a literal URL for the in-container healthcheck
  local hc_url="http://127.0.0.1:${APP_PORT}${HEALTH_PATH}"

  # If HEALTHCHECK_CMD is provided, use it exactly; otherwise compose based on tool.
  local hc_cmd=""
  if [[ -n "${HEALTHCHECK_CMD}" ]]; then
    hc_cmd="${HEALTHCHECK_CMD}"
  else
    case "${HEALTHCHECK_TOOL}" in
      curl)
        hc_cmd="sh -c 'curl -fsS --max-time 2 \"${hc_url}\" >/dev/null || exit 1'"
        ;;
      wget|*)
        hc_cmd="sh -c 'wget -q -T 2 -O- \"${hc_url}\" >/dev/null || exit 1'"
        ;;
    esac
  fi

  "${CONTAINER_RUNTIME}" run -d --name "${CONTAINER_NAME}" --restart unless-stopped \
    -u "${APP_UID}:${APP_GID}" \
    --workdir /app \
    -v "${CONFIG_DIR}:/app/config${CONFIG_MOUNT_SUFFIX}" \
    -v "${VAR_VOLUME}:/app/var" \
    -v "${LOG_VOLUME}:/app/log" \
    -p "${APP_PORT}:${APP_PORT}" \
    --cap-drop=ALL --security-opt no-new-privileges \
    --health-cmd "${hc_cmd}" \
    --health-interval "${HEALTHCHECK_INTERVAL}" \
    --health-timeout "${HEALTHCHECK_TIMEOUT}" \
    --health-retries "${HEALTHCHECK_RETRIES}" \
    --health-start-period "${HEALTHCHECK_START_PERIOD}" \
    "${IMAGE_NAME_LOCAL}" \
    --config "./config/provider-config.yaml" \
    --env-file "./config/.env"

  if ! exists_container; then
    die "Failed to start container: ${CONTAINER_NAME}" 10
  fi
  log_info "Production container deployed successfully"
}

probe_http(){
  local url="http://127.0.0.1:${APP_PORT}${HEALTH_PATH}"
  if command -v curl >/dev/null 2>&1; then
    curl -fsS --max-time 2 "${url}" >/dev/null
  else
    wget -q -T 2 -O- "${url}" >/dev/null
  fi
}

wait_for_health_or_fail(){
  local deadline=$(( $(date +%s) + HEALTH_TIMEOUT_SECS ))
  local streak=0
  local url="http://127.0.0.1:${APP_PORT}${HEALTH_PATH}"
  log_info "Waiting for service health on ${url} (timeout ${HEALTH_TIMEOUT_SECS}s, success streak ${HEALTH_SUCCESS_STREAK}) ..."
  while true; do
    if ! is_running; then
      log_error "Container is not running (exited prematurely)."
      return 1
    fi
    if probe_http; then
      streak=$((streak+1))
      if [[ "${streak}" -ge "${HEALTH_SUCCESS_STREAK}" ]]; then
        log_info "Health probe passed ${HEALTH_SUCCESS_STREAK} times."
        return 0
      fi
    else
      streak=0
    fi
    if [[ $(date +%s) -ge "${deadline}" ]]; then
      log_error "Health probe timed out after ${HEALTH_TIMEOUT_SECS}s."
      return 1
    fi
    sleep 1
  done
}

main(){
  detect_runtime
  ensure_tools
  extract_release
  prepare_config_dir
  resolve_port
  patch_build_context_port
  patch_runtime_env
  build_image
  detect_selinux_suffix
  run_container

  if wait_for_health_or_fail; then
    log_info "Service is healthy and running on port ${APP_PORT}."
    exit 0
  else
    exit 12
  fi
}

main "$@"
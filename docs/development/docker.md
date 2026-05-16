# Docker

Build and run the Nylas CLI in a container.

> **Quick Links:** [README](../../README.md) | [Development](../DEVELOPMENT.md) | [Security](../security/overview.md)

---

## Build Locally

```bash
docker build -t nylas-cli:dev .
docker run --rm nylas-cli:dev --version
docker run --rm nylas-cli:dev commands --json
```

The image runs as the unprivileged `nylas` user with `NYLAS_DISABLE_KEYRING=true` (containers don't have a keyring).

---

## Usage

Pass your API key as an environment variable and run commands directly:

```bash
docker run --rm \
  -e NYLAS_API_KEY="$NYLAS_API_KEY" \
  ghcr.io/nylas/cli:latest \
  email list --limit 5 --json
```

For commands that need a grant ID:

```bash
docker run --rm \
  -e NYLAS_API_KEY="$NYLAS_API_KEY" \
  -e NYLAS_GRANT_ID="$NYLAS_GRANT_ID" \
  ghcr.io/nylas/cli:latest \
  calendar list --json
```

Do not bake API keys or credentials into the image.

---

## Persistent Configuration (Optional)

For repeated use, you can run `init` with mounted volumes so config survives between runs:

```bash
docker volume create nylas-config
docker volume create nylas-cache

docker run --rm -it \
  -v nylas-config:/home/nylas/.config/nylas \
  -v nylas-cache:/home/nylas/.cache/nylas \
  ghcr.io/nylas/cli:latest \
  init
```

Then reuse the volumes for later commands without passing env vars each time.

---

## Web Interfaces

`nylas air`, `nylas ui`, and `nylas chat` bind to `localhost` inside the container, so published ports won't expose them to the host. Use Docker for CLI commands only.

---

## Release Images

Tagged releases publish multi-platform images to GitHub Container Registry:

```bash
docker pull ghcr.io/nylas/cli:latest
docker pull ghcr.io/nylas/cli:1.5.0
```

The release workflow packages GoReleaser Linux artifacts into `linux/amd64` and `linux/arm64` images.

# Production image for the Next.js frontend.
#
# Built from `output: 'standalone'`, so the runtime stage carries server.js and the handful of
# modules the app actually reached — not node_modules, not the source, not the build toolchain
# (FR-017). It runs as a non-root user (FR-018).
#
# The development image is a separate file (Dockerfile.dev): it mounts the source and runs
# `next dev`, which has nothing in common with this.

# ------------------------------------------------------------------------------------------------
# Dependencies
# ------------------------------------------------------------------------------------------------
FROM node:22-bookworm-slim AS deps

WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

# ------------------------------------------------------------------------------------------------
# Build
# ------------------------------------------------------------------------------------------------
FROM node:22-bookworm-slim AS build

WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .

# Baked in, not set at runtime. Next compiles rewrites into the build manifest, so the backend
# origin has to be known here; a value set on the running container would be ignored silently.
# Changing which backend this image proxies to means rebuilding it.
ARG VAULTORY_BACKEND_ORIGIN=http://backend:8080
ENV VAULTORY_BACKEND_ORIGIN=${VAULTORY_BACKEND_ORIGIN} \
    NEXT_TELEMETRY_DISABLED=1 \
    NODE_ENV=production

RUN npm run build

# ------------------------------------------------------------------------------------------------
# Runtime
# ------------------------------------------------------------------------------------------------
FROM node:22-bookworm-slim AS runtime

WORKDIR /app

# HOSTNAME is the same trap as the Go service's listen address: standalone's server.js binds to
# localhost unless told otherwise, and a container bound to loopback publishes a port that answers
# nothing.
ENV NODE_ENV=production \
    NEXT_TELEMETRY_DISABLED=1 \
    PORT=3000 \
    HOSTNAME=0.0.0.0

# Three copies, all of them necessary. standalone alone boots but serves no CSS or JS chunks:
# the build deliberately leaves .next/static and public/ out of it so they can be put on a CDN.
COPY --from=build --chown=node:node /app/.next/standalone ./
COPY --from=build --chown=node:node /app/.next/static ./.next/static
COPY --from=build --chown=node:node /app/public ./public

# `node` is uid 1000, and ships with the base image.
USER node

EXPOSE 3000

# No npm in front of it: npm would be PID 1 and would swallow SIGTERM, leaving the container to be
# killed on a timeout at every restart.
CMD ["node", "server.js"]

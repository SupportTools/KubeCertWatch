# No `# syntax=` directive on purpose: BuildKit would pull the docker/dockerfile frontend from Docker Hub.
#
# Fetch tiers (SupportTools cluster-services CLAUDE.md build rule; cluster-services-1498):
#   - base images: the Harbor Docker Hub proxy cache (a cache we run), tag AND digest pinned (resolved
#     2026-11-19). The builder matches go.mod's `toolchain go1.23.4`; GOTOOLCHAIN=local never downloads one.
#   - Go modules: Athens (GOPROXY=https://goproxy.support.tools, a cache we run). CI passes Athens only;
#     the default falls through to proxy.golang.org off-network. go.sum verifies every module.
#   - apk (runtime: ca-certificates, bash, curl, openssl for the emSign guard): the org fetch-cache
#     /alpine/ prefix (in-cluster name, resolves inside a buildx RUN on the self-hosted-linux runners).
#     APK_FALLBACK=none (CI) refuses upstream; apk verifies package signatures either way. The shipped
#     image's /etc/apk/repositories is restored to dl-cdn.alpinelinux.org.
ARG GO_IMAGE=harbor.support.tools/dockerhub/library/golang:1.23.4-alpine3.21@sha256:c23339199a08b0e12032856908589a6d41a0dab141b8b3b21f156fc571a3f1d3
ARG RUNTIME_IMAGE=harbor.support.tools/dockerhub/library/alpine:3.22@sha256:49340b346798875b5592e5d420241f2f9cfba5f47e62ebb1446a1b34734e898f

FROM ${GO_IMAGE} AS builder
WORKDIR /src
ARG GOPROXY=https://goproxy.support.tools|https://proxy.golang.org
ENV GOPROXY=${GOPROXY} \
    GOTOOLCHAIN=local \
    CGO_ENABLED=0
COPY go.mod go.sum ./
# -x prints every "# get <url>" line, so the build log shows which proxy served each module.
RUN echo "go module source: GOPROXY=${GOPROXY}" && go mod download -x
COPY . .
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown
RUN GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w \
    -X github.com/supporttools/KubeCertWatch/pkg/version.Version=${VERSION} \
    -X github.com/supporttools/KubeCertWatch/pkg/version.GitCommit=${GIT_COMMIT} \
    -X github.com/supporttools/KubeCertWatch/pkg/version.BuildTime=${BUILD_DATE}" \
    -o /cert-monitor

FROM ${RUNTIME_IMAGE}
ARG APK_MIRROR=http://fetch-cache.fetch-cache.svc.a1-ops-prd.local/alpine
ARG APK_FALLBACK=upstream
RUN set -eu; \
    repos=/etc/apk/repositories; cp -p "$repos" /tmp/repositories.orig; \
    if [ -n "${APK_MIRROR}" ]; then sed -i "s#https\?://dl-cdn.alpinelinux.org/alpine#${APK_MIRROR}#" "$repos"; fi; \
    if [ -n "${APK_MIRROR}" ] && apk update >/dev/null 2>&1; then \
      echo "apk source: mirror ${APK_MIRROR}"; \
    else \
      [ -z "${APK_MIRROR}" ] || echo "apk source: MIRROR FAILED (${APK_MIRROR})"; \
      if [ "${APK_FALLBACK}" = "none" ]; then echo "APK_FALLBACK=none: refusing upstream" >&2; exit 1; fi; \
      cp -p /tmp/repositories.orig "$repos"; echo "apk source: upstream"; \
    fi; \
    cat "$repos"; \
    apk add --no-cache ca-certificates bash curl openssl; \
    cp -p /tmp/repositories.orig "$repos"; rm -f /tmp/repositories.orig; rm -rf /var/cache/apk/*

COPY --from=builder /cert-monitor /cert-monitor

# Fleet CA standard (operator decision 2026-11-13, cluster-services-1375): NO emSign roots in any file of
# the image. Last root step. noemsign-ca.sh is a vendored copy of SupportTools/cluster-services
# noemsign/noemsign-ca.sh (keep the sha256 identical). guard fails the build on any emSign certificate.
COPY --chmod=0755 noemsign-ca.sh /usr/local/sbin/noemsign-ca.sh
RUN /usr/local/sbin/noemsign-ca.sh strip / && /usr/local/sbin/noemsign-ca.sh guard /

USER 65532:65532
ENTRYPOINT ["/cert-monitor"]

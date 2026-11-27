# SPDX-License-Identifier: GPL-3.0-or-later

FROM --platform=$BUILDPLATFORM node:24-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN mkdir -p ../internal/ui/dist && npm run build

FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
COPY --from=web /src/internal/ui/dist/ internal/ui/dist/
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/golub ./cmd/golub

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/golub /golub
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/golub"]

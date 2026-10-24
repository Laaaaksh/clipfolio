# syntax=docker/dockerfile:1

FROM node:22-slim AS frontend
WORKDIR /src
COPY web/dashboard/package*.json web/dashboard/
COPY web/player/package*.json web/player/
RUN cd web/dashboard && npm install && cd ../player && npm install
COPY web/dashboard web/dashboard
COPY web/player web/player
RUN mkdir -p internal/api/dashboarddist internal/api/playerdist
RUN cd web/dashboard && npm run build
RUN cd web/player && npm run build

FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /src/internal/api/dashboarddist internal/api/dashboarddist
COPY --from=frontend /src/internal/api/playerdist internal/api/playerdist
RUN CGO_ENABLED=0 go build -o /out/clipfolio ./cmd/clipfolio

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ffmpeg ca-certificates \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/clipfolio /usr/local/bin/clipfolio
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/clipfolio"]

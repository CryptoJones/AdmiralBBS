# --- build stage ---------------------------------------------------------
# Pure-Go build (modernc.org/sqlite needs no cgo), so CGO stays off and we get
# a fully static binary that runs on a scratch-style base.
FROM golang:1.26-alpine AS build
WORKDIR /src

# Cache modules first.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" \
    -o /out/admiralbbs ./src/cmd/admiralbbs

# --- runtime stage -------------------------------------------------------
FROM alpine:3.20
# Non-root user (DECISIONS.md: the daemon never runs as root).
RUN adduser -D -h /home/bbs bbs
WORKDIR /home/bbs

COPY --from=build /out/admiralbbs /usr/local/bin/admiralbbs
COPY --chown=bbs:bbs art/ ./art/

# Persistent state (DB, audit log, ssh host key) lives here.
RUN mkdir -p /home/bbs/data && chown bbs:bbs /home/bbs/data
VOLUME /home/bbs/data

USER bbs
EXPOSE 2323 2222

ENTRYPOINT ["admiralbbs"]
CMD ["-telnet", ":2323", "-ssh", ":2222", \
     "-db", "/home/bbs/data/admiralbbs.db", \
     "-audit", "/home/bbs/data/audit.jsonl", \
     "-hostkey", "/home/bbs/data/ssh_host_ed25519_key", \
     "-art", "art/welcome.ans"]

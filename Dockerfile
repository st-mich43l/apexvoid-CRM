FROM golang:1.25-alpine AS backend-builder
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/apexvoid-server ./cmd/server

FROM alpine:3.22 AS backend
RUN addgroup -S apexvoid \
    && adduser -S -G apexvoid apexvoid \
    && mkdir -p /var/lib/apexvoid/attachments \
    && chown apexvoid:apexvoid /var/lib/apexvoid /var/lib/apexvoid/attachments \
    && chmod 0750 /var/lib/apexvoid /var/lib/apexvoid/attachments
WORKDIR /app
COPY --from=backend-builder /out/apexvoid-server /app/apexvoid-server
COPY config/application.yaml /app/config/application.yaml
USER apexvoid
EXPOSE 6868
ENTRYPOINT ["/app/apexvoid-server"]

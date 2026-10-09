FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY integration ./integration
COPY test/fixtures/external-module ./test/fixtures/external-module
RUN CGO_ENABLED=0 go build -o /fixture ./test/fixtures/external-module

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /fixture /fixture
EXPOSE 8090
USER nonroot:nonroot
ENTRYPOINT ["/fixture"]

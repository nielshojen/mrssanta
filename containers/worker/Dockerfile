FROM golang:1.26.1 AS build

ARG TARGETARCH=amd64

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/worker .

FROM gcr.io/distroless/static-debian13:nonroot

COPY --from=build /out/worker /worker

USER nonroot:nonroot

ENTRYPOINT ["/worker"]

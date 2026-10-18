FROM golang:1.25 AS build

ARG TARGETARCH=amd64

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/eventupload ./cmd

FROM gcr.io/distroless/static-debian13:nonroot

COPY --from=build /out/eventupload /eventupload

ENV PORT=8080

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/eventupload"]

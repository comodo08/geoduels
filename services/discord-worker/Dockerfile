FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/discord-worker ./services/discord-worker

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/discord-worker /discord-worker
EXPOSE 8094
ENTRYPOINT ["/discord-worker"]

FROM golang:1.27.3-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY *.go ./
COPY internal ./internal
COPY web ./web
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /timo .

FROM alpine:3.23
RUN addgroup -S timo && adduser -S -G timo timo && mkdir -p /data && chown timo:timo /data
COPY --from=build /timo /usr/local/bin/timo
USER timo
ENV TIMO_DATA_FILE=/data/data.json
VOLUME ["/data"]
EXPOSE 8765
ENTRYPOINT ["timo"]
CMD ["web", "--host", "0.0.0.0", "--no-open"]

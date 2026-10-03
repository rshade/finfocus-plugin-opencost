FROM golang:1.27.1 AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/finfocus-plugin-opencost ./cmd/finfocus-plugin-opencost

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/finfocus-plugin-opencost /finfocus-plugin-opencost
COPY config.example.yaml /etc/finfocus/opencost/config.example.yaml
COPY plugin.manifest.json /etc/finfocus/opencost/plugin.manifest.json
USER nonroot:nonroot
ENTRYPOINT ["/finfocus-plugin-opencost"]

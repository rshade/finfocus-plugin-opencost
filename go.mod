module github.com/rshade/finfocus-plugin-kubecost

go 1.27.1

require (
	google.golang.org/grpc v1.69.4
	google.golang.org/protobuf v1.36.7
	gopkg.in/yaml.v3 v3.0.1
)

require (
	golang.org/x/net v0.32.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241202173237-19429a94021a // indirect
)

// Uncomment below when using local finfocus-spec (development)
// replace github.com/rshade/finfocus-spec => ../finfocus-spec

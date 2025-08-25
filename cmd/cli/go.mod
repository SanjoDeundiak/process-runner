module github.com/SanjoDeundiak/process-runner/cmd/cli

go 1.24

require (
	github.com/SanjoDeundiak/process-runner/api v0.0.0
	github.com/spf13/cobra v1.8.1
	google.golang.org/grpc v1.74.2
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250528174236-200df99c418a // indirect
	google.golang.org/protobuf v1.36.7 // indirect
)

replace github.com/SanjoDeundiak/process-runner/api => ../../api

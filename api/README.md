# Generate

From the project root directory:
```sh
protoc -I=. --go_out=. --go-grpc_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative  ./api/v1/process_runner.proto
```

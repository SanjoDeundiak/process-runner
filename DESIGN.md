# Design

## Scope

- Design a minimal library that allows
  - starting processes on Linux
  - managing their state
  - streaming their output
- Design a secure service that exposes the library API.
- Design a nice CLI tool that wraps the service client and provides good user experience. 

## Design

The service should:
- be able to start a process given a command and arguments; get status; stop a process.
- be able to stream process' stdout/stderr from the beginning. Output is binary and may or may not be a valid UTF-8 string.
- support multiple concurrent subscribers to output, with efficient, non‑polling delivery.
- have authentication via mutual TLS.
- have authorization that limits Stop and GetStatus calls only to the creator of the process.
- remain stateless across restarts (no persistence).
- serve its API using [this gRPC spec](proto/v1/process_runner.proto).
- stop all processes and clean all resources on shutdown.

The library should:
- provide a nice Go API for working with processes.
- expose all the functionality that the service needs.
- apply per‑process resource limits (CPU, memory, disk I/O) on Linux using cgroups (v2).

The CLI should:
- wrap the service client in a nice user experience.
- support all the commands above.
- allow using authentication and authorization methods mentioned above.

## CLI

### Examples
- Start
    ```sh
    prn start -- ping google.com
    > fcd2310e7b9a462a5d88c3a0ff4e2d9b # prints process_id (not PID)
    ```
  > This command is not idempotent and CLI doesn't implement retries. In case of a failure, the user is responsible
  for figuring out if the process was started or not (e.g., by calling `list` command that should be added in the future).

- Status if the process was created by that client
    ```sh
    prn status fcd2310e7b9a462a5d88c3a0ff4e2d9b
  
    +--------------------------------------------+-----------------+
    |                  ID              |  STATE  |     COMMAND     |
    +----------------------------------+---------+-----------------+
    | fcd2310e7b9a462a5d88c3a0ff4e2d9b | Running | ping google.com |
    +----------------------------------+---------+-----------------+
    ```

- Status if the process was created by a different client
    ```sh
    prn status fcd2310e7b9a462a5d88c3a0ff4e2d9b
  
    > Forbidden. Only the creator of the process can get its status.
    ```

- Logs (full replay)
    ```sh
    prn logs fcd2310e7b9a462a5d88c3a0ff4e2d9b
  
    PING google.com (142.250.203.142): 56 data bytes
    64 bytes from 142.250.203.142: icmp_seq=0 ttl=118 time=6.417 ms
    64 bytes from 142.250.203.142: icmp_seq=1 ttl=118 time=7.190 ms
    64 bytes from 142.250.203.142: icmp_seq=2 ttl=118 time=7.157 ms
    64 bytes from 142.250.203.142: icmp_seq=3 ttl=118 time=11.059 ms
    ```

- Stop if the process was created by that client
    ```sh
    prn stop fcd2310e7b9a462a5d88c3a0ff4e2d9b
  
    +--------------------------------------------+-----------------+
    |                  ID              |  STATE  |     COMMAND     |
    +----------------------------------+---------+-----------------+
    | fcd2310e7b9a462a5d88c3a0ff4e2d9b | Stopped | ping google.com |
    +----------------------------------+---------+-----------------+
    ```
  > This command is idempotent.

- Stop if the process was created by a different client
    ```sh
    > Forbidden. Only the creator of the process can stop it.
    ```

- Server start
    ```sh
    prn server start
  
    > Listening on 127.0.0.1:50051
    ```

### Environment variables
CLI supports the following environment variables for all commands:

#### Mandatory
- `PRN_TLS_KEY` is expected to contain the TLS secret key in PKCS#8 format.
- `PRN_TLS_CERT` is expected to contain the TLS certificate in `.pem` format.
- `PRN_CA_TLS_CERT` is expected to contain the CA TLS certificate in `.pem` format.

#### Optional
- `PRN_ADDRESS` can be used to change the address of the server (defaults to `127.0.0.1:50051`)

To set the environment variables, you can either use `export` command:
```sh
export PRN_TLS_KEY=$(cat key.pem)
export PRN_TLS_CERT=$(cat cert.pem)
export PRN_CA_TLS_CERT=$(cat ca_cert.pem)

prn start -- ping google.com
```

Or provide them when running the command:
```sh
PRN_TLS_KEY=$(cat key.pem) PRN_TLS_CERT=$(cat cert.pem) PRN_CA_TLS_CERT=$(cat ca_cert.pem) prn start -- ping google.com
```

## Process execution lifecycle

### Starting a process

1. Allocate id and a working dir
    - `processId := new_uuidv4()`.
    - create the working directory `/var/lib/prn/<processId>`.
2. Set up cgroup (v2)
    - Create `/sys/fs/cgroup/prn/<processId>/`
    - Activate `cpu`, `io`, and `memory` controllers in `/sys/fs/cgroup/cgroup.controllers` 
    - Write limits:
      - cpu.weight = 100
      - io.weight = 100 (applied to all devices by default)
      - memory.high = 512Mb
3. Start the process:
    - Use `exec.Command()`
    - Create a new process group to manage possible child processes easier.
    - Use [clone3](https://manpages.debian.org/testing/manpages-dev/clone3.2.en.html#CLONE_INTO_CGROUP) which is
       [supported in Go](https://cs.opensource.google/go/go/+/refs/tags/go1.25.0:src/syscall/exec_linux.go;l=107)
       to not let a process run without cgroup for a short period of time (which it could use to fork).
4. Wire up I/O:
   - create Linux pipes for stdout&stderr using `Cmd.StdoutPipe()` and `Cmd.StderrPipe()`
   - create a goroutine for each pipe that will read from it and append new data to a single linked list dedicated
     to that process
   - implement that single linked list using `atomic.Pointer`. There is only one writer for that list,
     so an atomic pointer should be enough to achieve concurrency safety.
   - notify the readers that new data is available in the list (using `sync.Cond` and `Broadcast()`, need to be careful
     here to avoid race conditions)
5. Track process via `pidfd`.

### Stopping a job

1. Write `1` to `cgroup.kill` (kills all descendants atomically).
2. Wait for `pidfd` to signal exit, collect status, usage.
3. Cleanup: purge cgroup, finalize logs.
4. Leverage Reaper notion to avoid zombies.

## Error handling
Error handling should align with
- [gRPC conventions](https://grpc.io/docs/guides/error/)
- [Go conventions](https://go.dev/blog/error-handling-and-go)

## Security

1. This service has a massive attack surface. I would start with talking to stakeholders to understand better why
   it's necessary, if we can reduce it to running only specific commands, if there is an XY problem,
   and what the alternatives are.
2. Use all the goodies Linux have to restrict the spawned process:
    - use [cgroups](https://man7.org/linux/man-pages/man7/cgroups.7.html) to limit the usage of resources (CPU, memory, disk IO, etc.)
    - more ways in the [Future improvements section](#f)
3. Use [mTLS](https://www.cloudflare.com/en-gb/learning/access-management/what-is-mutual-tls/) to authenticate the client and the server:
    - TLS 1.3 should be enforced on both sides with no downgrade option.
    - Any cipher suite from TLS 1.3 is fine.
    - I will stick to Ed25519 for the signing keys, as it offers the best security/performance ratio IMO.
    - Introduce a certificate authority (CA) to sign the server and client certificates.
    - CA certificate may be self-signed.
    - CA should issue certificates to clients and the server.
    - Certificates may be long-lived (e.g., 1 year). 
    - Server is identified as `server` and clients are identified as `client_{n}`, where `{n}` is a number.
    - Identity is expressed as `URI:spiffe://server` and `URI:spiffe://client_{n}` strings respectively that are stored
      inside certificate's SAN field ([SPIFFE](https://spiffe.io/)).
4. Use simple authorization:
    - Start and GetOutput calls are available to parties with `client_{n}` identity.
    - Stop and GetStatus calls are available only to the identity that started that process. 
5. The private key is consciously passed to the binary as an environment variable. This way the secret key can be
   injected by whatever method is chosen for deployment (e.g., as a k8s secret). The server process should
   wipe the environment variable as soon as it's done reading it.

## Testing

- Unit tests for process lifecycle.
- cgroups behavior tests (limits applied and enforced).
- e2e CLI tests.
- Output replay for late subscribers.
- Authentication tests:
  - what happens if a client or server tries to use a secret key that doesn't match the certificate
  - what happens if a client or server tries to use an expired certificate
- Authorization test:
  - what happens if a party tries to make a GetStatus or Stop call on a process that it didn't start
- Run race detector.

## <a name="f"></a> Future improvements

- Better process isolation:
    - run spawned processes in namespaces to isolate them from the host (possibly the server process itself as well),
      or at least use a dedicated Linux user.
    - use [seccomp](https://blog.cloudflare.com/sandboxing-in-linux-with-zero-lines-of-code/) to restrict system calls the process can make
    - use [capabilities](https://man7.org/linux/man-pages/man7/capabilities.7.html) to restrict the process's privileges
    - limit what a client can run, e.g., a predefined list of commands. Depends on the UX and security requirements.
    - limit the number of concurrent processes to avoid resource exhaustion.
- OpenTelemetry traces/metrics; structured logging and audit logs.
- Create a Dockerfile for deployment.
- Use secure key management for TLS secret key (e.g., OS keyring/KMS).
- Introduce a mechanism for CA to authorize clients and servers during the certificate issuance
- Make client and server certificates short-lived (e.g., 30 minutes) to force them to rotate their key regularly, which
  will also imply CA authorization check each time.
- Configurable resource limits per request and/or policy.
- gRPC compression (e.g., for GetOutput).
- [gRPC health checking](https://github.com/grpc/grpc/blob/master/doc/health-checking.md)
- Installer packaging (e.g., Homebrew).
- CLI help/manpages.
- Fine‑grained RBAC/ABAC.
- Add list command.

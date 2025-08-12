# Design

> Items marked with “*” are production‑grade improvements not planned for the interview submission.

## Scope

- Design a minimal library that allows
  - starting processes on Linux
  - managing their state
  - streaming their output
- Design a secure service that exposes the library API.
- Design a nice CLI tool that wraps the service client and provides good user experience. 

## Design

The service should:
- be able to start a process by command, list processes, get status, stop a process
- be able to stream its stdout/stderr from the beginning. Output is binary and may or may not be a valid UTF-8 string.
- support multiple concurrent subscribers to output, with efficient, non‑polling delivery.
- apply per‑process resource limits (CPU, memory, disk I/O) on Linux using cgroups (v2).
- have authentication via mutual TLS.
- have authorization via certificate pinning.
- remain stateless across restarts (no persistence).
- serve its API using [this gRPC spec](proto/v1/process_runner.proto).
- stop all processes and clean all resources on shutdown.
- `*` have Dockerfile for deployment.
- `*` use secure key management (e.g., OS keyring/KMS).

The library should:
- provide a nice Go API for working with processes.
- expose all the functionality that the service needs.

The CLI should:
- wrap the service client in a nice user experience.
- support all the commands above.
- allow using authentication and authorization methods mentioned above.

## CLI

### Examples
- Start
    ```sh
    prn start "ping google.com" --client-cert ./client.pem --server-cert ./server.pem
    > fcd2310e7b9a462a5d88c3a0ff4e2d9b # prints process_id (not PID)
    ```
  > This command is not idempotent and CLI doesn't implement retries. In case of a failure, the user is responsible
  for figuring out if the process was started or not (e.g., by calling `list` command).

- Status
    ```sh
    prn status fcd2310e7b9a462a5d88c3a0ff4e2d9b --client-cert ./client.pem --server-cert ./server.pem
  
    +--------------------------------------------+-----------------+
    |                  ID              |  STATE  |     COMMAND     |
    +----------------------------------+---------+-----------------+
    | fcd2310e7b9a462a5d88c3a0ff4e2d9b | Running | ping google.com |
    +----------------------------------+---------+-----------------+
    ```

- Logs (full replay)
    ```sh
    prn logs fcd2310e7b9a462a5d88c3a0ff4e2d9b --client-cert ./client.pem --server-cert ./server.pem
  
    PING google.com (142.250.203.142): 56 data bytes
    64 bytes from 142.250.203.142: icmp_seq=0 ttl=118 time=6.417 ms
    64 bytes from 142.250.203.142: icmp_seq=1 ttl=118 time=7.190 ms
    64 bytes from 142.250.203.142: icmp_seq=2 ttl=118 time=7.157 ms
    64 bytes from 142.250.203.142: icmp_seq=3 ttl=118 time=11.059 ms
    ```

- Stop
    ```sh
    prn stop fcd2310e7b9a462a5d88c3a0ff4e2d9b --client-cert ./client.pem --server-cert ./server.pem
  
    +--------------------------------------------+-----------------+
    |                  ID              |  STATE  |     COMMAND     |
    +----------------------------------+---------+-----------------+
    | fcd2310e7b9a462a5d88c3a0ff4e2d9b | Stopped | ping google.com |
    +----------------------------------+---------+-----------------+
    ```
  > This command is idempotent.

- List
    ```sh
    prn list --client-cert ./client.pem --server-cert ./server.pem
  
    +--------------------------------------------+-----------------+
    |                  ID              |  STATE  |     COMMAND     |
    +----------------------------------+---------+-----------------+
    | fcd2310e7b9a462a5d88c3a0ff4e2d9b | Running | ping google.com |
    | a3e56c1fb29d8a5477c2f819ef34a65d | Running | top             |
    +----------------------------------+---------+-----------------+
    ```

- Server start
    ```sh
    prn server start \
      --server-cert ./server.pem \
      --client-cert ./client1.pem \
      --client-cert ./client2.pem \
      --client-cert ./client3.pem
  
    > Listening on 127.0.0.1:50051
    ```

### Environment variables
- `PRN_TLS_KEY` is expected to contain the TLS secret key in PKCS#8 format.
- `PRN_ADDRESS` can be used to change the address of the server (defaults to `127.0.0.1:50051`)

## Process execution lifecycle

### Starting a process

1. Allocate id and a working dir
    - `processId := newid()`.
    - create the working directory `/var/lib/prn/<processId>`.
2. Set up cgroup (v2)
    - Create `/sys/fs/cgroup/prn/<processId>/`
    - Write limits: CPU, Memory, IO.
3. Fork/exec with supervision
    - Use `exec.Command()`
    - Create a new process group to manage possible child processes easier.
    - Join the cgroup by writing PID to cgroup.procs.
4. Wire up I/O - create pipes for stdout&stderr, wrap with a ring buffer (e.g., 10–50 MB).
5. Track process via `pidfd`.

### Stopping a job (graceful then hard)

1. SIGTERM the process group, allowing app shutdown hooks.
2. Timeout (e.g., 10s), keep streaming logs.
3. If still alive, SIGKILL the cgroup:
    - Preferred: write 1 to cgroup.kill (kills all descendants atomically).
    - Fallback: kill(-pgid, SIGKILL).
4. Wait for `pidfd` to signal exit, collect status, usage.
5. Cleanup: purge cgroup, finalize logs.
6. Leverage Reaper notion to avoid zombies.

## Error handling
Error handling should align with
- [gRPC conventions](https://grpc.io/docs/guides/error/)
- [Go conventions](https://go.dev/blog/error-handling-and-go).

## Security

1. This service has a massive attack surface. I would start with talking to stakeholders to understand better why
   it's necessary, if we can reduce it to running only specific commands, if there is an XY problem,
   and what the alternatives are.
2. Use all the goodies Linux have to restrict the spawned process:
    - use [cgroups](https://man7.org/linux/man-pages/man7/cgroups.7.html) to limit the usage of resources (CPU, memory, disk IO, etc.)
    - `*` run spawned processes in namespaces to isolate them from the host (possibly the server process itself as well),
      or at least use a dedicated Linux user.
    - `*` use [seccomp](https://blog.cloudflare.com/sandboxing-in-linux-with-zero-lines-of-code/) to restrict system calls the process can make
    - `*` use [capabilities](https://man7.org/linux/man-pages/man7/capabilities.7.html) to restrict the process's privileges
    - `*` limit what a client can run, e.g., a predefined list of commands. Depends on the UX and security requirements.
    - `*` limit the number of concurrent processes to avoid resource exhaustion.
3. Use [mTLS](https://www.cloudflare.com/en-gb/learning/access-management/what-is-mutual-tls/) to authenticate the client and the server:
    - TLS 1.3 should be enforced on both sides with no downgrade option.
    - Any cipher suite from TLS 1.3 is fine.
    - I will stick to Ed25519 for the signing keys, as it offers the best security/performance ratio IMO.
    - Self-signed certificates seem fine given the authorization system (see below).
    - The validity period of a certificate may be restricted and checked on both sides.
    - `*` Having a certificate authority instead of self-signed certificates can make things easier to use, especially
      if the number of clients is large.
4. Use certificate pinning on both client and server side for authorization
    - On startup the server is provided with a set of client certificates and only accepts connections from clients that are in the set.
    - On startup the client is provided with the server certificate and only connects to the server if the server certificate matches.
    - The server doesn't distinguish different clients, they all have the same access rights.
    - This is the most non-versatile authorization system one can think of, yet very secure and simple.
    - Certificate rotation and revocation are painful processes in this case.
5. The private key is consciously passed to the binary as an environment variable. This way the secret key can be
   injected by whatever method is chosen for deployment (e.g., as a k8s secret). The server process should
   wipe the environment variable as soon as it's done reading it.

## Testing

- Unit tests for process lifecycle.
- cgroups behavior tests (limits applied and enforced).
- e2e CLI tests.
- Output replay for late subscribers.
- Authorization test: what happens if a client or server tries to use a secret key that doesn't match the certificate.
- Authorization test: what happens if a client or server tries to use a certificate that's not known to the other party.
- Run race detector.

## Future improvements

- `*` OpenTelemetry traces/metrics; structured logging and audit logs.
- `*` Configurable resource limits per request and/or policy.
- `*` gRPC compression (e.g., for GetOutput).
- `*` [gRPC health checking](https://github.com/grpc/grpc/blob/master/doc/health-checking.md)
- `*` Installer packaging (e.g., Homebrew).
- `*` CLI help/manpages.
- `*` Pin certificate fingerprint(s) instead of full certs.
- `*` Fine‑grained RBAC/ABAC.

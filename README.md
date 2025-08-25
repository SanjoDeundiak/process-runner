# Process runner
This is a utility to run processes remotely.

## Design
Refer to [design.md](./DESIGN.md)

## Instructions

Run from root of repository to generate all required keys and certificates:

```sh
./scripts/generate_all.sh
```

To set up env variables to run as a specific actor there is a small script:

```sh
./scripts/set_evn.sh {NAME}
```

Possible valid values for `{NAME}` are:
 - server
 - client1
 - client2

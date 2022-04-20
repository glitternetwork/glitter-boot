## glitter-boot

Glitter bootstrap tool

### Install
```
go install github.com/glitternetwork/glitter-boot/cmd/glitter-boot@v0.1.1
```

## Commands

```shell
# ./glitter-boot 
Glitter bootstrap tool

Usage:
  glitter-boot [command]

Available Commands:
  completion     Generate the autocompletion script for the specified shell
  help           Help about any command
  init           init node
  show-node-info show node info
  start          start [target: `fullnode` or `validator`]
  stop           stop glitter and tendermint services

Flags:
  -h, --help   help for glitter-boot

Use "glitter-boot [command] --help" for more information about a command.
```

### init
Download glitter binary and init services

> Before executing this command, you need to create the `glitter` user and user group

- Argumets

|Name|Description|Required|Default|
|---|---|---|---|
|`seeds`|seed nodes for connect to testnet|true|""|
|`moniker`|moniker for node|true|""|
|`indexer`|fullnode indexMode 'es' or 'kv'|false|"kv"|
|`glitter_bin_url`|glitter download url|false|"https://storage.googleapis.com/glitterprotocol.appspot.com/tendermint"|
|`tendermint_bin_url`|tendermint download url|false|"https://storage.googleapis.com/glitterprotocol.appspot.com/glitter-v0.1.0/glitter"|

### start
Start as fullnode or validator

### stop
Stop all services

### show-node-info
Show node info

- Example

```
NodeID:         3d187f86dde4a5f5f412cb282e52a59838d68bad
Moniker:        node3

PubKey:         xxxy5074AbhOITINxFBqp/cQ4rZVEwen3JlZnRkrcII=
Address:        XXXX62A7A195983BABE17299EC375A486B25E1C5

Tendermint Status: active

Glitter    Status: active


PrivateKeyFile: ~/.glitter-boot/priv_validator_key.json
GlitterBootDir: ~/.glitter-boot
```
### Options

```
  -h, --help   help for glitter-boot
```
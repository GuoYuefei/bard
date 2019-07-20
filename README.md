- [Getting Started](#getting-started)
  + [Install](#install)
  + [Building the Application](#building-the-application)
  + [Usage](#usage)
- [Node](#node)
- [License](#license)
- [Contributing](#contributing)
- [Preface](#preface)

## Getting Started

### Install

```shell
go get -u install github.com/GuoYuefei/bard
```

### Building the Application

Now that you have do something it's time to build your application:

```shell
cd bard
go mod tidy
go build -o proxyserver ./server/server.go
go build -o proxyclient ./client/client.go
```

Then you can see two executable file named proxyserver and proxyclient in your current folder.

### Usage

The config.yml file in the ./debug/config folder. Configure the file based on the comment information in the file. 

The path name of the configuration file relative to the execution file is <code>./server/debug/config/config.yml</code> and <code>./client/debug/config/config.yml</code>.

At this time you can execute the program you just compiled.

```shell
./proxyserver 
./proxyclient
```
The plugin system allows you to place the plugins you need to open in the server/debug/plugin/ and client/debug/plugin/ directory.  
If you don't want to write the plugin yourself, I will provide a few plugins available.
As for the documentation needed to write the plugin, it will be released after I release the v1.0.0 version.

## Node 

Since golang does not support building plugins on Windows, the plugin system is temporarily invalid for Windows. Looking for a plugin system alternative. RFC may be used.

## License

GNU AFFERO GENERAL PUBLIC LICENSE (AGPLv3.0)

## Contributing

See the file named CONTRIBUTING.md

## Preface

Write a proxy software based on the socks5 protocol. Functional implementations are expected to have basic functionality including, but not limited to, support for udp and ipv6 proxy functions, as well as some special features including but not limited to encryption, etc. 

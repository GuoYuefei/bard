- [Getting Started](#getting-started)
  + [Install](#install)
  + [Building the Application](#building-the-application)
  + [Usage](#usage)
- [License](#license)
- [Contributing](#contributing)
- [Preface](#preface)

## Getting Started

### Install

```go
go get -u install github.com/GuoYuefei/bard
```

### Building the Application

Now that you have do something it's time to build your application:

```go
go install all-package
go build server.go
```

Then you can see the executable file named server in your current folder.

### Usage

The config.yml file in the ./debug/config folder. Configure the file based on the comment information in the file. 

The path name of the configuration file relative to the execution file is ./Debug/config/config.yml

At this time you can execute the program you just compiled.

```go
./server 
```

## License

GNU AFFERO GENERAL PUBLIC LICENSE (AGPLv3.0)

## Contributing

waiting.....

## Preface

Write a proxy software based on the socks5 protocol. Functional implementations are expected to have basic functionality including, but not limited to, support for udp and ipv6 proxy functions, as well as some special features including but not limited to encryption, etc.
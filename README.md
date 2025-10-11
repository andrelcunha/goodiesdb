# Goodies DB - A Redis implementation in Go

[![Go](https://github.com/andrelcunha/goodiesdb/actions/workflows/go.yml/badge.svg)](https://github.com/andrelcunha/goodiesdb/actions/workflows/go.yml)
[![Docker Image CI](https://github.com/andrelcunha/goodiesdb/actions/workflows/docker-image.yml/badge.svg)](https://github.com/andrelcunha/goodiesdb/actions/workflows/docker-image.yml)

GoodiesDb started as a Redis implementation written in Go, serving as an educational project to learn and understand the inner workings of Redis, a popular in-memory data structure store. The current state of the project implements a subset of Redis's commands, including `AUTH`, `SET`, `GET`, `DEL`, `EXISTS`, `SETNX`, `EXPIRE`, `INCR`, `DECR`, `TTL`, `SELECT`, `LPUSH`, `RPUSH`, `LPOP`, `RPOP`, `LRANGE`, `LTRIM`, `RENAME`, `TYPE`, `KEYS`, `INFO`, `PING`, `ECHO`, `QUIT`, `FLUSHDB` and `FLUSHALL`.

**Disclaimer:** This is not a production-ready Redis clone and it is not intended for use in production environments (yet).

---

## Table of Contents

- [Introduction](#introduction)
- [Features](#features)
- [Installation](#installation)
- [Usage](#usage)
- [License](#license)
- [Acknowledgements](#acknowledgements)

## Introduction

GoodiesDB aims to mimic the basic functionalities of Redis to provide a learning platform for developers interested in understanding distributed systems, data structures, and high-performance computing.

## Features

- In-memory key-value store
- Data persistence using RDB and AOF
- Support for lists, sets, and hash maps (planned)
- Publish/Subscribe messaging (planned)
- Basic transaction support
- Lua scripting execution (planned)
- Master-slave replication (planned)
- Sharding (planned)

## Installation

To get started with Redis Clone, follow these steps:

1. **Clone the repository**:

    ```bash
    git clone https://github.com/andrelcunha/goodiesdb.git
    cd GoodiesDB
    ```

2. **Install dependencies**:

    ```bash
    go mod tidy
    ```

3. **Build the project**:

    ```bash
    make build
    ```

## Usage

Run the GoodiesDb server:

```bash
make run
```

You can then interact with the server using PuTTY on raw TCP port 6379.

## License

This project is licensed under the MIT License.

## Acknowledgements

- [Redis](https://redis.io/documentation) for the inspiration and original implementation.
- [Golang](https://golang.org/) for the programming language.

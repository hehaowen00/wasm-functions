# wasm-functions

Functions as a Service platform using WASM + WASI modules

DO NOT USE IN PRODUCTION.

## Functions

Each function has its own configuration stored in the `app` sqlite database.
Environment variables are function specific.
Folders can be shared between functions.

Functions are called via the main or `_start` symbol.

Functions receive HTTP request payloads via the `REQUEST` environment variable.
Due to limitations with the go version of `wasmer`, `stdin` are inherited from 
the host process. The rust crate for `wasmer` supports specifying individual streams 
for `stdin` and `stdout`.

Functions respond by printing out a formatted HTTP response to stdout.
HTTP request processing must be done by the function.

Functions are accessed using the `/wasm/{id}` HTTP endpoint.

## Usage

[Functions](USAGE.md)

```
go build .
./wasm-functions
```
## Examples

[examples](/examples)

A todo REST API is implemented using Rust compiled to WASM.
Each rust crate corresponds to one function.
Database functionality is achieved using sqlite via the Rusqlite and sqlite-vfs crates.

Due to limitations in compilation, only the main function can be called.
In order to have more than one function exported in Rust, crates need to be compiled as a library
using `crate-type = ["cdylib"]`. This is currently not possible with rusqlite and/or 
sqlite-vfs.


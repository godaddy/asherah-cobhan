# Go Warmup Library

## Purpose

The `go_warmup` library provides a minimal Go shared library that can be loaded via FFI from JavaScript runtimes to initialize the Go runtime before loading CGO-based N-API modules. This prevents runtime errors that occur when crypto/x509 initialization runs in certain JavaScript runtime environments.

## Background

Some JavaScript runtimes (like Bun) don't properly initialize the Go runtime when loading CGO-based N-API modules, leading to crashes during crypto/x509 initialization. By loading this minimal warmup library first via FFI, we ensure the Go runtime is properly initialized before the main library loads.

## Implementation

The library exports a single function:
- `Warmup() int` - Initializes Go runtime and returns 1

## Build Artifacts

Platform-specific libraries are built and included in releases:
- **Linux x64**: `go-warmup-linux-x64.so`
- **Linux ARM64**: `go-warmup-linux-arm64.so`
- **Darwin x64**: `go-warmup-darwin-x64.dylib`
- **Darwin ARM64**: `go-warmup-darwin-arm64.dylib`

## Usage Example

JavaScript runtimes can use this library to ensure Go runtime compatibility:

```javascript
// Detect runtime and load warmup library if needed
if (typeof Bun !== 'undefined') {
  const { dlopen, FFIType } = require('bun:ffi');
  const lib = dlopen('path/to/go-warmup-platform.so', {
    Warmup: { returns: FFIType.int, args: [] }
  });
  lib.symbols.Warmup();
}
// Now safe to load CGO-based N-API modules
```

## Testing

The warmup library has been tested with asherah-node in the Bun runtime and confirmed to resolve compatibility issues, enabling full functionality of CGO-based N-API modules.
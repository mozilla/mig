# Client Daemon API Types

*Note*: In the case that a type is suffixed by `?`, meanining that it
can be `undefined`, or `type | null`, the value can be left out. This
notation is used to describe configuration parameters that are optional
and have defaults.

In this example, `value` and `other` can be left out, so the following are
all valid values of type `Example`.

```typescript
type Example = {
    value: string | null,
    other: number?,
    required: string
}

const ex1: Example = {
    value: "test",
    other: 32,
    required: "no"
}

const ex2: Example = {
    value: null,
    required: "hello"
}

const ex3: Example = {
    required: "test"
}
```

## API Types

```typescript
type Parameter = {
    name: string,
    type: string,
    description: string,
    example: any,
}

type Endpoint = {
    method: string,
    route: string,
    parameters: Array<Parameter>,
    response: Array<Parameter>,
    statusCodes: {
        [code: number]: string,
    }
}

type RawSQL = string

type Target = RawSQL

type RegExp = string

type Module
    = AgentDestroyModule
    | AuditModule
    | DispatchModule
    | PingModule
    | ScribeModule
    | TimeDriftModule
    | PkgModule
    | FileModule
    | FSWatchModule
    | MemoryModule
    | NetStatModule
    | SSHKeyModule
    | YaraModule

type AgentDestroyModule = {
    pid: number,
    version: string
}

// No parameters
type AuditModule = {
}

// No parameters
type DispatchModule = {
}

type PingModule = {
    destination: string,
    protocol: 'tcp' | 'udp' | 'icmp',
    destinationPort: number?,
    count: number?,
    timeout: number?
}

type ScribeModule = {
    path: string,
    onlyTrueDocTests: bool?,
    humanReadableOutput: bool?,
    jsonOutput: bool?
}

// No parameters
type TimeDriftModule = {
}

type PkgModule = {
    name: RegExp,
    version: string | null
}

type FileModule = {
    options: null | {
        maxDepth: number?,
        matchAll: bool?,
        matchAny: bool?,
        matchEntireFile: bool??,
        findMismatchingFiles: bool?,
        limit: number?,
        includeFileSha256: bool?,
        decompressFiles: bool?,
        maxErrors: number?
    },
    search: {
        path: string,
        name: string?,
        description: string?
        content: string?,
        sizeBytes: number?,
        modifiedSinceMinutes: number?,
        modifiedAfterMinutes: number?,
        mode: string?,
        md5: string?,
        sha1: string?,
        sha2: string?,
        sha3: string?,
    }
}

// No parameters
type FSWatchModule = {
}

type MemoryModule = {
    options: null | {
        offset: number?,
        maxLength: number?,
        logFailures: bool?,
        matchAll: bool?
    },
    search: {
        description: string?,
        name: string?,
        library: string?,
        bytes: string?,
        content: string?
    }
}

type NetStatModule = {
    localMACAddress: string?,
    neighborMACAddress: string?,
    localIPAddress: string?,
    neighborIPAddress: string?,
    remoteConnectedIPAddress: string?,
    listeningPort: number?,
    resolveNamespaces: bool?
}

type SSHKeyModule = {
    path: string,
    maxDepth: number?
}

type YaraModule = {
    yaraRules: string,
    fileSearch: string
}
```

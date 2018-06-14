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

type RegExp = string

// Multiple target queries can be specified and will be joined with `AND`.
type Target = Array<TargetQuery>

type TargetQuery
    = TargetWithSQL
    | TargetAll
    | TargetByAgentDetails
    | TargetByHostDetails
    | TargetByTag

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

// Contains whatever result information is produced by agents that have run an action.
type Result = {
    errors: Array<string>,
    foundanything: boolean,
    success: boolean,
    elements: Array<object>,
    statistics: Array<object>
}

// This type is a fallback for investigators to write the arbitrary SQL queries they may have used
// for advanced targeting.
type TargetWithSQL = {
    sql: RawSQL
}

// Regardless of whether `all` is `true` or `false`, this type specifies all online agents as targets.
type TargetAll = {
    all: bool
}

// Enables targeting agents by details specific to a given agent.
// At least one field must be present for this target query to be considered valid.
type TargetByAgentDetails = {
    id: number?,
    name: string?,
    queueLocation: string?,
    version: string?,
    pid: number?,
    status: string?
}

// Enables targeting agents by details specific to the agent's host environment.
// At least one field must be present for this target query to be considered valid.
// See the [cheatsheet](https://github.com/mozilla/mig/blob/master/doc/cheatsheet.rst#environments)
// for more information about this.
type TargetByHostDetails = {
    ident: string?,
    os: string?,
    arch: string?,
    publicIP: string?
}

// Enables targeting agents by one of their tags.
type TargetByTag = {
    tagName: string,
    value: string
}

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

type TimeDriftModule = {
    drift: string
}

type PkgModule = {
    packageName: RegExp,
    packageVersion: string | null
}

type FileModule = {
    options: null | {
        maxDepth: number?,
        matchAll: bool?,
        matchAny: bool?,
        matchEntireFile: bool?,
        mismatchingContent: string?,
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
        minSizeBytes: number?,
        maxSizeBytes: number?,
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

## Examples

### Target

```typescript
[
    {
        tagName: "operator",
        value: "IT"
    },
    {
        os: "linux"
    }
]
```

```typescript
[
    {
        os: "linux"
    }
    {
        name: "buildbot"
    }
]
```

```typescript
[
    {
        sql: "id IN (SELECT agentid FROM commands, json_array_elements(commands.results) AS r WHERE commands.actionid = 12345 AND r#>>'{foundanything}' = 'true')"
    }
]
```

```typescript
[
    {
        all: true
    }
]
```

### Module

Ping module

```typescript
{
    destination: "127.0.0.1",
    destinationPort: 8080,
    protocol: "tcp"
}
```

File module

```typescript
{
    options: {
        maxDepth: 1,
        limit: 100
    },
    search: {
        path: "/etc/passwd",
        modifiedSinceMinutes: 2880
    }
}
```

NetStat module

```typescript
{
    remoteConnectedIPAddress: "1.2.3.4"
}
```

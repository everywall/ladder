## TLDR
- If you create, delete or rename any request/response modifier, run `go run codegen.go`, so that ruleset unmarshaling will work properly.

## Overview

The `codegen.go` file is a utility for the rulesets that automatically generates Go code that maps functional options names found in response/request modifiers to corresponding factory functions. This generation is crucial for the serialization of rulesets from JSON or YAML into functional options suitable for use in proxychains. The tool processes Go files containing modifier functions and generates the necessary mappings.

- The generated mappings will be written in `proxychain/ruleset/rule_reqmod_types.gen.go` and `proxychain/ruleset/rule_resmod_types.gen.go`.
- These files are used in UnmarshalJSON and UnmarshalYAML methods of the rule type, found in `proxychain/ruleset/rule.go`


## Usage
```sh
go run codegen.go
```


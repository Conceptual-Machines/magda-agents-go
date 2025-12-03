# MAGDA Agents - Go Implementation

Go implementation of the MAGDA Agents framework.

## Installation

```bash
go get github.com/Conceptual-Machines/magda-agents-go
```

## Usage

```go
import (
    "github.com/Conceptual-Machines/magda-agents-go/agents/daw"
    "github.com/Conceptual-Machines/magda-agents-go/config"
)

cfg := &config.Config{
    OpenAIAPIKey: "your-key",
}

agent := daw.NewDawAgent(cfg)
result, err := agent.GenerateActions(ctx, question, state)
```

## Documentation

For complete documentation, see the main [MAGDA Agents](https://github.com/Conceptual-Machines/magda-agents) repository.

## License

AGPL v3 - See LICENSE file for details.

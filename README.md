# go-expect

[![Build Status](https://travis-ci.org/Netflix/go-expect.svg?branch=master)](https://travis-ci.org/Netflix/go-expect)
[![GoDoc](https://godoc.org/github.com/Netflix/go-expect?status.svg)](https://godoc.org/github.com/Netflix/go-expect)
[![NetflixOSS Lifecycle](https://img.shields.io/osslifecycle/Netflix/go-expect.svg)]()

Package expect provides an expect-like interface to automate control of terminal or console based programs. It is unlike expect and other go expect packages in that it does not spawn or control process lifecycle. This package only interfaces with a stdin and stdout and controls the interaction through those files alone.

## Usage

```go
package prompt

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrompt(t *testing.T) {
  t.Parallel()

  c, err := expect.NewTestConsole(t)
  require.Nil(t, err)
  defer c.Close()

  cmd := exec.Command("prompt")
  cmd.Stdin = c.Stdin()
  cmd.Stdout = c.Stdout()
  cmd.Stderr = c.Stdout()

  go func() {
    c.Expect("What is 1+1?")
    c.SendLine("2")
    c.Expect("What is Netflix backwards?")
    c.SendLine("xilfteN")
    c.ExpectEOF()
  }()

  err = cmd.Run()
  require.Nil(t, err)
}
```

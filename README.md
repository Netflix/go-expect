# go-expect

Package expect provides an expect-like interface to automate control of terminal or console based programs. It is unlike expect and other go expect packages in that it does not spawn or control process lifecycle. This package only interfaces with a stdin and stdout and controls the interaction through those files alone.

## Usage

```go
c, err := NewConsole()
if err != nil {
  log.Fatal(err)
}

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
if err != nil {
  log.Fatal(err)
}
c.Close()
```

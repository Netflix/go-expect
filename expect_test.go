// Copyright 2018 Netflix, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package expect

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
)

var (
	ErrWrongAnswer = errors.New("wrong answer")
)

type Survey struct {
	Prompt string
	Answer string
}

func Prompt(in io.Reader, out io.Writer) error {
	reader := bufio.NewReader(in)

	for _, survey := range []Survey{
		{
			"What is 1+1?", "2",
		},
		{
			"What is Netflix backwards?", "xilfteN",
		},
	} {
		fmt.Fprint(out, fmt.Sprintf("%s: ", survey.Prompt))
		text, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		fmt.Fprint(out, text)
		text = strings.TrimSpace(text)
		if text != survey.Answer {
			return ErrWrongAnswer
		}
	}

	return nil
}

func TestExpect(t *testing.T) {
	t.Parallel()

	c, err := NewTestConsole(t)
	if err != nil {
		t.Errorf("Expected no error but got'%s'", err)
	}
	defer c.Close()

	go func() {
		c.ExpectString("What is 1+1?")
		c.SendLine("2")
		c.ExpectString("What is Netflix backwards?")
		c.SendLine("xilfteN")
		c.ExpectEOF()
	}()

	err = Prompt(c.Tty(), c.Tty())
	if err != nil {
		t.Errorf("Expected no error but got '%s'", err)
	}
}

func TestExpectOutput(t *testing.T) {
	t.Parallel()

	c, err := NewTestConsole(t)
	if err != nil {
		t.Errorf("Expected no error but got'%s'", err)
	}
	defer c.Close()

	go func() {
		c.ExpectString("What is 1+1?")
		c.SendLine("3")
		c.ExpectEOF()
	}()

	err = Prompt(c.Tty(), c.Tty())
	if err == nil || err != ErrWrongAnswer {
		t.Errorf("Expected error '%s' but got '%s' instead", ErrWrongAnswer, err)
	}
}

func TestConsoleChain(t *testing.T) {
	t.Parallel()

	c1, err := NewConsole()
	if err != nil {
		t.Errorf("Expected no error but got'%s'", err)
	}
	defer c1.Close()

	go func() {
		c1.ExpectString("What is Netflix backwards?")
		c1.SendLine("xilfteN")
		c1.ExpectEOF()
	}()

	c2, err := NewTestConsole(t, WithStdin(c1.Tty()), WithStdout(c1.Tty()))
	if err != nil {
		t.Errorf("Expected no error but got'%s'", err)
	}
	defer c2.Close()

	go func() {
		c2.ExpectString("What is 1+1?")
		c2.SendLine("2")
		c2.ExpectEOF()
	}()

	err = Prompt(c2.Tty(), c2.Tty())
	if err != nil {
		t.Errorf("Expected no error but got '%s'", err)
	}
}

func TestEditor(t *testing.T) {
	if _, err := exec.LookPath("vi"); err != nil {
		t.Skip("vi not found in PATH")
	}
	t.Parallel()

	c, err := NewConsole()
	if err != nil {
		t.Errorf("Expected no error but got '%s'", err)
	}
	defer c.Close()

	file, err := ioutil.TempFile("", "")
	if err != nil {
		t.Errorf("Expected no error but got '%s'", err)
	}

	cmd := exec.Command("vi", file.Name())
	cmd.Stdin = c.Tty()
	cmd.Stdout = c.Tty()
	cmd.Stderr = c.Tty()

	go func() {
		c.Send("iHello world\x1b")
		c.SendLine(":wq!")
		c.ExpectEOF()
	}()

	err = cmd.Run()
	if err != nil {
		t.Errorf("Expected no error but got '%s'", err)
	}

	data, err := ioutil.ReadFile(file.Name())
	if err != nil {
		t.Errorf("Expected no error but got '%s'", err)
	}
	if string(data) != "Hello world\n" {
		t.Errorf("Expected '%s' to equal '%s'", string(data), "Hello world\n")
	}
}

func ExampleConsoleCat() {
	c, err := NewConsole(WithStdout(os.Stdout))
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	cmd := exec.Command("cat")
	cmd.Stdin = c.Tty()
	cmd.Stdout = c.Tty()
	cmd.Stderr = c.Tty()

	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	c.Send("Hello world")
	c.ExpectString("Hello world")
	c.Close()
	c.ExpectEOF()

	err = cmd.Wait()
	if err != nil {
		log.Fatal(err)
	}

	// Output: Hello world
}

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
	"runtime/debug"
	"strings"
	"sync"
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

func expectNoError(t *testing.T) ConsoleOpt {
	return WithExpectObserver(
		func(matcher Matcher, buf string, err error) {
			if matcher == nil && err != nil {
				t.Errorf("Error occured while matching %q: %s\n%s", buf, err, string(debug.Stack()))
			} else if err != nil {
				t.Errorf("Failed to find %v in %q: %s\n%s", matcher.Criteria(), buf, err, string(debug.Stack()))
			}
		},
	)
}

func sendNoError(t *testing.T) ConsoleOpt {
	return WithSendObserver(
		func(msg string, n int, err error) {
			if err != nil {
				t.Errorf("Failed to send %q: %s\n%s", msg, err, string(debug.Stack()))
			}
			if len(msg) != n {
				t.Errorf("Only sent %d of %d bytes for %q\n%s", n, len(msg), msg, string(debug.Stack()))
			}
		},
	)
}

func testCloser(t *testing.T, closer io.Closer) {
	if err := closer.Close(); err != nil {
		t.Errorf("Close failed: %s", err)
		debug.PrintStack()
	}
}

func TestExpectf(t *testing.T) {
	t.Parallel()

	c, err := NewTestConsole(t)
	if err != nil {
		t.Errorf("Expected no error but got'%s'", err)
	}
	defer testCloser(t, c)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.Expectf("What is 1+%d?", 1)
		c.SendLine("2")
		c.Expectf("What is %s backwards?", "Netflix")
		c.SendLine("xilfteN")
		c.ExpectEOF()
	}()

	err = Prompt(c.Tty(), c.Tty())
	if err != nil {
		t.Errorf("Expected no error but got '%s'", err)
	}
	testCloser(t, c.Tty())
	wg.Wait()
}

func TestExpect(t *testing.T) {
	t.Parallel()

	c, err := NewTestConsole(t, expectNoError(t), sendNoError(t))
	if err != nil {
		t.Errorf("Expected no error but got'%s'", err)
	}
	defer testCloser(t, c)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
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
	// close the pts so we can expect EOF
	testCloser(t, c.Tty())
	wg.Wait()
}

func TestExpectOutput(t *testing.T) {
	t.Parallel()

	c, err := NewTestConsole(t, expectNoError(t), sendNoError(t))
	if err != nil {
		t.Errorf("Expected no error but got'%s'", err)
	}
	defer testCloser(t, c)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.ExpectString("What is 1+1?")
		c.SendLine("3")
		c.ExpectEOF()
	}()

	err = Prompt(c.Tty(), c.Tty())
	if err == nil || err != ErrWrongAnswer {
		t.Errorf("Expected error '%s' but got '%s' instead", ErrWrongAnswer, err)
	}
	testCloser(t, c.Tty())
	wg.Wait()
}

func TestConsoleChain(t *testing.T) {
	t.Parallel()

	c1, err := NewConsole(expectNoError(t), sendNoError(t))
	if err != nil {
		t.Errorf("Expected no error but got'%s'", err)
	}
	defer testCloser(t, c1)

	var wg1 sync.WaitGroup
	wg1.Add(1)
	go func() {
		defer wg1.Done()
		c1.ExpectString("What is Netflix backwards?")
		c1.SendLine("xilfteN")
		c1.ExpectEOF()
	}()

	c2, err := NewTestConsole(t, WithStdin(c1.Tty()), WithStdout(c1.Tty()), expectNoError(t), sendNoError(t))
	if err != nil {
		t.Errorf("Expected no error but got'%s'", err)
	}
	defer testCloser(t, c2)

	var wg2 sync.WaitGroup
	wg2.Add(1)
	go func() {
		defer wg2.Done()
		c2.ExpectString("What is 1+1?")
		c2.SendLine("2")
		c2.ExpectEOF()
	}()

	err = Prompt(c2.Tty(), c2.Tty())
	if err != nil {
		t.Errorf("Expected no error but got '%s'", err)
	}

	testCloser(t, c2.Tty())
	wg2.Wait()

	testCloser(t, c1.Tty())
	wg1.Wait()
}

func TestEditor(t *testing.T) {
	if _, err := exec.LookPath("vi"); err != nil {
		t.Skip("vi not found in PATH")
	}
	t.Parallel()

	c, err := NewConsole(expectNoError(t), sendNoError(t))
	if err != nil {
		t.Errorf("Expected no error but got '%s'", err)
	}
	defer testCloser(t, c)

	file, err := ioutil.TempFile("", "")
	if err != nil {
		t.Errorf("Expected no error but got '%s'", err)
	}

	cmd := exec.Command("vi", file.Name())
	cmd.Stdin = c.Tty()
	cmd.Stdout = c.Tty()
	cmd.Stderr = c.Tty()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.Send("iHello world\x1b")
		c.SendLine(":wq!")
		c.ExpectEOF()
	}()

	err = cmd.Run()
	if err != nil {
		t.Errorf("Expected no error but got '%s'", err)
	}

	testCloser(t, c.Tty())
	wg.Wait()

	data, err := ioutil.ReadFile(file.Name())
	if err != nil {
		t.Errorf("Expected no error but got '%s'", err)
	}
	if string(data) != "Hello world\n" {
		t.Errorf("Expected '%s' to equal '%s'", string(data), "Hello world\n")
	}
}

func ExampleConsole_echo() {
	c, err := NewConsole(WithStdout(os.Stdout))
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	cmd := exec.Command("echo")
	cmd.Stdin = c.Tty()
	cmd.Stdout = c.Tty()
	cmd.Stderr = c.Tty()

	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	c.Send("Hello world")
	c.ExpectString("Hello world")
	c.Tty().Close()
	c.ExpectEOF()

	err = cmd.Wait()
	if err != nil {
		log.Fatal(err)
	}

	// Output: Hello world
}

// maskFilter will just replace the `mask` byte with an `*` on all reads from
// the io.Reader
type maskFilter struct {
	r    io.Reader
	mask byte
}

func newMaskFilter(mask byte) ExpectFilter {
	return func(r io.Reader) io.Reader {
		return &maskFilter{
			r:    r,
			mask: mask,
		}
	}
}

func (mf *maskFilter) Read(p []byte) (int, error) {
	n, err := mf.r.Read(p)
	for i, _ := range p {
		if p[i] == mf.mask {
			p[i] = '*'
		}
	}
	return n, err
}

func TestExpectFiltered(t *testing.T) {
	t.Parallel()

	c, err := NewTestConsole(t,
		expectNoError(t),
		sendNoError(t),
		// replace 'x' or '1' with '*'
		WithExpectFilter(newMaskFilter('x'), newMaskFilter('1')),
		// verify we do not see any 'x' or '1' in the expect buffer
		WithExpectObserver(
			func(_ Matcher, buf string, _ error) {
				for _, maskChar := range []byte("x1") {
					if strings.Contains(buf, string(maskChar)) {
						t.Errorf("found Mask character [%q] in expect buffer: %q", maskChar, buf)
					}
				}
			},
		),
	)
	if err != nil {
		t.Errorf("Expected no error but got'%s'", err)
	}
	defer testCloser(t, c)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.ExpectString("What is *+*?")
		c.SendLine("2")
		c.ExpectString("What is Netfli* backwards?")
		c.SendLine("xilfteN")
		c.ExpectEOF()
	}()

	err = Prompt(c.Tty(), c.Tty())
	if err != nil {
		t.Errorf("Expected no error but got '%s'", err)
	}

	testCloser(t, c.Tty())
	wg.Wait()
}

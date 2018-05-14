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

// prompt is an example interactive program that can be automated by go-expect.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
)

var (
	ErrWrongAnswer = errors.New("wrong answer")
)

func main() {
	err := prompt()
	if err != nil {
		log.Fatal(err)
	}
}

type Survey struct {
	Prompt string
	Answer string
}

func prompt() error {
	reader := bufio.NewReader(os.Stdin)

	for _, survey := range []Survey{
		{
			"What is 1+1?", "2",
		},
		{
			"What is Netflix backwards?", "xilfteN",
		},
	} {
		fmt.Print(fmt.Sprintf("%s: ", survey.Prompt))
		text, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		fmt.Print(text)
		text = strings.TrimSpace(text)
		if text != survey.Answer {
			return ErrWrongAnswer
		}
	}

	log.Println("Success!")
	return nil
}

/*
Copyright 2018 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package input

import (
	"fmt"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/pipeline/constants"
	"github.com/pkg/errors"
	survey "gopkg.in/AlecAivazis/survey.v1"
)

// UserSelectedList takes in a list of object and returns the user selection
func UserSelectedList(object string, list []string) (string, error) {
	selection := ""
	prompt := &survey.Select{
		Message: fmt.Sprintf("Please select a %s", object),
		Options: list,
	}
	if err := survey.AskOne(prompt, &selection, nil); err != nil {
		return "", err
	}
	if selection == constants.Other {
		return queryAndConfirmUserInput(fmt.Sprintf("Please enter the desired %s:", object))
	}
	return selection, nil
}

// Password makes a user request which requires hidden user input
func Password(message string) (string, error) {
	input := ""
	prompt := &survey.Password{
		Message: message,
	}
	err := survey.AskOne(prompt, &input, nil)
	return input, err
}

func confirmOrQueryUser(input, query string) (string, error) {
	confirmed, err := confirmUserInput(input)
	if err != nil {
		return "", errors.Wrapf(err, "confirming input %s", input)
	}
	if confirmed {
		return input, nil
	}
	newInput, err := queryAndConfirmUserInput(query)
	return newInput, err
}
func confirmUserInput(input string) (bool, error) {
	c := false
	confirm := &survey.Confirm{
		Message: fmt.Sprintf("Is %s correct?", input),
	}
	return c, survey.AskOne(confirm, &c, nil)
}
func queryAndConfirmUserInput(query string) (string, error) {
	input := ""
	correct := false
	for {
		if correct {
			break
		}
		i := &survey.Input{
			Message: query,
		}
		err := survey.AskOne(i, &input, nil)
		if err != nil {
			return "", err
		}
		correct, err = confirmUserInput(input)
		if err != nil {
			return "", err
		}
	}
	return input, nil
}

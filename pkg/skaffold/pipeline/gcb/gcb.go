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

package gcb

import (
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var requiredCloudAPIs = []string{
	"container.googleapis.com",
	"cloudbuild.googleapis.com",
}

// EnableRequiredCloudAPIs enables requiredCloudAPIs
func EnableRequiredCloudAPIs() error {
	for _, r := range requiredCloudAPIs {
		logrus.Debugf("Enabling required cloud API %s", r)
		cmd := exec.Command("gcloud", "services", "enable", r)
		if err := util.RunCmd(cmd); err != nil {
			return errors.Wrapf(err, "enabling cloud API %s", r)
		}
	}
	return nil
}

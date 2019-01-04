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

package kubernetes

// ParseKubernetesYaml attempts to parse k8s objects from a yaml file
// if successful, it will return the images referenced in the k8s config
// so they can be built by the generated skaffold yaml
func ParseKubernetesYaml(filepath string) ([]string, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, errors.Wrap(err, "opening config file")
	}
	r := k8syaml.NewYAMLReader(bufio.NewReader(f))

	objects := []runtime.Object{}
	images := []string{}

	for {
		doc, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "reading config file")
		}
		d := scheme.Codecs.UniversalDeserializer()
		obj, _, err := d.Decode(doc, nil, nil)
		if err != nil {
			return nil, errors.Wrap(err, "decoding kubernetes yaml")
		}

		m := make(map[interface{}]interface{})
		if err := yaml.Unmarshal(doc, &m); err != nil {
			return nil, errors.Wrap(err, "reading kubernetes YAML")
		}

		images = append(images, parseImagesFromYaml(m)...)
		objects = append(objects, obj)
	}
	if len(objects) == 0 {
		return nil, errors.New("no valid kubernetes objects decoded")
	}
	return images, nil
}

// adapted from pkg/skaffold/deploy/kubectl/recursiveReplaceImage()
func parseImagesFromYaml(doc interface{}) []string {
	images := []string{}
	switch t := doc.(type) {
	case []interface{}:
		for _, v := range t {
			images = append(images, parseImagesFromYaml(v)...)
		}
	case map[interface{}]interface{}:
		for k, v := range t {
			if k.(string) != "image" {
				images = append(images, parseImagesFromYaml(v)...)
				continue
			}

			images = append(images, v.(string))
		}
	}
	return images
}
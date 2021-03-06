package transformers

import (
	"sort"
	"strings"

	"github.com/ContainerSolutions/helm-convert/pkg/types"
	ktypes "sigs.k8s.io/kustomize/pkg/types"
)

// imageTagTransformer replace image tags
type imageTagTransformer struct {
	imageTags []ktypes.ImageTag
}

var _ Transformer = &imageTagTransformer{}

// NewImageTagTransformer constructs a imageTagTransformer.
func NewImageTagTransformer() Transformer {
	return &imageTagTransformer{}
}

// Transform finds all images and store them in the kustomization.yaml file
func (pt *imageTagTransformer) Transform(config *ktypes.Kustomization, resources *types.Resources) error {
	for id := range resources.ResMap {
		obj := resources.ResMap[id].Map()
		err := pt.findImage(config, obj)
		if err != nil {
			continue
		}
	}

	sort.Slice(config.ImageTags, func(i, j int) bool {
		return imageTagString(config.ImageTags[i]) < imageTagString(config.ImageTags[j])
	})

	return nil
}

func (pt *imageTagTransformer) findImage(config *ktypes.Kustomization, obj map[string]interface{}) error {
	paths := []string{"containers", "initContainers"}
	found := false
	for _, path := range paths {
		_, found = obj[path]
		if found {
			err := pt.getImageTag(config, obj, path)
			if err != nil {
				return err
			}
		}
	}
	if !found {
		return pt.findContainers(config, obj)
	}
	return nil
}

func (pt *imageTagTransformer) getImageTag(config *ktypes.Kustomization, obj map[string]interface{}, path string) error {
	containers := obj[path].([]interface{})
LOOP_CONTAINERS:
	for i := range containers {
		container := containers[i].(map[string]interface{})
		imagePath, found := container["image"]

		if !found {
			continue
		}

		image := imagePath.(string)

		hasDigest := strings.Contains(image, "@")
		separator := ":"

		if hasDigest {
			separator = "@"
		}

		s := strings.Split(image, separator)

		imageTag := ktypes.ImageTag{
			Name: s[0],
		}

		// doesn't add image if already in the list
		for _, v := range config.ImageTags {
			if v.Name == imageTag.Name {
				continue LOOP_CONTAINERS
			}
		}

		if len(s) > 1 {
			if hasDigest {
				imageTag.Digest = s[1]
			} else {
				imageTag.NewTag = s[1]
			}
		}

		config.ImageTags = append(config.ImageTags, imageTag)
	}
	return nil
}

func (pt *imageTagTransformer) findContainers(config *ktypes.Kustomization, obj map[string]interface{}) error {
	for key := range obj {
		switch typedV := obj[key].(type) {
		case map[string]interface{}:
			err := pt.findImage(config, typedV)
			if err != nil {
				return err
			}
		case []interface{}:
			for i := range typedV {
				item := typedV[i]
				typedItem, ok := item.(map[string]interface{})
				if ok {
					err := pt.findImage(config, typedItem)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func imageTagString(imageTag ktypes.ImageTag) string {
	if imageTag.Digest != "" {
		return imageTag.Name + "@" + imageTag.Digest
	}
	return imageTag.Name + ":" + imageTag.NewTag
}

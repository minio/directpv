package utils

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/yaml"
)

func ParseSingleKubeNativeFromBytes(data []byte) (runtime.Object, error) {
	obj := map[string]interface{}{}
	err := yaml.Unmarshal(data, &obj)
	if err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{
		Object: obj,
	}, nil
}

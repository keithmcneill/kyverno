package kube

import (
	"encoding/json"

	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// BytesToUnstructured converts the resource to unstructured format
func BytesToUnstructured(data []byte) (*unstructured.Unstructured, error) {
	resource := &unstructured.Unstructured{}
	err := resource.UnmarshalJSON(data)
	if err != nil {
		return nil, err
	}
	return resource, nil
}

func ObjToUnstructured(obj interface{}) (*unstructured.Unstructured, error) {
	raw, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	unstrObj := map[string]interface{}{}
	err = json.Unmarshal(raw, &unstrObj)
	if err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: unstrObj}, nil
}

func UnstructuredToBytes(obj *unstructured.Unstructured) ([]byte, error) {
	raw, err := yaml.Marshal(obj.Object)

	if err != nil {
		return nil, err
	}
	return raw, err
}

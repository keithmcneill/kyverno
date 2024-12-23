package resource

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v5"
	"github.com/kyverno/kyverno/cmd/cli/kubectl-kyverno/source"
	yamlutils "github.com/kyverno/kyverno/ext/yaml"
	"github.com/kyverno/kyverno/pkg/client/clientset/versioned/scheme"
	kubeutils "github.com/kyverno/kyverno/pkg/utils/kube"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

func GetUnstructuredResources(resourceBytes []byte) ([]*unstructured.Unstructured, error) {
	documents, err := yamlutils.SplitDocuments(resourceBytes)
	if err != nil {
		return nil, err
	}
	resources := make([]*unstructured.Unstructured, 0, len(documents))
	for _, document := range documents {
		resource, err := YamlToUnstructured(document)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

func YamlToUnstructured(resourceYaml []byte) (*unstructured.Unstructured, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	_, metaData, decodeErr := decode(resourceYaml, nil, nil)
	if decodeErr != nil {
		if !strings.Contains(decodeErr.Error(), "no kind") {
			return nil, decodeErr
		}
	}
	resourceJSON, err := yaml.YAMLToJSON(resourceYaml)
	if err != nil {
		return nil, err
	}
	resource, err := kubeutils.BytesToUnstructured(resourceJSON)
	if err != nil {
		return nil, err
	}
	if decodeErr == nil {
		resource.SetGroupVersionKind(*metaData)
	}
	if resource.GetNamespace() == "" {
		resource.SetNamespace("default")
	}
	return resource, nil
}

func GetResourceFromPath(fs billy.Filesystem, path string) ([]*unstructured.Unstructured, error) {
	var resourceBytes []byte
	if fs == nil {
		data, err := GetFileBytes(path)
		if err != nil {
			return nil, err
		}
		resourceBytes = data
	} else {
		file, err := fs.Open(path)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil {
			return nil, err
		}
		resourceBytes = data
	}
	resources, err := GetUnstructuredResources(resourceBytes)
	if err != nil {
		return nil, err
	}
	if len(resources) == 0 {
		return nil, fmt.Errorf("no resources found")
	}
	return resources, nil
}

func GetFileBytes(path string) ([]byte, error) {
	if source.IsHttp(path) {
		// We accept here that a random URL might be called based on user provided input.
		req, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, err
		}
		file, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return file, nil
	} else {
		path = filepath.Clean(path)
		// We accept the risk of including a user provided file here.
		file, err := os.ReadFile(path) // #nosec G304
		if err != nil {
			return nil, err
		}
		return file, nil
	}
}

func WriteResourceToPath(fs billy.Filesystem, obj *unstructured.Unstructured, path string) error {
	bytes, err := kubeutils.UnstructuredToBytes(obj)
	if err != nil {
		return err
	}
	return WriteBytesToPath(fs, bytes, path)
}

func WriteBytesToPath(fs billy.Filesystem, resourceBytes []byte, path string) error {
	if fs == nil {
		if source.IsHttp(path) {
			return fmt.Errorf("Unable to write resource as source is http: %s", path)
		}

		path = filepath.Clean(path)
		return os.WriteFile(path, resourceBytes, 0644)
	} else {
		path = filepath.Clean(path)
		// Create or open the file
		file, err := fs.Create(path)
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}
		defer file.Close()

		// Write the byte array to the file
		_, err = file.Write(resourceBytes)
		if err != nil {
			return fmt.Errorf("failed to write data to file: %w", err)
		}
		return nil
	}
}

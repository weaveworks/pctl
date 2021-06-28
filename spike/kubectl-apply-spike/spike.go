package spike

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/kubectl/pkg/scheme"
)

func Merge(original, userModified, latest runtime.Object) (runtime.Object, error) {
	originalJson, err := runtime.Encode(unstructured.UnstructuredJSONScheme, original)
	if err != nil {
		return nil, fmt.Errorf("failed to encode: %w", err)
	}

	userModifiedJson, err := runtime.Encode(unstructured.UnstructuredJSONScheme, userModified)
	if err != nil {
		return nil, fmt.Errorf("failed to encode: %w", err)
	}

	latestJson, err := runtime.Encode(unstructured.UnstructuredJSONScheme, latest)
	if err != nil {
		return nil, fmt.Errorf("failed to encode: %w", err)
	}
	patch, err := strategicpatch.CreateThreeWayMergePatch(originalJson, userModifiedJson, latestJson, strategicpatch.PatchMetaFromOpenAPI{Schema: scheme.Scheme}, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create patch: %w", err)
	}
	fmt.Println(string(patch))
	return latest, nil
}

package spike

import (
	"encoding/json"
	"fmt"
	"reflect"

	jsonpatch "gopkg.in/evanphx/json-patch.v4"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/kubectl/pkg/scheme"
	//"sigs.k8s.io/yaml"
)

func Merge(original, userModified, latest runtime.Object) ([]byte, error) {
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

	versionedObject, err := scheme.Scheme.New(original.GetObjectKind().GroupVersionKind())
	if err != nil {
		return nil, fmt.Errorf("failed to get schema from group version kind: %w", err)
	}
	lookupPatchMeta, err := strategicpatch.NewPatchMetaFromStruct(versionedObject)
	if err != nil {
		return nil, fmt.Errorf("failed to get patch meta from struct: %w", err)
	}
	patch, err := strategicpatch.CreateThreeWayMergePatch(originalJson, userModifiedJson, latestJson, lookupPatchMeta, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create patch: %w", err)
	}
	data, err := jsonpatch.MergePatch(latestJson, patch)
	if err != nil {
		return nil, fmt.Errorf("failed to apply merge patch: %w", err)
	}
	r := make(map[string]interface{})
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	//changed := removeNulls(r)
	//for changed {
	//	changed = removeNulls(r)
	//}
	return yaml.Marshal(r)
}

func removeNulls(m map[string]interface{}) bool {
	changed := false
	val := reflect.ValueOf(m)
	for _, e := range val.MapKeys() {
		v := val.MapIndex(e)
		if v.IsNil() {
			delete(m, e.String())
			changed = true
			continue
		}
		if v.IsZero() {
			delete(m, e.String())
			changed = true
			continue
		}
		if i, ok := v.Interface().(map[string]interface{}); ok {
			if len(i) == 0 {
				delete(m, e.String())
				changed = true
				continue
			}
		}
		switch t := v.Interface().(type) {
		case map[string]interface{}:
			removeNulls(t)
		}
	}
	return changed
}

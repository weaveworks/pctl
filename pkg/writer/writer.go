package writer

import (
	"fmt"
	"io"
	"os"

	"k8s.io/apimachinery/pkg/runtime"

	kjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
)

// Writer takes a profile and writes it out into a medium.
type Writer interface {
	Output(obj runtime.Object) error
}

// FileWriter is a Writer using a file as backing medium.
type FileWriter struct {
	Filename string
}

// Output writes the profile subscription yaml data into a given file.
func (fw *FileWriter) Output(obj runtime.Object) error {
	e := kjson.NewSerializerWithOptions(kjson.DefaultMetaFactory, nil, nil, kjson.SerializerOptions{Yaml: true, Strict: true})
	f, err := os.OpenFile(fw.Filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		if err := f.Close(); err != nil {
			fmt.Printf("Failed to close file %s\n", f.Name())
		}
	}(f)
	if err := e.Encode(obj, f); err != nil {
		return err
	}
	return nil
}

// StringWriter is writer which puts the generated output into a provided io.Writer such as bytes.Buffer for example.
type StringWriter struct {
	Out io.Writer
}

// Output will write the generated output into an attached io.Writer like bytes.Buffer.
func (sw *StringWriter) Output(obj runtime.Object) error {
	e := kjson.NewSerializerWithOptions(kjson.DefaultMetaFactory, nil, nil, kjson.SerializerOptions{Yaml: true, Strict: true})
	if err := e.Encode(obj, sw.Out); err != nil {
		return err
	}
	return nil
}

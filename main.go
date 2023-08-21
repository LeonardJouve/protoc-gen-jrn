package main

import (
	"google.golang.org/protobuf/compiler/protogen"
)

func main() {
	protogen.Options{}.Run(func(plugin *protogen.Plugin) error {
		for _, file := range plugin.Files {
			err := generate(plugin, file)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func pascalCase(str string) string {
	return str
}

func generate(plugin *protogen.Plugin, file *protogen.File) error {
	fileName := pascalCase(file.GeneratedFilenamePrefix) + ".java"
	f := plugin.NewGeneratedFile(fileName, file.GoImportPath)

	for _, message := range file.Messages {
		for _, field := range message.Fields {
			f.P(field.Desc.Kind().String())
		}
	}

	return nil
}

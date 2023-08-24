package main

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const template = `package $packageName$;
import androidx.annotation.NonNull;

import com.facebook.react.bridge.Arguments;
import com.facebook.react.bridge.Promise;
import com.facebook.react.bridge.ReactApplicationContext;
import com.facebook.react.bridge.ReactContextBaseJavaModule;
import com.facebook.react.bridge.ReactMethod;
import com.facebook.react.bridge.ReadableMap;
import com.facebook.react.bridge.WritableMap;

import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;

public class GrpcModule extends ReactContextBaseJavaModule {
	private class Messages {
		$*messages*$
	}
	
	String host;
    Integer port;

    GrpcModule(ReactApplicationContext context) {
        super(context);
    }

    private void checkHostAndPort() throws Exception {
        if (this.host == null) {
            throw new Exception("\"host\" is not defined");
        }
        if (this.port == null) {
            throw new Exception("\"port\" is not defined");
        }
    }

    @NonNull
    @Override
    public String getName() {
        return "GrpcModule";
    }

    @ReactMethod()
    public void setHost(String host) {
        this.host = host;
    }

    @ReactMethod()
    public void setPort(Integer port) {
        this.port = port;
    }

	$*methods*$
}`

var kinds = map[protoreflect.Kind]string{
	protoreflect.BoolKind:     "Boolean",
	protoreflect.Int32Kind:    "Int",
	protoreflect.Sint32Kind:   "Int",
	protoreflect.Uint32Kind:   "Int",
	protoreflect.Int64Kind:    "Int",
	protoreflect.Sint64Kind:   "Int",
	protoreflect.Uint64Kind:   "Int",
	protoreflect.Sfixed32Kind: "Int",
	protoreflect.Fixed32Kind:  "Int",
	protoreflect.FloatKind:    "Int",
	protoreflect.Sfixed64Kind: "Int",
	protoreflect.Fixed64Kind:  "Int",
	protoreflect.DoubleKind:   "Double",
	protoreflect.StringKind:   "String",
	// TODO: handle these
	// protogen.EnumKind: ""
	// protogen.BytesKind: "",
	// protogen.MessageKind: "",
	// protogen.GroupKind:   "",
}

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

func generate(plugin *protogen.Plugin, in *protogen.File) error {
	fileName := "GrpcModule.java"
	out := plugin.NewGeneratedFile(fileName, protogen.GoImportPath(upper(string(in.GoImportPath))))

	variables := make(map[string]string)
	lists := make(map[string][]string)

	variables["packageName"] = *in.Proto.Options.JavaPackage

	generateMessages(in.Messages, &lists)

	generateServices(in.Services, &lists)

	out.P(format(template, &variables, &lists))

	return nil
}

func generateServices(services []*protogen.Service, lists *map[string][]string) {
	variables := make(map[string]string)

	for _, service := range services {
		variables["serviceName"] = upper(string(service.Desc.Name()))
		generateMethods(service.Methods, &variables, lists)
	}
}

func generateMethods(methods []*protogen.Method, variables *map[string]string, lists *map[string][]string) {
	for _, method := range methods {
		(*lists)["methods"] = append((*lists)["methods"], generateMethod(method, variables, lists))
	}
}

func generateMethod(method *protogen.Method, variables *map[string]string, lists *map[string][]string) string {
	(*variables)["methodName"] = lower(string(method.Desc.Name()))
	(*variables)["inputKind"] = upper(string(method.Input.Desc.Name()))
	(*variables)["outputKind"] = upper(string(method.Output.Desc.Name()))

	for _, field := range method.Input.Fields {
		inputFieldType, ok := kinds[field.Desc.Kind()]
		if !ok {
			continue
		}
		(*lists)["inputFieldTypes"] = append((*lists)["inputFieldTypes"], upper(inputFieldType))
		(*lists)["inputFieldNamesLower"] = append((*lists)["inputFieldNamesLower"], lower(string(field.Desc.Name())))
		(*lists)["inputFieldNamesUpper"] = append((*lists)["inputFieldNamesUpper"], upper(string(field.Desc.Name())))
	}

	for _, field := range method.Output.Fields {
		outputFieldType, ok := kinds[field.Desc.Kind()]
		if !ok {
			continue
		}
		(*lists)["outputFieldTypes"] = append((*lists)["outputFieldTypes"], upper(outputFieldType))
		(*lists)["outputFieldNamesLower"] = append((*lists)["outputFieldNamesLower"], lower(string(field.Desc.Name())))
		(*lists)["outputFieldNamesUpper"] = append((*lists)["outputFieldNamesUpper"], upper(string(field.Desc.Name())))
	}

	const template = `@ReactMethod()
	public void $methodName$(ReadableMap message, Promise promise) {
		try {
			checkHostAndPort();
			ManagedChannel channel = ManagedChannelBuilder.forAddress(host, port).usePlaintext().build();
			$serviceName$Grpc.$serviceName$BlockingStub stub = $serviceName$Grpc.newBlockingStub(channel);
			$inputKind$ request = Messages.set$inputKind$(message);
			$outputKind$ response = stub.$methodName$(request);
			WritableMap result = Messages.get$outputKind$(response);
			promise.resolve(result);
		} catch (Exception e) {
			promise.reject("Error", "Unable to call remote procedure \"$methodName$\"", e);
		}
	}`

	return format(template, variables, lists)
}

func generateMessages(messages []*protogen.Message, lists *map[string][]string) {
	for _, message := range messages {
		(*lists)["messages"] = append((*lists)["messages"], generateMessage(message))
	}
}

func generateMessage(message *protogen.Message) string {
	variables := make(map[string]string)
	lists := make(map[string][]string)

	variables["messageName"] = upper(string(message.Desc.Name()))

	var nestedMessages string
	for _, nestedMessage := range message.Messages {
		nestedMessages += generateMessage(nestedMessage)
	}
	variables["nestedMessages"] = nestedMessages

	for _, field := range message.Fields {
		fieldKind := field.Desc.Kind()
		fieldType, ok := kinds[fieldKind]
		if !ok {
			if fieldKind == protoreflect.MessageKind {
				lists["fieldTypes"] = append(lists["fieldTypes"], upper(string(field.Message.Desc.Name())))
				lists["fieldNames"] = append(lists["fieldNamesLower"], lower(string(field.Desc.Name())))
				lists["fieldNames"] = append(lists["fieldNamesUpper"], upper(string(field.Desc.Name())))
			}
			continue
		}
		lists["fieldTypes"] = append(lists["fieldTypes"], fieldType)
		lists["fieldNamesLower"] = append(lists["fieldNamesLower"], lower(string(field.Desc.Name())))
		lists["fieldNamesUpper"] = append(lists["fieldNamesUpper"], upper(string(field.Desc.Name())))
	}

	template := `private $messageName$ set$messageName$(ReadableMap message) {
			return $messageName$.newBuilder()
				.set$*fieldNamesUpper*$(message.get$*fieldTypes*$("$*fieldNamesLower*$"))
				.build();
		}
		private WritableMap get$messageName$($messageName$ message) {
			WritableMap writableMap = Arguments.createMap();
			writableMap.put$*fieldTypes*$("$*fieldNamesLower*$", message.get$*fieldNamesUpper*$());
			return writableMap;
		}`
	if len(nestedMessages) > 0 {
		template += `
		$nestedMessages$`
	}

	return format(template, &variables, &lists)
}

func format(template string, variables *map[string]string, lists *map[string][]string) string {
	result := strings.Clone(template)
	for key, value := range *variables {
		result = strings.ReplaceAll(result, "$"+key+"$", value)
	}
	for key := range *lists {
		start := 0
		for start != -1 {
			oldStart := start
			start = strings.Index(result[oldStart:], "$*"+key+"*$")
			if start == -1 {
				break
			}
			start += oldStart
			lineStart := strings.LastIndex(result[:start], "\n")
			if lineStart == -1 {
				break
			}
			lineStart++
			lineEnd := strings.Index(result[start:], "\n")
			if lineEnd == -1 {
				break
			}
			lineEnd += start
			line := result[lineStart:lineEnd]

			var variablesInLine []string
			var listLength int
			for i := 0; i < len(line)-1; i++ {
				if line[i] == '$' && line[i+1] == '*' {
					variableStart := i + 2
					variableEnd := strings.Index(line[variableStart:], "*$")
					if variableEnd == -1 {
						continue
					}
					variableEnd += variableStart
					variableName := line[variableStart:variableEnd]
					variable, ok := (*lists)[variableName]
					if !ok {
						continue
					}
					if variableLen := len(variable); listLength == 0 || listLength > variableLen {
						listLength = variableLen
					}
					if contains(variablesInLine, variableName) {
						continue
					}
					variablesInLine = append(variablesInLine, variableName)
				}
			}

			var formattedLine string
			for i := 0; i < listLength; i++ {
				currentLine := strings.Clone(line)
				for j := 0; j < len(variablesInLine); j++ {
					currentLine = strings.ReplaceAll(currentLine, "$*"+variablesInLine[j]+"*$", (*lists)[variablesInLine[j]][i])
				}
				if i != 0 {
					formattedLine += "\n"
				}
				formattedLine += currentLine
			}
			result = strings.ReplaceAll(result, line, formattedLine)
		}
	}
	return result
}

func upper(str string) string {
	if len(str) == 0 {
		return str
	}
	return strings.ToUpper(str[:1]) + str[1:]
}

func lower(str string) string {
	if len(str) == 0 {
		return str
	}
	return strings.ToLower(str[:1]) + str[1:]
}

func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

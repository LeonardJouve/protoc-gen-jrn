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
	private static class Messages {
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
		(*lists)["messages"] = append((*lists)["messages"], generateMessage(message, ""))
	}
}

func generateMessage(message *protogen.Message, parent string) string {
	variables := make(map[string]string)
	lists := make(map[string][]string)

	variables["messageName"] = upper(string(message.Desc.Name()))
	variables["parent"] = parent

	var nestedMessages string
	for _, nestedMessage := range message.Messages {
		nestedMessages += generateMessage(nestedMessage, parent+variables["messageName"]+".")
	}
	variables["nestedMessages"] = nestedMessages

	for _, field := range message.Fields {
		fieldKind := field.Desc.Kind()
		fieldType, ok := kinds[fieldKind]
		if !ok {
			if fieldKind == protoreflect.MessageKind {
				lists["inputFieldTypes"] = append(lists["inputFieldTypes"], "Map")
				lists["outputFieldTypes"] = append(lists["outputFieldTypes"], "Map")
				lists["fieldNamesLower"] = append(lists["fieldNamesLower"], lower(string(field.Desc.Name())))
				lists["fieldNamesUpper"] = append(lists["fieldNamesUpper"], upper(string(field.Desc.Name())))
				lists["setPrefixes"] = append(lists["setPrefixes"], "Messages.set"+upper(string(field.Message.Desc.Name()))+"(")
				lists["setSuffixes"] = append(lists["setSuffixes"], ")")
				lists["getPrefixes"] = append(lists["getPrefixes"], "Messages.get"+upper(string(field.Message.Desc.Name()))+"(")
				lists["getSuffixes"] = append(lists["getSuffixes"], ")")
			}
			continue
		}
		lists["inputFieldTypes"] = append(lists["inputFieldTypes"], fieldType)
		lists["outputFieldTypes"] = append(lists["outputFieldTypes"], fieldType)
		lists["fieldNamesLower"] = append(lists["fieldNamesLower"], lower(string(field.Desc.Name())))
		lists["fieldNamesUpper"] = append(lists["fieldNamesUpper"], upper(string(field.Desc.Name())))
		lists["setPrefixes"] = append(lists["setPrefixes"], "")
		lists["setSuffixes"] = append(lists["setSuffixes"], "")
		lists["getPrefixes"] = append(lists["getPrefixes"], "")
		lists["getSuffixes"] = append(lists["getSuffixes"], "")
	}

	template := `private static $parent$$messageName$ set$messageName$(ReadableMap message) {
			return $parent$$messageName$.newBuilder()
				.set$*fieldNamesUpper*$($*setPrefixes*$message.get$*inputFieldTypes*$("$*fieldNamesLower*$")$*setSuffixes*$)
				.build();
		}
		private static WritableMap get$messageName$($parent$$messageName$ message) {
			WritableMap writableMap = Arguments.createMap();
			writableMap.put$*outputFieldTypes*$("$*fieldNamesLower*$", $*getPrefixes*$message.get$*fieldNamesUpper*$()$*getSuffixes*$);
			return writableMap;
		}`
	if len(nestedMessages) > 0 {
		template += `
		$nestedMessages$`
	}

	return format(template, &variables, &lists)
}

func formatVariables(template string, variables *map[string]string) string {
	result := template
	for key, value := range *variables {
		result = strings.ReplaceAll(result, "$"+key+"$", value)
	}
	return result
}

func formatLists(template string, lists *map[string][]string) string {
	lines := strings.Split(template, "\n")
	resultLines := strings.Builder{}

	for i, line := range lines {
		var start int
		lineAmount := -1
		lineVariables := make(map[string][]string)
		for start != -1 {
			oldStart := start
			start = strings.Index(line[start:], "$*")
			if start == -1 {
				break
			}
			start += oldStart

			end := strings.Index(line[start:], "*$")
			if end == -1 {
				break
			}
			end += start

			name := line[start+2 : end]
			if variable, ok := (*lists)[name]; ok {
				if variableLen := len(variable); lineAmount == -1 || lineAmount > variableLen {
					lineAmount = variableLen
				}
				lineVariables[name] = variable
			}

			start = end + 2
		}

		if lineAmount == -1 {
			lineAmount = 1
		}
		for j := 0; j < lineAmount; j++ {
			currentLine := line
			for name, value := range lineVariables {
				currentLine = strings.ReplaceAll(currentLine, "$*"+name+"*$", value[j])
			}
			if i > 0 {
				resultLines.WriteByte('\n')
			}
			resultLines.WriteString(currentLine)
		}
	}

	return resultLines.String()
}

func format(template string, variables *map[string]string, lists *map[string][]string) string {
	result := formatVariables(template, variables)
	result = formatLists(result, lists)
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

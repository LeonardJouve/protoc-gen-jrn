package main

import (
	"strings"

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

func generate(plugin *protogen.Plugin, in *protogen.File) error {
	fileName := *in.Proto.Options.JavaOuterClassname + ".java"
	out := plugin.NewGeneratedFile(fileName, protogen.GoImportPath(upper(string(in.GoImportPath))))
	out.P("package ", *in.Proto.Options.JavaPackage)

	generateModule(out)

	variables := make(map[string]string)
	lists := make(map[string][]string)

	for _, service := range in.Services {
		variables["serviceName"] = upper(string(service.Desc.Name()))

		for _, method := range service.Methods {
			err := generateMethod(method, &variables, &lists, out)
			if err != nil {
				return err
			}
		}
	}

	out.P("}")

	return nil
}

func generateModule(out *protogen.GeneratedFile) error {
	out.P(`
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
    }`)
	return nil
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
			for i := 0; i < len(line); i++ {
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

func generateMethod(method *protogen.Method, variables *map[string]string, lists *map[string][]string, out *protogen.GeneratedFile) error {
	(*variables)["methodName"] = lower(string(method.Desc.Name()))
	(*variables)["inputKind"] = upper(string(method.Input.Desc.Name()))
	(*variables)["outputKind"] = upper(string(method.Output.Desc.Name()))

	for _, field := range method.Input.Fields {
		(*lists)["inputFieldTypes"] = append((*lists)["inputFieldTypes"], upper(field.Desc.Kind().String())) // TODO: java type
		(*lists)["inputFieldNames"] = append((*lists)["inputFieldNames"], upper(string(field.Desc.Name())))
	}

	for _, field := range method.Output.Fields {
		(*lists)["outputFieldTypes"] = append((*lists)["outputFieldTypes"], upper(field.Desc.Kind().String())) // TODO: java type
		(*lists)["outputFieldNames"] = append((*lists)["outputFieldNames"], upper(string(field.Desc.Name())))
	}

	const template = `
	@ReactMethod()
	public void $methodName$(ReadableMap message, Promise promise) {
		try {
			checkHostAndPort();
			ManagedChannel channel = ManagedChannelBuilder.forAddress(host, port).usePlaintext().build();
			$serviceName$Grpc.$serviceName$BlockingStub stub = $serviceName$Grpc.newBlockingStub(channel);
			$inputKind$ request = $inputKind$.newBuilder()
				.set$*inputFieldNames*$(message.get$*inputFieldTypes*$("$*inputFieldNames*$"))
				.build();

			$outputKind$ response = stub.$methodName$(request);
			WritableMap result = Arguments.createMap();
			result.put$*outputFieldTypes*$("$*outputFieldNames*$", response.get$*outputFieldNames*$());

			promise.resolve(result);
		} catch (Exception e) {
			promise.reject("Error", "Unable to call remote procedure \"$methodName$\"", e);
		}
	}`

	out.P(format(template, variables, lists))

	return nil
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

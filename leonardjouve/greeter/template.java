package com.grpc;

import com.facebook.react.bridge.Arguments;
import com.facebook.react.bridge.Promise;
import com.facebook.react.bridge.ReactApplicationContext;
import com.facebook.react.bridge.ReactContextBaseJavaModule;
import com.facebook.react.bridge.ReactMethod;
import com.facebook.react.bridge.ReadableMap;
import com.facebook.react.bridge.WritableMap;

import java.util.HashMap;

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

    @ReactMethod()
    public void greet(ReadableMap message, Promise promise) {
        try {
            checkHostAndPort();
            ManagedChannel channel = ManagedChannelBuilder.forAddress(host, port).usePlaintext().build();
            GreeterGrpc.GreeterBlockingStub stub = GreeterGrpc.newBlockingStub(channel);
            HelloRequest request = HelloRequest.newBuilder()
                .setName(message.getString("name"))
                .build();

            HelloResponse response = stub.greet(request);
            WritableMap result = Arguments.createMap();
            result.putString("greetings", response.getGreetings());
            promise.resolve(result);
        } catch (Exception e) {
            promise.reject("Error", "Unable to call remote procedure \"greet\"", e);
        }
    }
}

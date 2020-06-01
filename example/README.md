## Generate the files

In order to edit the generated .proto.go files

	$ protoc -I example/ example/pb/helloworld.proto --go_out=plugins=grpc:example

## Start the gRPC backend

	$ go build -o backend ./example/backend
	$ ./backend

## KrakenD setup

### As custom bindings

Check the `example/krakend/grpc_bindings.go` for examples about how to build the Encoder/Decoder pair, register a client and so on.

With your bindings already defined, just execute the Register

		registerer := new(Registerer)
		registerer.RegisterClients(grpc.RegisterClient)

Build and start the example

	$ go build -o krakend ./example/krakend
	$ ./krakend -c ./example/krakend/krakend.json -d -l DEBUG 

### As a plugin (dynamic linked bin)

Define your gRPC bindings in a dedicated package, ready to be compiled as a golang plugin (`main` package with and empty `main` function)

The plugin loader will lookup a symbol called "Registerer". This symbol must implement the interface `plugin.Registerer` in order to be loaded properly

As you can see, the `example/krakend/plugin/plugin.go` is almost a copy of `example/krakend/grpc_bindings.go`.

Build the plugin and where you put the .so file

	$ go build -buildmode=plugin -o gRPC-hello.so

Add a `plugin` section into the service config file with the path to the folder containing the .so and a pattern to be requested

	{
		...
		"plugin":{
			"folder": "./plugins",
			"pattern": ".so"
		},
		...
	}

Build and start the example

	$ go build -o krakend ./example/krakend
	$ ./krakend -c ./example/krakend/krakend.json -d -l DEBUG -plugin
	[KRAKEND] DEBUG: gRPC: total loaded plugins = 0

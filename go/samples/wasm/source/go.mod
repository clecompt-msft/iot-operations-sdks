module github.com/Azure/iot-operations-sdks/go/samples/wasm/source

go 1.24.0

require github.com/Azure/iot-operations-sdks/go/wasm v0.0.0

require (
	go.bytecodealliance.org/cm v0.2.2 // indirect
	tinygo.org/x/drivers v0.31.0 // indirect
)

replace github.com/Azure/iot-operations-sdks/go/wasm v0.0.0 => ../../../wasm

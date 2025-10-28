# zencodeio

Stream-based encoding and decoding for Go data structures without inline error handling.

## Overview

`zencodeio` provides a clean abstraction for un/marshalling data structures directly from to/from streams, rather than reading and writing manually and then un/marshalling.

## Core Concepts

- **Encoder**: Acts as a `io.Reader` that produces the marshalled data structure
- **Decoder**: Acts as an `io.WriteCloser` that unmarshals the written data into a data structure
- **Read**: Reads everything from a `io.Reader` into a `Decoder`, and then closes the `Decoder`. Effectively marshalling the contents of the `Reader` into the data structure within the `Decoder`.

## Usage

### Encoding data structures

Write data structures directly to streams without manual marshalling:

```go
data := MyStruct{Name: "test", Value: 42}

// Write the JSON representation of a data structure directly to any io.Writer
io.Copy(w, zencodeio.NewJSONEncoder(data))
```

### Decoding data structures

Read from streams directly into data structures without manual unmarshalling:

```go
var result MyStruct

// Read JSON from any io.Reader (http request, file, connection, etc.) into a data structure
err := zencodeio.Read(r, zencodeio.NewJSONDecoder(&result))
```

### Custom formats

```go
// Custom encoder
data := "custom data"
encoder := zencodeio.NewMarshallerEncoder(data, func(v any) ([]byte, error) {
    return []byte(fmt.Sprintf("CUSTOM:%v", v)), nil
})
io.Copy(writer, encoder)

// Custom decoder
var result string
decoder := zencodeio.NewMarshallerDecoder(&result, "custom/type",
    func(data []byte, v any) error {
        ptr := v.(*string)
        *ptr = string(data)
        return nil
    },
)
err := zencodeio.Read(reader, decoder)
```


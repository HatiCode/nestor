package json

import jsoniter "github.com/json-iterator/go"

var (
	json       = jsoniter.ConfigCompatibleWithStandardLibrary
	jsonFast   = jsoniter.ConfigFastest
	jsonNestor = jsoniter.Config{
		EscapeHTML:             false,
		SortMapKeys:            false,
		ValidateJsonRawMessage: true,
		UseNumber:              false,
		DisallowUnknownFields:  false,
	}.Froze()
)

// Performance comparison functions (for benchmarking)

func MarshalStandard(v any) ([]byte, error) {
	return json.Marshal(v)
}

func MarshalFast(v any) ([]byte, error) {
	return jsonFast.Marshal(v)
}

func MarshalNestor(v any) ([]byte, error) {
	return jsonNestor.Marshal(v)
}

func UnmarshalStandard(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func UnmarshalFast(data []byte, v any) error {
	return jsonFast.Unmarshal(data, v)
}

func UnmarshalNestor(data []byte, v any) error {
	return jsonNestor.Unmarshal(data, v)
}

// Convenience methods for common operations

func ToJSON(v any) ([]byte, error) {
	return jsonNestor.Marshal(v)
}

func ToJSONString(v any) (string, error) {
	return jsonNestor.MarshalToString(v)
}

func FromJSON(data []byte, v any) error {
	return jsonNestor.Unmarshal(data, v)
}

func FromJSONString(data string, v any) error {
	return jsonNestor.UnmarshalFromString(data, v)
}

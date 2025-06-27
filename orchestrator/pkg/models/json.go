package models

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

func MarshalStandard(v any) ([]byte, error) {
	return json.Marshal(v)
}

func MarshalFast(v any) ([]byte, error) {
	return jsonFast.Marshal(v)
}

func MarshalNestor(v any) ([]byte, error) {
	return jsonNestor.Marshal(v)
}

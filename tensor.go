package main

import (
	"fmt"
	"reflect"
	tff "tensorflow/core/framework"
)

/**
 * Utility functions to deal with TensorProto
 */

func InferTensorFromValue(value interface{}) (tensor *tff.TensorProto) {
	tensor = &tff.TensorProto{
		TensorShape: &tff.TensorShapeProto{
			Dim: []*tff.TensorShapeProto_Dim{},
		},
	}

	convertToTensor(tensor, value, 0)

	return
}

func convertToTensor(tensor *tff.TensorProto, value interface{}, level int) {
	valueType := reflect.TypeOf(value)

	if valueType.Kind() == reflect.Slice || valueType.Kind() == reflect.Array {
		s := reflect.ValueOf(value)

		if s.Len() == 0 {
			panic("Slice can not be empty")
		}

		// extend tensor shape
		if len(tensor.TensorShape.Dim) <= level {
			tensor.TensorShape.Dim = append(
				tensor.TensorShape.Dim,
				&tff.TensorShapeProto_Dim{
					Size: int64(s.Len()),
					Name: "",
				},
			)
		}

		elemValue := reflect.ValueOf(s.Index(0).Interface())

		// TODO: support Array
		if elemValue.Kind() == reflect.Slice || elemValue.Kind() == reflect.Array {
			for i := 0; i < s.Len(); i++ {
				convertToTensor(tensor, s.Index(i).Interface(), level+1)
			}
		} else {
			copyTensorData(tensor, value, s)
		}
	} else {
		panic("value must be a slice")
	}
}

func copyTensorData(tensor *tff.TensorProto, value interface{}, s reflect.Value) {
	var idx int

	// peek value type
	switch s.Index(0).Interface().(type) {
	case float32, float64:
		tensor.Dtype = tff.DataType_DT_FLOAT
		idx = len(tensor.FloatVal)

		tmp := make([]float32, idx+s.Len())
		copy(tmp, tensor.FloatVal)
		tensor.FloatVal = tmp
	case int, int32, int64:
		tensor.Dtype = tff.DataType_DT_INT32
		idx = len(tensor.IntVal)

		tmp := make([]int32, idx+s.Len())
		copy(tmp, tensor.IntVal)
		tensor.IntVal = tmp
	}

	// TODO: support float 64
	for i := 0; i < s.Len(); i++ {
		iv := s.Index(i).Interface()

		switch iv.(type) {
		case float32:
			tensor.FloatVal[idx+i] = iv.(float32)
		case float64:
			tensor.FloatVal[idx+i] = float32(iv.(float64))
		case int:
			tensor.IntVal[idx+i] = int32(iv.(int))
		case int32:
			tensor.IntVal[idx+i] = iv.(int32)
		default:
			panic(fmt.Sprintf("Unknown type %T", iv))
		}
	}
}

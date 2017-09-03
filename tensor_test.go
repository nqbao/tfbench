package main

import (
	"reflect"
	tff "tensorflow/core/framework"
	"testing"
)

type convertTest struct {
	input         interface{}
	expectedShape []int64
	expectedValue interface{}
}

func getShape(tensor *tff.TensorProto) (shape []int64) {
	shape = make([]int64, len(tensor.TensorShape.Dim))

	for i := range tensor.TensorShape.Dim {
		dim := tensor.TensorShape.Dim[i]
		shape[i] = dim.Size
	}

	return
}

func getRawValue(tensor *tff.TensorProto) (value interface{}) {
	switch tensor.Dtype {
	case tff.DataType_DT_FLOAT:
		value = tensor.FloatVal
	case tff.DataType_DT_INT32:
		value = tensor.IntVal
	}

	return
}

func TestConvertSliceToTensor(t *testing.T) {
	tests := []convertTest{
		convertTest{
			[]float32{1.0, 2.0},
			[]int64{2},
			[]float32{1.0, 2.0},
		},
		convertTest{
			[][]float32{
				[]float32{1.0, 2.0},
				[]float32{3.0, 4.0},
				[]float32{5.0, 6.0},
			},
			[]int64{3, 2},
			[]float32{1.0, 2.0, 3.0, 4.0, 5.0, 6.0},
		},
		// Test should work with array as well
		convertTest{
			[2]float32{1.0, 2.0},
			[]int64{2},
			nil,
		},
		// Test should work with int type
		convertTest{
			[3]int{1, 2, 3},
			[]int64{3},
			[]int32{1, 2, 3},
		},
	}

	for i := range tests {
		test := tests[i]

		tensor := InferTensorFromValue(test.input)

		if !reflect.DeepEqual(getShape(tensor), test.expectedShape) {
			t.Errorf("Case %v fails: expected shape is %v", i, test.expectedShape)
		}

		if test.expectedValue != nil {
			raw := getRawValue(tensor)
			if !reflect.DeepEqual(raw, test.expectedValue) {
				t.Errorf("Case %v fails: expect %v, got %v", i, test.expectedValue, raw)
			}
		}
	}
}

package main

import (
	"encoding/json"
	_ "fmt"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"io/ioutil"
	"reflect"
	tff "tensorflow/core/framework"
	pb "tensorflow_serving/apis"
)

func LoadRequestFromJson(modelName string, fileName string) (*pb.PredictRequest, error) {
	inputsMap, err := readJson(fileName)

	if err != nil {
		return nil, err
	}

	inputs := make(map[string]*tff.TensorProto)
	v := reflect.ValueOf(inputsMap)

	for _, k := range v.MapKeys() {
		inputs[k.Interface().(string)] = InferTensorFromValue(v.MapIndex(k).Interface())
	}

	return BuildRequest(modelName, inputs), nil
}

func readJson(fileName string) (interface{}, error) {
	raw, err := ioutil.ReadFile(fileName)

	if err != nil {
		return nil, err
	}

	var result interface{}

	jsonErr := json.Unmarshal(raw, &result)

	if jsonErr != nil {
		return nil, jsonErr
	}

	return result, nil
}

func BuildRequest(modelName string, inputs map[string]*tff.TensorProto) (request *pb.PredictRequest) {
	request = &pb.PredictRequest{
		ModelSpec: &pb.ModelSpec{
			Name: modelName,
		},
		Inputs: inputs,
	}

	return
}

func SendRequest(addr string, request *pb.PredictRequest) (*pb.PredictResponse, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())

	if err != nil {
		return nil, err
	}

	defer conn.Close()

	return SendRequestToClient(conn, request)
}

func SendRequestToClient(conn *grpc.ClientConn, request *pb.PredictRequest) (*pb.PredictResponse, error) {
	// TODO: add timeout here
	client := pb.NewPredictionServiceClient(conn)
	return client.Predict(context.Background(), request)
}

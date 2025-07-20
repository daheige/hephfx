// GENERATED CODE -- DO NOT EDIT!

'use strict';
var grpc = require('@grpc/grpc-js');
var hello_pb = require('./hello_pb.js');
// var google_api_annotations_pb = require('./google/api/annotations_pb.js');

function serialize_Hello_HelloReply(arg) {
  if (!(arg instanceof hello_pb.HelloReply)) {
    throw new Error('Expected argument of type Hello.HelloReply');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_Hello_HelloReply(buffer_arg) {
  return hello_pb.HelloReply.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_Hello_HelloReq(arg) {
  if (!(arg instanceof hello_pb.HelloReq)) {
    throw new Error('Expected argument of type Hello.HelloReq');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_Hello_HelloReq(buffer_arg) {
  return hello_pb.HelloReq.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_Hello_InfoReply(arg) {
  if (!(arg instanceof hello_pb.InfoReply)) {
    throw new Error('Expected argument of type Hello.InfoReply');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_Hello_InfoReply(buffer_arg) {
  return hello_pb.InfoReply.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_Hello_InfoReq(arg) {
  if (!(arg instanceof hello_pb.InfoReq)) {
    throw new Error('Expected argument of type Hello.InfoReq');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_Hello_InfoReq(buffer_arg) {
  return hello_pb.InfoReq.deserializeBinary(new Uint8Array(buffer_arg));
}


// Greeter service 定义开放调用的服务
var GreeterService = exports.GreeterService = {
  sayHello: {
    path: '/Hello.Greeter/SayHello',
    requestStream: false,
    responseStream: false,
    requestType: hello_pb.HelloReq,
    responseType: hello_pb.HelloReply,
    requestSerialize: serialize_Hello_HelloReq,
    requestDeserialize: deserialize_Hello_HelloReq,
    responseSerialize: serialize_Hello_HelloReply,
    responseDeserialize: deserialize_Hello_HelloReply,
  },
  info: {
    path: '/Hello.Greeter/Info',
    requestStream: false,
    responseStream: false,
    requestType: hello_pb.InfoReq,
    responseType: hello_pb.InfoReply,
    requestSerialize: serialize_Hello_InfoReq,
    requestDeserialize: deserialize_Hello_InfoReq,
    responseSerialize: serialize_Hello_InfoReply,
    responseDeserialize: deserialize_Hello_InfoReply,
  },
};

exports.GreeterClient = grpc.makeGenericClientConstructor(GreeterService, 'Greeter');

package bandersnatchErrors

var _ BatchSerializationError = &batchSerializationError{}
var _ BatchSerializationError = &sliceSerializationError{}
var _ BatchDeserializationError = &batchDeserializationError{}
var _ BatchDeserializationError = &sliceDeserializationError{}

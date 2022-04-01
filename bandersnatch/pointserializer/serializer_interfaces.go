package pointserializer

import (
	"io"

	. "github.com/GottfriedHerold/Bandersnatch/bandersnatch"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/bandersnatchErrors"
)


/*
const (
	basicSerializerCloneFun            = "Clone"
	basicSerializerNewEndiannessFun    = "WithEndianness"
	basicSerializerNewSubgroupRestrict = "WithSubgroupOnly"
)
*/

type CurvePointDeserializer interface {
	curvePointDeserializer_basic // TODO: Copy definition for godoc
	DeserializePoints(inputStream io.Reader, outputPoints CurvePointSlice) (bytesRead int, err bandersnatchErrors.BatchSerializationError)
	DeserializeBatch(inputStream io.Reader, outputPoints ...CurvePointPtrInterfaceWrite) (bytesRead int, err bandersnatchErrors.BatchSerializationError)

	// Matches SerializeSlice
	DeserializeSlice(inputStream io.Reader) (outputPoints CurvePointSlice, bytesRead int, err bandersnatchErrors.BatchSerializationError)
	DeserializeSliceToBuffer(inputStream io.Reader, outputPoints CurvePointSlice) (bytesRead int, pointsRead int, err bandersnatchErrors.BatchSerializationError)
}

type CurvePointSerializer interface {
	CurvePointDeserializer
	curvePointSerializer_basic
	SerializePoints(outputStream io.Writer, inputPoints CurvePointSlice) (bytesWritten int, err bandersnatchErrors.BatchSerializationError) // SerializeBatch(os, points) is equivalent (if no error occurs) to calling Serialize(os, point[i]) for all i. NOTE: This provides the same functionality as SerializePoints, but with a different argument type.
	SerializeBatch(outputStream io.Writer, inputPoints ...CurvePointPtrInterfaceRead) (bytesWritten int, err error)                         // SerializePoints(os, &x1, &x2, ...) is equivalent (if not error occurs, at least) to Serialize(os, &x1), Serialize(os, &x1), ... NOTE: Using SerializePoints(os, points...) with ...-notation might not work due to the need to convert []concrete Point type to []CurvePointPtrInterface. Use SerializeBatch to avoid this.
	SerializeSlice(outputStream io.Writer, inputSlice CurvePointSlice) (bytesWritten int, err bandersnatchErrors.BatchSerializationError)   // SerializeSlice(os, points) serializes a slice of points to outputStream. As opposed to SerializeBatch and SerializePoints, the number of points written is stored in the output stream and can NOT be read back individually, but only by DeserializeSlice
}

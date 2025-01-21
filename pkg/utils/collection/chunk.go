package collection

import "math"

// Chunk divides []string into chunks of size.
func Chunk[T string | *string](collection []T, size int) [][]T {
	if len(collection) == 0 || size <= 0 {
		return nil
	}

	chunkNum := int(math.Ceil(float64(len(collection)) / float64(size)))
	res := make([][]T, 0, chunkNum)
	for i := 0; i < chunkNum-1; i++ {
		res = append(res, collection[i*size:(i+1)*size])
	}
	res = append(res, collection[(chunkNum-1)*size:])
	return res
}

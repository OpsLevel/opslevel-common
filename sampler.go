package opslevel_common

import (
	"math/rand"
)

func GetSample[T any](sampleCount int, data []T) []T {
	if sampleCount < 1 || sampleCount > len(data) {
		return data
	}
	rand.Shuffle(len(data), func(i, j int) {
		data[i], data[j] = data[j], data[i]
	})
	return data[:sampleCount]
}

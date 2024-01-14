package opslevel_common

import (
	"math/rand"
	"sort"
	"time"
)

func getSamples(start int, end int, count int) []int {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	if end < start || (end-start) < count {
		return nil
	}
	nums := make([]int, 0)
	for len(nums) < count {
		num := rand.Intn(end-start) + start
		exist := false
		for _, v := range nums {
			if v == num {
				exist = true
				break
			}
		}
		if !exist {
			nums = append(nums, num)
		}
	}
	sort.Ints(nums)
	return nums
}

func GetSample[T any](sampleCount int, data []T) []T {
	if sampleCount < 1 {
		return data
	}
	totalItems := len(data)
	if sampleCount >= totalItems {
		return data
	}
	output := make([]T, sampleCount)
	for i, index := range getSamples(0, totalItems, sampleCount) {
		output[i] = data[index]
	}
	return output
}

// GetSample returns a random selection of N items in the slice in the original order.
// The elements are copied using assignment, so this is a shallow clone.
// func GetSample[T any](sampleCount int, data []T) []T {
// 	var (
// 		keys = make([]int, len(data))
// 		copy []T
// 	)
// 	if sampleCount < 1 || sampleCount >= len(data) {
// 		return slices.Clone(data)
// 	}
// 	for i := range keys {
// 		keys[i] = i
// 	}
// 	rand.Shuffle(len(keys), func(i, j int) {
// 		keys[i], keys[j] = keys[j], keys[i]
// 	})
// 	keys = keys[:sampleCount]
// 	slices.Sort(keys)
// 	copy = make([]T, sampleCount)
// 	for i := range keys {
// 		copy[i] = data[keys[i]]
// 	}
// 	return copy
// }

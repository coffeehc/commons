package utils

func IntsToMap(ids []int64) map[int64]struct{} {
	m := make(map[int64]struct{})
	for _, id := range ids {
		m[id] = struct{}{}
	}
	return m
}

func EqInts(src []int64, target []int64) ([]int64, []int64) {
	srcMap := make(map[int64]struct{}, len(src))
	targetmap := make(map[int64]struct{}, len(target))
	for _, i := range src {
		srcMap[i] = struct{}{}
	}
	for _, i := range target {
		targetmap[i] = struct{}{}
	}
	srcDiff := make([]int64, 0, len(src))
	targetDiff := make([]int64, 0, len(src))
	for _, i := range src {
		_, ok := targetmap[i]
		if !ok {
			srcDiff = append(srcDiff, i)
		}
	}
	for _, i := range target {
		_, ok := srcMap[i]
		if !ok {
			targetDiff = append(targetDiff, i)
		}
	}
	return srcDiff, targetDiff
}

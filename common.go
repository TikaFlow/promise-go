package promise

import "math/rand"

// randString 生成一个以 "@" 开头的指定长度随机字符串，用于钩子唯一标识。
func randString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return "@" + string(result)
}

// deleteFromSlice 从 slice 中删除内容为 key 的元素（仅首个匹配）。
func deleteFromSlice(slice *[]string, key string) bool {
	target := false
	for i, k := range *slice {
		if k == key {
			target = true
			*slice = append((*slice)[:i], (*slice)[i+1:]...)
			break
		}
	}
	return target
}

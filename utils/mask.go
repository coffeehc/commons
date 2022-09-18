package utils

var masker = []rune("****")

func MaskMobile(mobile string) string {
	if mobile == "" {
		return mobile
	}
	mobileRune := []rune(mobile)
	copy(mobileRune[3:7], masker)
	return string(mobileRune)
}

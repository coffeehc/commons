package utils

import (
	"regexp"
	"strconv"
)

var _VerifyMobile = regexp.MustCompile(`^1[3456789]\d{9}$`)
var _VerifyUsername = regexp.MustCompile(`^[a-zA-Z0-9\._@-]{6,32}$`)
var _VerifyPassword = regexp.MustCompile(`^[.*[A-Za-z]][.*\d[.*[$@$!%*#?&]][A-Za-z\d$@$!%*#?&]{6,20}$`)

var _VerifyEmail = regexp.MustCompile(`^[a-zA-Z0-9_-]+@[a-zA-Z0-9_-]+(\.[a-zA-Z0-9_-]+)+$`)

func VerifyEmail(email string) bool {
	return _VerifyEmail.MatchString(email)
}
func VerifyMobild(mobile string) bool {
	return _VerifyMobile.MatchString(mobile)
}

func VerifyIdCard(idCard string) bool {
	return len(idCard) == 18 && string(idCard[17]) == getIdCardVerifyId(idCard)
}

func VerifyUsername(username string) bool {
	return _VerifyUsername.MatchString(username)
}

func VerifyPassword(password string) bool {
	return _VerifyPassword.MatchString(password)
}

var (
	wi = [17]int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
	yi = []string{`1`, `0`, `x`, `9`, `8`, `7`, `6`, `5`, `4`, `3`, `2`}
)

func getIdCardVerifyId(id string) string { // len(id)= 17
	arry := make([]int, 17)
	for i := 0; i < 17; i++ {
		arry[i], _ = strconv.Atoi(string(id[i]))
	}
	var res int
	for i := 0; i < 17; i++ {
		res += arry[i] * wi[i]
	}
	return yi[(res % 11)]
}

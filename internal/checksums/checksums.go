package checksums

func Luhn(number int64) bool {
	var result int64

	for i := 0; number > 0; i++ {
		cur := number % 10

		if (i+1)%2 == 0 {
			cur = cur * 2
			if cur > 9 {
				cur = cur%10 + cur/10
			}
		}

		result += cur
		number = number / 10
	}
	return result%10 == 0
}

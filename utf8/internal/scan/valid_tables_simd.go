//go:build goexperiment.simd

package scan

// These tables describe UTF-8 byte-pair validity independently of vector
// width. Architecture-specific predicates load them in the shape appropriate
// for their lane layout.
const (
	utf8TooShort         = 1 << 0
	utf8TooLong          = 1 << 1
	utf8Overlong3        = 1 << 2
	utf8TooLarge         = 1 << 3
	utf8Surrogate        = 1 << 4
	utf8Overlong2        = 1 << 5
	utf8TooLarge1000     = 1 << 6
	utf8TwoContinuations = 1 << 7

	utf8Carry = utf8TooShort | utf8TooLong | utf8TwoContinuations
)

var (
	utf8IncompleteThresholds = [16]uint8{
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xef, 0xdf, 0xbf,
	}

	utf8SpecialPrevHighTable = [16]uint8{
		utf8TooLong, utf8TooLong, utf8TooLong, utf8TooLong,
		utf8TooLong, utf8TooLong, utf8TooLong, utf8TooLong,
		utf8TwoContinuations, utf8TwoContinuations, utf8TwoContinuations, utf8TwoContinuations,
		utf8TooShort | utf8Overlong2,
		utf8TooShort,
		utf8TooShort | utf8Overlong3 | utf8Surrogate,
		utf8TooShort | utf8TooLarge | utf8TooLarge1000,
	}
	utf8SpecialPrevLowTable = [16]uint8{
		utf8Carry | utf8Overlong3 | utf8Overlong2 | utf8TooLarge1000,
		utf8Carry | utf8Overlong2,
		utf8Carry, utf8Carry,
		utf8Carry | utf8TooLarge,
		utf8Carry | utf8TooLarge | utf8TooLarge1000,
		utf8Carry | utf8TooLarge | utf8TooLarge1000,
		utf8Carry | utf8TooLarge | utf8TooLarge1000,
		utf8Carry | utf8TooLarge | utf8TooLarge1000,
		utf8Carry | utf8TooLarge | utf8TooLarge1000,
		utf8Carry | utf8TooLarge | utf8TooLarge1000,
		utf8Carry | utf8TooLarge | utf8TooLarge1000,
		utf8Carry | utf8TooLarge | utf8TooLarge1000,
		utf8Carry | utf8TooLarge | utf8TooLarge1000 | utf8Surrogate,
		utf8Carry | utf8TooLarge | utf8TooLarge1000,
		utf8Carry | utf8TooLarge | utf8TooLarge1000,
	}
	utf8SpecialCurrentHighTable = [16]uint8{
		utf8TooShort, utf8TooShort, utf8TooShort, utf8TooShort,
		utf8TooShort, utf8TooShort, utf8TooShort, utf8TooShort,
		utf8TooLong | utf8Overlong2 | utf8TwoContinuations | utf8Overlong3 | utf8TooLarge1000,
		utf8TooLong | utf8Overlong2 | utf8TwoContinuations | utf8Overlong3 | utf8TooLarge,
		utf8TooLong | utf8Overlong2 | utf8TwoContinuations | utf8Surrogate | utf8TooLarge,
		utf8TooLong | utf8Overlong2 | utf8TwoContinuations | utf8Surrogate | utf8TooLarge,
		utf8TooShort, utf8TooShort, utf8TooShort, utf8TooShort,
	}
)

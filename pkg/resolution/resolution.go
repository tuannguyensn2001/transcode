package resolution

type Resolution int

const (
	R1080 Resolution = 1080
	R720  Resolution = 720
	R480  Resolution = 480
	R360  Resolution = 360
)

func (r Resolution) Level() int {
	switch r {
	case R360:
		return 1
	case R480:
		return 2
	case R720:
		return 3
	case R1080:
		return 4
	default:
		return 0
	}
}

func FromLevel(level int) Resolution {
	switch level {
	case 1:
		return R360
	case 2:
		return R480
	case 3:
		return R720
	case 4:
		return R1080
	default:
		return 0
	}
}

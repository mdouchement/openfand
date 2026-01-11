package openfand

import "io"

func ReadSSE(r io.Reader) ([]byte, error) {
	buf := make([]byte, 512<<10) // 512kB is far enough to read a SSE from openfand.

	var n int
	var lf uint8
	var err error
	for {
		_, err = r.Read(buf[n : n+1])
		if err != nil {
			return buf[:n], err
		}

		if buf[n] == '\n' {
			lf++
		} else {
			lf = 0
		}

		if lf == 2 {
			return buf[:n-1], nil
		}

		n++
	}
}

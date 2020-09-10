package securecomm

import (
	"encoding/binary"
)

func toByteArray(i int64) (arr [8]byte) {
	binary.BigEndian.PutUint64(arr[0:8], uint64(i))
	return
}

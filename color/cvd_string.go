// Code generated by "stringer -type=CVD"; DO NOT EDIT.

package color

import "strconv"

const _CVD_name = "NoneRedGreenBlue"

var _CVD_index = [...]uint8{0, 4, 12, 16}

func (i CVD) String() string {
	if i >= CVD(len(_CVD_index)-1) {
		return "CVD(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _CVD_name[_CVD_index[i]:_CVD_index[i+1]]
}

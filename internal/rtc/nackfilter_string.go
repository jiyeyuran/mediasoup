// Code generated by "stringer -type=NackFilter"; DO NOT EDIT.

package rtc

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[NackFilterSeq-0]
	_ = x[NackFilterTime-1]
}

const _NackFilter_name = "NackFilterSeqNackFilterTime"

var _NackFilter_index = [...]uint8{0, 13, 27}

func (i NackFilter) String() string {
	if i < 0 || i >= NackFilter(len(_NackFilter_index)-1) {
		return "NackFilter(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _NackFilter_name[_NackFilter_index[i]:_NackFilter_index[i+1]]
}
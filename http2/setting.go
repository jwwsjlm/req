package http2

import (
	"fmt"
)

// A SettingID is an HTTP/2 setting as defined in
// https://httpwg.org/specs/rfc7540.html#iana-settings
type SettingID uint16

const (
	SettingHeaderTableSize      SettingID = 0x1
	SettingEnablePush           SettingID = 0x2
	SettingMaxConcurrentStreams SettingID = 0x3
	SettingInitialWindowSize    SettingID = 0x4
	SettingMaxFrameSize         SettingID = 0x5
	SettingMaxHeaderListSize    SettingID = 0x6
)

var settingName = map[SettingID]string{
	SettingHeaderTableSize:      "HEADER_TABLE_SIZE",
	SettingEnablePush:           "ENABLE_PUSH",
	SettingMaxConcurrentStreams: "MAX_CONCURRENT_STREAMS",
	SettingInitialWindowSize:    "INITIAL_WINDOW_SIZE",
	SettingMaxFrameSize:         "MAX_FRAME_SIZE",
	SettingMaxHeaderListSize:    "MAX_HEADER_LIST_SIZE",
}

// String returns the standard name of the setting ID or an UNKNOWN_SETTING value.
// String 返回设置 ID 的标准名称；未知 ID 返回 UNKNOWN_SETTING 形式的名称。
func (s SettingID) String() string {
	if v, ok := settingName[s]; ok {
		return v
	}
	return fmt.Sprintf("UNKNOWN_SETTING_%d", uint16(s))
}

// Setting is a setting parameter: which setting it is, and its value.
type Setting struct {
	// ID is which setting is being set.
	// See https://httpwg.org/specs/rfc7540.html#SettingValues
	ID SettingID

	// Val is the value.
	Val uint32
}

// String formats the setting as its ID and numeric value.
// String 将设置格式化为 ID 与数值组成的字符串。
func (s Setting) String() string {
	return fmt.Sprintf("[%v = %d]", s.ID, s.Val)
}

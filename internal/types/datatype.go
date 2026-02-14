package types

type DataType int

const (
	Text DataType = iota + 1
	Image
	Audio
	Video
)

// To make enum values human-readable stings implementing Stringer interface of fmt
// func (d DataType) String() string {
// 	return [...]string{"Text, Image, Audio, Videa"}[d]
// }

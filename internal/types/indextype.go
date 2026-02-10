package types

type IndexType int

const (
	LinearIndex IndexType = iota
	HNSWIndex
	IVFIndex
	PQIndex
)

// func (it IndexType) String() string {
// 	switch it {
// 	case LinearIndex:
// 		return "LinearIndex"
// 	case HNSWIndex:
// 		return "HNSWIndex"
// 	case IVFIndex:
// 		return "IVFIndex"
// 	case PQIndex:
// 		return "PQIndex"
// 	default:
// 		panic("unsupported Index Type")
// 	}
// }

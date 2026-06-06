package tidal

type Operation uint8

const (
	Unknown Operation = iota
	List
	Create
	Retrieve
	Update
	Delete
)

var (
	SelectOperations  = [2]Operation{List, Retrieve}
	EditOperations    = [3]Operation{Create, Update, Delete}
	PrepareOperations = [2]Operation{Create, Update}
)

func (o Operation) String() string {
	switch o {
	case List:
		return "List"
	case Create:
		return "Create"
	case Retrieve:
		return "Retrieve"
	case Update:
		return "Update"
	case Delete:
		return "Delete"
	default:
		return "Unknown"
	}
}

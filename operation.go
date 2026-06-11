package tidal

// Operation identifies which CRUD step a [Model] method is serving.
type Operation uint8

const (
	Unknown Operation = iota
	List
	Create
	Retrieve
	Update
	Delete
)

// SelectOperations lists read-only [Operation] values.
var SelectOperations = [2]Operation{List, Retrieve}

// EditOperations lists write [Operation] values.
var EditOperations = [3]Operation{Create, Update, Delete}

// PrepareOperations lists [Operation] values that call [Preparer].
var PrepareOperations = [2]Operation{Create, Update}

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

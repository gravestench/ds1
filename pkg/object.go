package pkg

// Object is a game world object
type Object struct {
	Type  int32
	ID    int32
	X     int32
	Y     int32
	Flags int32
	Paths []Path
}

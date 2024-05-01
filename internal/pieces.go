package mes

type Piece struct {
	Recipe      []*Transformation
	CurrentStep int
	ControlID   int16 // ControlID of the piece in Codesys
}

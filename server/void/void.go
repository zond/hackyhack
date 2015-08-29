package void

type Void struct{}

func (v *Void) GetShortDesc() string {
	return "the Void"
}

func (v *Void) GetLongDesc() string {
	return "The infinite darkness of space."
}

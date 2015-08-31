package void

import "github.com/zond/hackyhack/server/persist"

type Void struct {
	persister *persist.Persister
}

func New(p *persist.Persister) *Void {
	return &Void{
		persister: p,
	}
}

func (v *Void) GetShortDesc() string {
	return "the Void"
}

func (v *Void) GetLongDesc() string {
	return "The infinite darkness of space."
}

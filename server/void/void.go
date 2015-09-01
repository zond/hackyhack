package void

import (
	"fmt"

	"github.com/zond/hackyhack/proc/messages"
	"github.com/zond/hackyhack/server/persist"
	"github.com/zond/hackyhack/server/resource"
)

type Void struct {
	persister *persist.Persister
}

func New(p *persist.Persister) *Void {
	return &Void{
		persister: p,
	}
}

func (v *Void) GetShortDesc() (string, *messages.Error) {
	return "the Void", nil
}

func (v *Void) GetLongDesc() (string, *messages.Error) {
	content := []resource.Resource{}
	if err := v.persister.Find(persist.NewF(resource.Resource{}).Add("Container"), &content); err != nil {
		return "", &messages.Error{
			Message: fmt.Sprintf("persister.Find failed: %v", err),
			Code:    messages.ErrorCodeDatabase,
		}
	}
	return "The infinite darkness of space.", nil
}

package resource

import (
	"time"

	"github.com/zond/hackyhack/server/persist"
)

type Resource struct {
	Id        string
	Owner     string
	Code      string
	Container string
	Content   []string
	UpdatedAt time.Time
	CreatedAt time.Time
}

func (r *Resource) RemoveContent(resource string) {
	newContent := []string{}
	for _, res := range r.Content {
		if res != resource {
			newContent = append(newContent, res)
		}
	}
	r.Content = newContent
}

func (r *Resource) AddContent(resource string) {
	found := false
	for _, res := range r.Content {
		if res == resource {
			found = true
			break
		}
	}
	if !found {
		r.Content = append(r.Content, resource)
	}
}

func (r *Resource) Remove(p *persist.Persister) error {
	return p.Transact(func(p *persist.Persister) error {
		if err := p.Get(r.Id, r); err != nil {
			return err
		}
		if r.Container != "" {
			oldCont := &Resource{}
			if err := p.Get(r.Container, oldCont); err != nil {
				return err
			}
			oldCont.RemoveContent(r.Id)
			if err := p.Put(r.Container, oldCont); err != nil {
				return err
			}
			r.Container = ""
			if err := p.Put(r.Id, r); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Resource) MoveTo(p *persist.Persister, container string) error {
	return p.Transact(func(p *persist.Persister) error {
		if err := p.Get(r.Id, r); err != nil {
			return err
		}
		cont := &Resource{}
		if err := p.Get(container, cont); err != nil {
			return err
		}
		if r.Container != "" {
			oldCont := &Resource{}
			if err := p.Get(r.Container, oldCont); err != nil {
				return err
			}
			oldCont.RemoveContent(r.Id)
			if err := p.Put(r.Container, oldCont); err != nil {
				return err
			}
		}
		r.Container = container
		if err := p.Put(r.Id, r); err != nil {
			return err
		}
		cont.AddContent(r.Id)
		if err := p.Put(container, cont); err != nil {
			return err
		}
		return nil
	})
}

package persist

type Mem map[string]interface{}

func NewMem() Mem {
	return Mem{}
}

func (m Mem) Set(val string, keys ...string) error {
	if len(keys) == 1 {
		m[keys[0]] = val
		return nil
	}
	current, found := m[keys[0]]
	currentM, ok := current.(Mem)
	if found && ok {
		currentM.Set(val, keys[1:]...)
		return nil
	}
	currentM = Mem{}
	m[keys[0]] = currentM
	return currentM.Set(val, keys[1:]...)
}

func (m Mem) Get(keys ...string) (string, error) {
	value, found := m[keys[0]]
	if !found {
		return "", ErrNotFound
	}
	if len(keys) == 1 {
		valueS, ok := value.(string)
		if !ok {
			return "", ErrNotFound
		}
		return valueS, nil
	}
	valueM, ok := value.(Mem)
	if !ok {
		return "", ErrNotFound
	}
	return valueM.Get(keys[1:]...)
}

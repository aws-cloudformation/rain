package diff

type mode int

const (
	added mode = iota
	changed
	removed
	unchanged
)

type diff interface {
	mode() mode
}

type diffValue struct {
	value     interface{}
	valueMode mode
}

type diffSlice []diff

type diffMap map[string]diff

func (m mode) mode() mode {
	return m
}

func (d diffValue) mode() mode {
	return d.valueMode
}

func (d diffSlice) mode() mode {
	mode := added

	for i, v := range d {
		if i == 0 {
			mode = v.mode()
		} else {
			if mode != v.mode() {
				mode = changed
			}
		}
	}

	return mode
}

func (d diffMap) mode() mode {
	slice := make(diffSlice, 0)

	for _, v := range d {
		slice = append(slice, v)
	}

	return slice.mode()
}

package diff

type mode string

const (
	Added     mode = "+ "
	Removed   mode = "- "
	Changed   mode = "| "
	Unchanged mode = "  "
)

type Diff interface {
	Mode() mode
}

type diffValue struct {
	value     interface{}
	valueMode mode
}

type diffSlice []Diff

type diffMap map[string]Diff

func (m mode) String() string {
	return string(m)
}

func (d diffValue) Mode() mode {
	return d.valueMode
}

func (d diffSlice) Mode() mode {
	mode := Added

	for i, v := range d {
		if i == 0 {
			mode = v.Mode()
		} else {
			if mode != v.Mode() {
				mode = Changed
			}
		}
	}

	return mode
}

func (d diffMap) Mode() mode {
	slice := make(diffSlice, 0)

	for _, v := range d {
		slice = append(slice, v)
	}

	return slice.Mode()
}

func (d diffMap) Keys() []string {
	keys := make([]string, len(d))

	i := 0
	for k, _ := range d {
		keys[i] = k
		i++
	}

	return keys
}

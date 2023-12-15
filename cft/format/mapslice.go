package format

// Borrowed this so that I could customize the encoder
// https://github.com/ake-persson/mapslice-json/blob/22c8edf57807/mapslice.go

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
)

// MapItem representation of one map item.
type MapItem struct {
	Key, Value interface{}
	index      uint64
}

// MapSlice of map items.
type MapSlice []MapItem

func (ms MapSlice) Len() int           { return len(ms) }
func (ms MapSlice) Less(i, j int) bool { return ms[i].index < ms[j].index }
func (ms MapSlice) Swap(i, j int)      { ms[i], ms[j] = ms[j], ms[i] }

var indexCounter uint64

func nextIndex() uint64 {
	indexCounter++
	return indexCounter
}

// MapItem as a string.
func (mi MapItem) String() string {
	return fmt.Sprintf("{%v %v}", mi.Key, mi.Value)
}

// MarshalJSON for map slice.
func (ms MapSlice) MarshalJSON() ([]byte, error) {
	buf := &bytes.Buffer{}
	buf.Write([]byte{'{'})
	for i, mi := range ms {
		b, err := ToJson(&mi.Value, "")
		if err != nil {
			return nil, err
		}
		buf.WriteString(fmt.Sprintf("%q:", fmt.Sprint(mi.Key)))
		buf.Write(b)
		if i < len(ms)-1 {
			buf.Write([]byte{','})
		}
	}
	buf.Write([]byte{'}'})
	return buf.Bytes(), nil
}

// UnmarshalJSON for map slice.
func (ms *MapSlice) UnmarshalJSON(b []byte) error {
	m := map[string]MapItem{}
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	for k, v := range m {
		*ms = append(*ms, MapItem{Key: k, Value: v.Value, index: v.index})
	}
	sort.Sort(*ms)
	return nil
}

// UnmarshalJSON for map item.
func (mi *MapItem) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	mi.Value = v
	mi.index = nextIndex()
	return nil
}

package state

import (
	"luago/number"
	"math"
)

type luaTable struct {
	a []luaValue
	m map[luaValue]luaValue
}

func newLuaTable(nArr, nRec int) *luaTable {
	t := &luaTable{}
	if nArr > 0 {
		t.a = make([]luaValue, 0, nArr)
	}
	if nRec > 0 {
		t.m = make(map[luaValue]luaValue, nRec)
	}
	return t
}

func (table *luaTable) get(key luaValue) luaValue {
	key = _normalizeKey(key)
	if idx, ok := key.(int64); ok {
		if 1 <= idx && idx <= int64(len(table.a)) {
			return table.a[idx-1]
		}
	}
	return table.m[key]
}

func (table *luaTable) len() int {
	return len(table.a)
}

func (table *luaTable) put(key, val luaValue) {
	if key == nil {
		panic("table index is nil")
	}

	if f, ok := key.(float64); ok && math.IsNaN(f) {
		panic("table index is NaN")
	}

	key = _normalizeKey(key)
	if idx, ok := key.(int64); ok && idx >= 1 {
		nArr := int64(len(table.a))
		if idx <= nArr {
			table.a[idx-1] = val
			if idx == nArr && val == nil {
				table._shrinkArr()
			}
			return
		}
		if idx == nArr+1 {
			delete(table.m, key)
			if val != nil {
				table.a = append(table.a, val)
				table._expandArr()
			}
			return
		}
	}

	if val != nil {
		if table.m == nil {
			table.m = make(map[luaValue]luaValue, 8)
		}
		table.m[key] = val
	} else {
		delete(table.m, key)
	}
}

func _normalizeKey(key luaValue) luaValue {
	if f, ok := key.(float64); ok {
		if i, ok := number.FloatToInteger(f); ok {
			return i
		}
	}
	return key
}

func (table *luaTable) _shrinkArr() {
	nArr := len(table.a)
	for nArr > 0 {
		if table.a[nArr-1] != nil {
			break
		}
	}
	table.a = table.a[:nArr]
}

func (table *luaTable) _expandArr() {
	for idx := int64(len(table.a)) + 1; true; idx++ {
		if val, found := table.m[idx]; found {
			delete(table.m, idx)
			table.a = append(table.a, val)
		} else {
			break
		}
	}
}

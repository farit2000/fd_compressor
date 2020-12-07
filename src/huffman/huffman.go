package huffman

import (
	"sort"
)

// ValueType - это тип значения, хранящегося в Node.
type ValueType int32

// Узел в дереве Хаффмана.
type Node struct {
	Parent *Node     // Необязательный родительский узел для быстрого считывания кода
	Left   *Node     // Необязательный левый узел
	Right  *Node     // Необязательный правый узел
	Count  int       // Относительная частота
	Value  ValueType // Необязательное значение, устанавливается, если это лист
}

// Код возвращает код Хаффмана узла.
// Левые дочерние элементы получают бит 0, правые дочерние элементы получают бит 1.
// Реализация использует Node.Parent для перехода «вверх» по дереву.
func (n *Node) Code() (r uint64, bits byte) {
	for parent := n.Parent; parent != nil; n, parent = parent, parent.Parent {
		if parent.Right == n { // бит 1
			r |= 1 << bits
		} // иначе бит 0 => ничего общего с r
		bits++
	}
	return
}

// SortNodes реализует sort.Interface в порядке, определяемом Node.Count.
type SortNodes []*Node

func (sn SortNodes) Len() int           { return len(sn) }
func (sn SortNodes) Less(i, j int) bool { return sn[i].Count < sn[j].Count }
func (sn SortNodes) Swap(i, j int)      { sn[i], sn[j] = sn[j], sn[i] }

// BuildSorted строит дерево Хаффмана из указанных листьев, которые должны быть отсортированы по Node.Count.
// Содержимое переданного фрагмента изменяется, если это нежелательно, передать копию.
// Гарантированно, что один и тот же входной срез приведет к тому же дереву Хаффмана.
func BuildSorted(leaves []*Node) *Node {
	if len(leaves) == 0 {
		return nil
	}
	for len(leaves) > 1 {
		left, right := leaves[0], leaves[1]
		parentCount := left.Count + right.Count
		parent := &Node{Left: left, Right: right, Count: parentCount}
		left.Parent = parent
		right.Parent = parent
		ls := leaves[2:]
		idx := sort.Search(len(ls), func(i int) bool { return ls[i].Count >= parentCount })
		idx += 2
		// Вставка
		copy(leaves[1:], leaves[2:idx])
		leaves[idx-1] = parent
		leaves = leaves[1:]
	}
	return leaves[0]
}

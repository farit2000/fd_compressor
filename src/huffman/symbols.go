package huffman

import (
	"sort"
)

const (
	newValue    ValueType           = 1<<31 - 1 - iota // Значение, представляющее новое значение
	eofValue                                           // Значение, представляющее конец данных
	extraValues = iota                                 // Количество дополнительных пользовательских значений
	maxValues   = 256 + extraValues                    // Максимальные значения: количество байтов + дополнительные значения
)

// win - буфер скользящего окна, основа таблицы символов.
type win struct {
	buf    []ValueType // Содержимое буфера окна
	pos    int         // Содержимое буфера окна
	filled bool        // Сообщает, заполнен ли буфер
}

// store сохраняет следующий символ и сдвигает окно, если оно уже заполнено.
func (w *win) store(symbol ValueType) {
	w.buf[w.pos] = symbol
	w.pos++
	if w.pos == len(w.buf) {
		w.pos = 0
		if !w.filled {
			w.filled = true
		}
	}
}

// символы управляют таблицей символов и их частотами.
type symbols struct {
	// листья дерева Хаффмана, ранее встреченные символы.
	// Отсортировано по Node.Count, потомку (так что новые узлы можно просто добавлять)!
	leaves []*Node
	buffer []*Node // Многоразовый буфер для передачи при построении дерева Хаффмана
	root *Node // Корень дерева Хаффмана.
	valueMap map[ValueType]*Node // Сопоставить значение с узлом
	win *win // Буфер окна, ноль, если буфер окна не используется
}

// newSymbols создает новые символы.
func newSymbols(o *Options) *symbols {
	// начальные листья: 2 узла (newValue и eofValue) с count = 1 и большой емкостью
	leaves := make([]*Node, extraValues, maxValues)
	leaves[0] = &Node{Value: newValue, Count: 1}
	leaves[1] = &Node{Value: eofValue, Count: 1}
	valueMap := make(map[ValueType]*Node, cap(leaves))
	for _, v := range leaves {
		valueMap[v.Value] = v
	}
	s := &symbols{leaves: leaves, valueMap: valueMap, buffer: make([]*Node, 0, maxValues)}
	if o.WinSize > 0 {
		s.win = &win{buf: make([]ValueType, o.WinSize)}
	}
	// Читателю нужно сразу же подготовить дерево Хаффмана, поэтому создайте его:
	s.rebuildTree()
	return s
}

// update обновляет таблицу символов, увеличивая счетчик вхождений указанного узла.
func (s *symbols) update(node *Node) {
	// Мы должны отсортировать листья!
	// Итак, сначала находим узел в срезе листьев с помощью двоичного поиска
	// (помните: листья сортируются по потомку Node.Count)
	ls, count := s.leaves, node.Count
	idx := sort.Search(len(ls)-extraValues, func(i int) bool { return ls[i].Count <= count })
	idx2 := idx // Store it for later use
	// idx указывает на первый (самый низкий) узел, имеющий счетчик.
	// Может быть больше узлов с таким же количеством, найдем наш узел:
	for ; ls[idx] != node; idx++ {}
	// Если есть больше узлов с таким же количеством, наш узел должен быть переключен
	// с узлом, имеющим такое же количество и самый низкий индекс
	// (поэтому срез остается отсортированным после увеличения счетчика нашего узла).
	if idx2 != idx {
		ls[idx2], ls[idx] = ls[idx], ls[idx2]
	}
	node.Count++
	s.updateWin(node.Value)
	s.rebuildTree()
}

// updateWin обновляет окно: сдвигает его, если оно уже заполнено, и сохраняет текущий обработанный символ.
func (s *symbols) updateWin(symbol ValueType) {
	if s.win == nil {
		return
	}
	if s.win.filled {
		// Обрабатываем перемещение символа из оконного буфера:
		node := s.valueMap[s.win.buf[s.win.pos]]
		// сначала находим узел в срезе листьев с помощью двоичного поиска
		ls, count := s.leaves, node.Count
		idx := sort.Search(len(ls)-extraValues, func(i int) bool { return ls[i].Count <= count })
		// idx указывает на первый (самый низкий) узел, имеющий счетчик.
		// Может быть больше узлов с таким же количеством
		for ; ls[idx] != node; idx++ {
		}
		if count > 1 {
			// Если есть больше узлов с таким же количеством, наш узел должен быть переключен
			// с узлом, имеющим такое же количество и самый высокий индекс
			// (поэтому срез остается отсортированным после уменьшения счетчика нашего узла)
			idx2 := idx + 1
			for ; idx2 < len(ls)-extraValues && ls[idx2].Count == count; idx2++ {}
			if idx2 = idx2 - 1; idx2 != idx {
				ls[idx2], ls[idx] = ls[idx], ls[idx2]
			}
			node.Count--
		} else {
			// Счетчик уменьшится до нуля: удалить узел
			s.leaves = append(ls[:idx], ls[idx+1:]...)
			// Также удаляем из valueMap:
			delete(s.valueMap, node.Value)
		}
	}
	s.win.store(symbol)
}

// insert вставляет обнаруженный новый символ.
func (s *symbols) insert(symbol ValueType) {
	node := &Node{Value: symbol, Count: 1}
	// оставляет отсортированный потомок, поэтому мы можем просто добавить.
	// Но дополнительные значения в конце никогда не увеличиваются, поэтому мы вставляем перед ними:
	ls := s.leaves
	// расширить на 1 для нового узла
	ls = ls[:len(ls)+1]
	// Копируем лишние значения в конец (на 1 выше)
	copy(ls[len(ls)-extraValues:], ls[len(ls)-extraValues-1:])
	// И вставляем новый узел
	ls[len(ls)-extraValues-1] = node
	s.leaves = ls
	s.valueMap[node.Value] = node
	s.updateWin(node.Value)
	s.rebuildTree()
}

// rebuildTree перестраивает дерево Хаффмана.
func (s *symbols) rebuildTree() {
	// BuildSorted () изменяет срез, поэтому сделайте копию:
	// оставляет отсортированный потомок, поэтому заполняем в обратном направлении:
	j := len(s.leaves)
	ls := s.buffer[:j]
	for _, v := range s.leaves {
		j--
		ls[j] = v
	}
	s.root = BuildSorted(ls)
}

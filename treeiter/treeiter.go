//go:build !solution

package treeiter

type Treeier[N any] interface {
	comparable
	Left() N
	Right() N
}

func DoInOrder[T Treeier[T]](tree T, f func(t T)) {
	var zero T
	if tree == zero {
		return
	}

	DoInOrder[T](tree.Left(), f)
	f(tree)
	DoInOrder[T](tree.Right(), f)
}

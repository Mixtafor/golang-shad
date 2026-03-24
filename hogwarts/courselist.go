//go:build !solution

package hogwarts

import "slices"

func GetCourseList(prereqs map[string][]string) []string {
	uniqueVert := make(map[string]struct{})

	vertDegrees := make(map[string]int)

	for vertFrom := range prereqs {
		uniqueVert[vertFrom] = struct{}{}
		for _, vertTo := range prereqs[vertFrom] {
			vertDegrees[vertTo]++
			uniqueVert[vertTo] = struct{}{}
		}
	}

	queue := make([]string, 0)

	ans := make([]string, 0)

	for vertFrom := range uniqueVert {
		if vertDegrees[vertFrom] == 0 {
			queue = append(queue, vertFrom)
		}
	}

	for len(queue) != 0 {
		front := queue[0]
		queue = queue[1:]
		ans = append(ans, front)

		for _, neighbor := range prereqs[front] {
			vertDegrees[neighbor]--
			if vertDegrees[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if len(ans) != len(uniqueVert) {
		panic("cycle")
	}
	slices.Reverse(ans)
	return ans
}

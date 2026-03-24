//go:build !solution

package hotelbusiness

import "sort"

type Guest struct {
	CheckInDate  int
	CheckOutDate int
}

type Load struct {
	StartDate  int
	GuestCount int
}

func ComputeLoad(guests []Guest) []Load {
	load := []Load{}
	dates := make(map[int]int)

	for _, duration := range guests {
		dates[duration.CheckInDate]++
		dates[duration.CheckOutDate]--
	}

	sorteddates := make([]int, 0, len(dates))

	for date := range dates {
		sorteddates = append(sorteddates, date)
	}

	sort.Ints(sorteddates)

	sum := 0
	for _, date := range sorteddates {
		if dates[date] != 0 {
			sum += dates[date]
			load = append(load, Load{date, sum})
		}
	}

	return load
}

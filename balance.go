package main

import (
	"errors"
	"fmt"
	"sort"
)

type Programmer struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Points int    `json:"points"`
}

type Team struct {
	Members []Programmer
	Total   int
}

// Split распределяет участников по k командам так, чтобы суммы баллов
// команд были как можно ближе друг к другу.
// Если equalSize == true, размеры команд отличаются не более чем на 1.
// Каждая команда получает хотя бы одного участника.
//
// Алгоритм: жадное распределение (самый сильный — в самую слабую команду),
// затем локальный поиск: переносы и обмены участниками между командами,
// пока уменьшается сумма квадратов баллов команд (т.е. разброс).
func Split(people []Programmer, k int, equalSize bool) ([]Team, error) {
	n := len(people)
	if n == 0 {
		return nil, errors.New("список участников пуст")
	}
	if k < 1 {
		return nil, errors.New("количество команд должно быть не меньше 1")
	}
	if k > n {
		return nil, fmt.Errorf("команд (%d) больше, чем участников (%d)", k, n)
	}

	sorted := make([]Programmer, n)
	copy(sorted, people)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Points > sorted[j].Points })

	teams := make([]Team, k)
	base, extra := n/k, n%k // при равных размерах: extra команд получат base+1
	big := 0                // сколько команд уже имеют размер base+1

	for _, p := range sorted {
		best := -1
		for i := range teams {
			size := len(teams[i].Members)
			if equalSize && (size > base || (size == base && big >= extra)) {
				continue // команда заполнена
			}
			if best == -1 ||
				teams[i].Total < teams[best].Total ||
				(teams[i].Total == teams[best].Total && size < len(teams[best].Members)) {
				best = i
			}
		}
		if equalSize && len(teams[best].Members) == base {
			big++
		}
		teams[best].Members = append(teams[best].Members, p)
		teams[best].Total += p.Points
	}

	improve(teams, equalSize)

	for i := range teams {
		m := teams[i].Members
		sort.SliceStable(m, func(a, b int) bool { return m[a].Points > m[b].Points })
	}
	sort.SliceStable(teams, func(a, b int) bool { return teams[a].Total > teams[b].Total })
	return teams, nil
}

// improve выполняет локальный поиск: применяет перенос или обмен участников,
// если он строго уменьшает сумму квадратов баллов двух команд.
// Целевая функция целочисленная и строго убывает, поэтому цикл конечен.
func improve(teams []Team, equalSize bool) {
	gain := func(a, b, d int) bool { // выгодно ли a -= d, b += d
		return (a-d)*(a-d)+(b+d)*(b+d) < a*a+b*b
	}

	for changed := true; changed; {
		changed = false
		for a := range teams {
			for b := range teams {
				if a == b {
					continue
				}
				A, B := &teams[a], &teams[b]

				// Перенос участника из A в B.
				canMove := len(A.Members) > 1
				if equalSize {
					// размеры остаются в пределах ±1: меняются местами base+1 и base
					canMove = len(A.Members) == len(B.Members)+1
				}
				if canMove {
					for i, x := range A.Members {
						if gain(A.Total, B.Total, x.Points) {
							A.Members = append(A.Members[:i:i], A.Members[i+1:]...)
							B.Members = append(B.Members, x)
							A.Total -= x.Points
							B.Total += x.Points
							changed = true
							break
						}
					}
				}

				// Обмен участниками x из A и y из B.
			swap:
				for i, x := range A.Members {
					for j, y := range B.Members {
						if d := x.Points - y.Points; d > 0 && gain(A.Total, B.Total, d) {
							A.Members[i], B.Members[j] = y, x
							A.Total -= d
							B.Total += d
							changed = true
							break swap
						}
					}
				}
			}
		}
	}
}

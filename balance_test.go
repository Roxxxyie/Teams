package main

import "testing"

var sample = []Programmer{
	{1, "A", 25}, {2, "B", 75}, {3, "C", 25},
	{4, "D", 25}, {5, "E", 50}, {6, "F", 25},
}

func spread(teams []Team) int {
	lo, hi := teams[0].Total, teams[0].Total
	for _, t := range teams {
		lo, hi = min(lo, t.Total), max(hi, t.Total)
	}
	return hi - lo
}

func checkAllAssigned(t *testing.T, people []Programmer, teams []Team) {
	t.Helper()
	seen := map[int]int{}
	for _, tm := range teams {
		if len(tm.Members) == 0 {
			t.Fatal("пустая команда")
		}
		sum := 0
		for _, m := range tm.Members {
			seen[m.ID]++
			sum += m.Points
		}
		if sum != tm.Total {
			t.Fatalf("Total=%d, а сумма участников %d", tm.Total, sum)
		}
	}
	for _, p := range people {
		if seen[p.ID] != 1 {
			t.Fatalf("участник %d распределён %d раз", p.ID, seen[p.ID])
		}
	}
}

func TestSplit(t *testing.T) {
	cases := []struct {
		name       string
		k          int
		equal      bool
		wantSpread int
	}{
		{"2 команды, ровно", 2, true, 25},    // 225 нечётно в единицах по 25
		{"3 команды, ровно", 3, true, 50},    // 75+25 / 50+25 / 25+25
		{"3 команды, свободно", 3, false, 0}, // 75 / 50+25 / 25+25+25
		{"6 команд", 6, true, 50},
		{"1 команда", 1, false, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			teams, err := Split(sample, c.k, c.equal)
			if err != nil {
				t.Fatal(err)
			}
			if len(teams) != c.k {
				t.Fatalf("команд %d, ожидалось %d", len(teams), c.k)
			}
			checkAllAssigned(t, sample, teams)
			if got := spread(teams); got != c.wantSpread {
				t.Errorf("разброс %d, ожидался %d", got, c.wantSpread)
			}
			if c.equal {
				lo, hi := len(sample), 0
				for _, tm := range teams {
					lo, hi = min(lo, len(tm.Members)), max(hi, len(tm.Members))
				}
				if hi-lo > 1 {
					t.Errorf("размеры команд отличаются на %d", hi-lo)
				}
			}
		})
	}
}

func TestSplitUnequalSizesBalance(t *testing.T) {
	// Один сильный против многих слабых: без требования равных размеров
	// оптимум 100 / 25×4.
	people := []Programmer{{1, "", 100}, {2, "", 25}, {3, "", 25}, {4, "", 25}, {5, "", 25}}
	teams, err := Split(people, 2, false)
	if err != nil {
		t.Fatal(err)
	}
	checkAllAssigned(t, people, teams)
	if s := spread(teams); s != 0 {
		t.Errorf("разброс %d, ожидался 0", s)
	}
}

func TestSplitErrors(t *testing.T) {
	if _, err := Split(sample, 0, false); err == nil {
		t.Error("ожидалась ошибка при k=0")
	}
	if _, err := Split(sample, 7, false); err == nil {
		t.Error("ожидалась ошибка при k > n")
	}
	if _, err := Split(nil, 1, false); err == nil {
		t.Error("ожидалась ошибка для пустого списка")
	}
}

package balance

import (
	"fmt"
	"strings"
	"testing"
)

func TestParseValid(t *testing.T) {
	data := "\xEF\xBB\xBF" + `{"items":[{"id":1,"name":" Аня ","points":0},{"id":2,"points":100}]}`
	people, err := Parse([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if len(people) != 2 || people[0].Name != "Аня" || people[1].Points != 100 {
		t.Fatalf("неожиданный результат: %+v", people)
	}
}

func TestParseErrors(t *testing.T) {
	cases := []struct {
		name, json string
		want       []string // все подстроки должны быть в сообщении
	}{
		{"обрыв JSON", `{"items":[`, []string{"обрывается"}},
		{"синтаксис JSON", `{"items":[{"id":1,}]}`, []string{"корректным JSON"}},
		{"нет items", `{}`, []string{`нет поля "items"`}},
		{"пустой список", `{"items":[]}`, []string{"пуст"}},
		{"опечатка в поле", `{"items":[{"id":1,"point":50}]}`, []string{`"point"`, "опечатка"}},
		{"строка вместо числа", `{"items":[{"id":1,"points":"50"}]}`, []string{"points", "целое число"}},
		{"дробные баллы", `{"items":[{"id":1,"points":50.5}]}`, []string{"points", "целое число"}},
		{"нет points", `{"items":[{"id":1,"name":"Аня"}]}`, []string{"№1", "«Аня»", `нет поля "points"`}},
		{"нет id", `{"items":[{"points":10}]}`, []string{`нет поля "id"`}},
		{"баллы вне диапазона", `{"items":[{"id":1,"points":120},{"id":2,"points":-5}]}`,
			[]string{"проблем: 2", "баллы 120", "баллы -5", "0–100"}},
		{"дубликат id", `{"items":[{"id":7,"points":10},{"id":3,"points":10},{"id":7,"name":"Боря","points":10}]}`,
			[]string{"участник №3", "id 7 уже занят участником №1"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := Parse([]byte(c.json))
			if err == nil {
				t.Fatal("ожидалась ошибка")
			}
			for _, w := range c.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("в сообщении нет %q:\n%s", w, err)
				}
			}
		})
	}
}

// signature — состав команд в виде строки, чтобы сравнивать варианты.
func signature(teams []Team) string {
	var b strings.Builder
	for _, tm := range teams {
		for _, m := range tm.Members {
			fmt.Fprintf(&b, "%d,", m.ID)
		}
		b.WriteString("|")
	}
	return b.String()
}

func TestSplitSeed(t *testing.T) {
	people := make([]Programmer, 12)
	for i := range people {
		people[i] = Programmer{ID: i + 1, Points: 25 * (1 + i%3)} // много равных баллов
	}

	base, _ := Split(people, 3, true, 0)
	variants := map[string]bool{}
	for seed := int64(1); seed <= 20; seed++ {
		a, err := Split(people, 3, true, seed)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := Split(people, 3, true, seed)
		if signature(a) != signature(b) {
			t.Fatalf("seed %d: результат не воспроизводится", seed)
		}
		checkAllAssigned(t, people, a)
		if spread(a) != spread(base) {
			t.Errorf("seed %d: разброс %d, без seed %d — seed не должен ухудшать баланс",
				seed, spread(a), spread(base))
		}
		variants[signature(a)] = true
	}
	if len(variants) < 2 {
		t.Error("разные seed дают одинаковый состав команд")
	}
}

package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type input struct {
	Items []Programmer `json:"items"`
}

func main() {
	file := flag.String("file", "programmers.json", "путь к JSON-файлу с участниками")
	flag.Parse()

	people, err := load(*file)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Ошибка:", err)
		os.Exit(1)
	}
	fmt.Printf("Загружено участников: %d\n\n", len(people))

	in := bufio.NewScanner(os.Stdin)
	k := askTeams(in, len(people))
	equal := askYesNo(in, "Обязательно равное количество участников в командах? (д/н): ")

	teams, err := Split(people, k, equal)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Ошибка:", err)
		os.Exit(1)
	}
	printTeams(teams)
}

func load(path string) ([]Programmer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var in input
	if err := json.Unmarshal(data, &in); err != nil {
		return nil, fmt.Errorf("не удалось разобрать %s: %w", path, err)
	}
	if len(in.Items) == 0 {
		return nil, fmt.Errorf("в %s нет участников (поле \"items\")", path)
	}
	return in.Items, nil
}

func readLine(in *bufio.Scanner, prompt string) string {
	fmt.Print(prompt)
	if !in.Scan() {
		fmt.Fprintln(os.Stderr, "\nВвод прерван")
		os.Exit(1)
	}
	// PowerShell при перенаправлении ввода может добавить BOM в начало.
	return strings.TrimSpace(strings.TrimPrefix(in.Text(), string(rune(0xFEFF))))
}

func askTeams(in *bufio.Scanner, n int) int {
	for {
		s := readLine(in, fmt.Sprintf("На сколько команд делить (1-%d)? ", n))
		k, err := strconv.Atoi(s)
		if err == nil && k >= 1 && k <= n {
			return k
		}
		fmt.Printf("Введите целое число от 1 до %d\n", n)
	}
}

func askYesNo(in *bufio.Scanner, prompt string) bool {
	for {
		switch strings.ToLower(readLine(in, prompt)) {
		case "д", "да", "y", "yes":
			return true
		case "н", "нет", "n", "no":
			return false
		}
		fmt.Println("Ответьте «д» или «н»")
	}
}

func printTeams(teams []Team) {
	minT, maxT, sum := teams[0].Total, teams[0].Total, 0
	for i, t := range teams {
		fmt.Printf("\nКоманда %d — %d чел., сумма баллов: %d\n", i+1, len(t.Members), t.Total)
		for _, m := range t.Members {
			name := m.Name
			if name == "" {
				name = "(без имени)"
			}
			fmt.Printf("  #%-3d %-25s %d\n", m.ID, name, m.Points)
		}
		sum += t.Total
		minT, maxT = min(minT, t.Total), max(maxT, t.Total)
	}
	fmt.Printf("\nСредняя сумма: %.1f, разброс (макс − мин): %d\n",
		float64(sum)/float64(len(teams)), maxT-minT)
}

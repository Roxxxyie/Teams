package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	MinPoints = 0
	MaxPoints = 100
)

// rawProgrammer — участник в том виде, в каком он записан в JSON.
// Указатели нужны, чтобы отличить отсутствующее поле от нулевого значения.
type rawProgrammer struct {
	ID     *int    `json:"id"`
	Name   *string `json:"name"`
	Points *int    `json:"points"`
}

type rawInput struct {
	Items *[]rawProgrammer `json:"items"`
}

// Parse разбирает JSON и проверяет данные до запуска распределения.
// Возвращает все найденные проблемы сразу, а не только первую.
func Parse(data []byte) ([]Programmer, error) {
	data = bytes.TrimPrefix(data, []byte("\xEF\xBB\xBF")) // BOM от Блокнота/PowerShell

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields() // опечатка вроде "point" вместо "points" — ошибка, а не тихий 0
	var in rawInput
	if err := dec.Decode(&in); err != nil {
		return nil, explainJSONError(err)
	}
	if in.Items == nil {
		return nil, errors.New(`нет поля "items" со списком участников`)
	}
	if len(*in.Items) == 0 {
		return nil, errors.New(`список "items" пуст`)
	}

	var problems []string
	people := make([]Programmer, 0, len(*in.Items))
	firstPos := map[int]int{} // id → номер участника, где он встретился впервые

	for i, r := range *in.Items {
		pos := i + 1
		who := describe(pos, r)

		if r.ID == nil {
			problems = append(problems, who+`: нет поля "id"`)
		} else if prev, dup := firstPos[*r.ID]; dup {
			problems = append(problems, fmt.Sprintf("%s: id %d уже занят участником №%d", who, *r.ID, prev))
		} else {
			firstPos[*r.ID] = pos
		}

		if r.Points == nil {
			problems = append(problems, who+`: нет поля "points"`)
		} else if *r.Points < MinPoints || *r.Points > MaxPoints {
			problems = append(problems, fmt.Sprintf("%s: баллы %d вне диапазона %d–%d",
				who, *r.Points, MinPoints, MaxPoints))
		}

		p := Programmer{}
		if r.ID != nil {
			p.ID = *r.ID
		}
		if r.Name != nil {
			p.Name = strings.TrimSpace(*r.Name)
		}
		if r.Points != nil {
			p.Points = *r.Points
		}
		people = append(people, p)
	}

	if len(problems) > 0 {
		return nil, fmt.Errorf("найдено проблем: %d\n  - %s", len(problems), strings.Join(problems, "\n  - "))
	}
	return people, nil
}

func describe(pos int, r rawProgrammer) string {
	s := fmt.Sprintf("участник №%d", pos)
	var parts []string
	if r.ID != nil {
		parts = append(parts, fmt.Sprintf("id %d", *r.ID))
	}
	if r.Name != nil && strings.TrimSpace(*r.Name) != "" {
		parts = append(parts, "«"+strings.TrimSpace(*r.Name)+"»")
	}
	if len(parts) > 0 {
		s += " (" + strings.Join(parts, ", ") + ")"
	}
	return s
}

func explainJSONError(err error) error {
	var syntax *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	switch {
	case errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		return errors.New("файл пуст или JSON обрывается (не хватает закрывающих скобок)")
	case errors.As(err, &syntax):
		return fmt.Errorf("файл не является корректным JSON (позиция %d): %v", syntax.Offset, err)
	case errors.As(err, &typeErr):
		return fmt.Errorf("поле %q должно быть типа %s, а в файле %s",
			typeErr.Field, typeName(typeErr.Type.String()), typeErr.Value)
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		field := strings.TrimPrefix(err.Error(), "json: unknown field ")
		return fmt.Errorf("неизвестное поле %s (допустимы: items, id, name, points) — возможно, опечатка", field)
	}
	return fmt.Errorf("не удалось разобрать JSON: %w", err)
}

func typeName(t string) string {
	switch strings.TrimLeft(t, "*[]") {
	case "int":
		return "целое число"
	case "string":
		return "строка"
	}
	return t
}

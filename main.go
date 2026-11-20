// Балансировщик команд — настольное приложение на Fyne.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"teams/internal/balance"
)

const defaultFile = "programmers.json"

type ui struct {
	app fyne.App
	win fyne.Window

	people []balance.Programmer
	teams  []balance.Team
	dir    string // папка последнего открытого файла

	fileLabel  *widget.Label
	statsLabel *widget.Label
	list       *widget.List

	teamsEntry *widget.Entry
	equalCheck *widget.Check
	seedEntry  *widget.Entry

	summary *widget.Label
	results *container.Scroll
	copyBtn *widget.Button
}

func main() {
	a := app.NewWithID("ru.hackathon.teams")
	w := a.NewWindow("Балансировщик команд")

	u := &ui{app: a, win: w}
	w.SetContent(u.build())
	w.Resize(fyne.NewSize(1150, 720))
	w.CenterOnScreen()

	u.loadDefault()
	w.ShowAndRun()
}

func (u *ui) build() fyne.CanvasObject {
	// ---- Участники ----
	u.fileLabel = widget.NewLabel("Файл не выбран")
	u.fileLabel.Truncation = fyne.TextTruncateEllipsis
	u.statsLabel = widget.NewLabel("")
	u.statsLabel.Importance = widget.LowImportance

	u.list = widget.NewList(
		func() int { return len(u.people) },
		func() fyne.CanvasObject {
			id := widget.NewLabel("#000")
			id.Importance = widget.LowImportance
			name := widget.NewLabel("")
			name.Truncation = fyne.TextTruncateEllipsis
			pts := widget.NewLabel("000")
			pts.TextStyle.Bold = true
			return container.NewBorder(nil, nil, id, pts, name)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			p := u.people[i]
			c := o.(*fyne.Container)
			name, id, pts := c.Objects[0].(*widget.Label), c.Objects[1].(*widget.Label), c.Objects[2].(*widget.Label)
			id.SetText(fmt.Sprintf("#%d", p.ID))
			name.SetText(displayName(p))
			pts.Importance = pointsImportance(p.Points)
			pts.SetText(strconv.Itoa(p.Points))
		},
	)
	u.list.OnSelected = func(widget.ListItemID) { u.list.UnselectAll() }

	openBtn := widget.NewButtonWithIcon("Открыть JSON…", theme.FolderOpenIcon(), u.openFile)
	reloadBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), u.reload)

	peopleCard := widget.NewCard("Участники", "", container.NewBorder(
		container.NewVBox(
			container.NewBorder(nil, nil, nil, reloadBtn, openBtn),
			u.fileLabel,
		),
		u.statsLabel, nil, nil,
		u.list,
	))

	// ---- Параметры ----
	u.teamsEntry = widget.NewEntry()
	u.teamsEntry.SetText("2")
	u.teamsEntry.OnSubmitted = func(string) { u.distribute() }
	minus := widget.NewButtonWithIcon("", theme.ContentRemoveIcon(), func() { u.stepTeams(-1) })
	plus := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() { u.stepTeams(+1) })

	u.equalCheck = widget.NewCheck("Равное количество участников", func(bool) {
		if u.teams != nil {
			u.distribute()
		}
	})
	u.equalCheck.SetChecked(true)

	u.seedEntry = widget.NewEntry()
	u.seedEntry.SetText("0")
	u.seedEntry.OnSubmitted = func(string) { u.distribute() }
	diceBtn := widget.NewButtonWithIcon("Другой вариант", theme.ViewRefreshIcon(), u.nextSeed)

	runBtn := widget.NewButtonWithIcon("Распределить", theme.MediaPlayIcon(), u.distribute)
	runBtn.Importance = widget.HighImportance

	form := widget.NewForm(
		widget.NewFormItem("Команд", container.NewBorder(nil, nil, minus, plus, u.teamsEntry)),
		widget.NewFormItem("Вариант (seed)", container.NewBorder(nil, nil, nil, diceBtn, u.seedEntry)),
	)
	settingsCard := widget.NewCard("Параметры", "", container.NewVBox(form, u.equalCheck, runBtn))

	left := container.NewBorder(nil, settingsCard, nil, nil, peopleCard)

	// ---- Результат ----
	u.summary = widget.NewLabel("")
	u.summary.TextStyle.Bold = true
	u.summary.Wrapping = fyne.TextWrapWord
	u.copyBtn = widget.NewButtonWithIcon("Копировать", theme.ContentCopyIcon(), u.copyResult)
	u.copyBtn.Disable()

	u.results = container.NewVScroll(placeholder("Загрузите участников и нажмите «Распределить»"))

	right := container.NewBorder(
		container.NewBorder(nil, widget.NewSeparator(), nil, u.copyBtn, u.summary),
		nil, nil, nil,
		u.results,
	)

	split := container.NewHSplit(left, container.NewPadded(right))
	split.Offset = 0.32
	return split
}

// ---------- Загрузка данных ----------

func (u *ui) loadDefault() {
	candidates := []string{defaultFile}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), defaultFile))
	}
	for _, path := range candidates {
		if data, err := os.ReadFile(path); err == nil {
			abs, _ := filepath.Abs(path)
			u.setData(abs, data)
			return
		}
	}
}

func (u *ui) reload() {
	if u.fileLabel.Text == "" || u.dir == "" {
		u.openFile()
		return
	}
	path := filepath.Join(u.dir, u.fileLabel.Text)
	data, err := os.ReadFile(path)
	if err != nil {
		u.showError(fmt.Errorf("не удалось прочитать %s: %w", path, err))
		return
	}
	u.setData(path, data)
}

func (u *ui) openFile() {
	d := dialog.NewFileOpen(func(rc fyne.URIReadCloser, err error) {
		if err != nil {
			u.showError(err)
			return
		}
		if rc == nil {
			return // отмена
		}
		defer rc.Close()
		data, err := io.ReadAll(rc)
		if err != nil {
			u.showError(err)
			return
		}
		u.setData(rc.URI().Path(), data)
	}, u.win)
	d.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
	if u.dir != "" {
		if loc, err := storage.ListerForURI(storage.NewFileURI(u.dir)); err == nil {
			d.SetLocation(loc)
		}
	}
	d.Resize(fyne.NewSize(800, 550))
	d.Show()
}

func (u *ui) setData(path string, data []byte) {
	people, err := balance.Parse(data)
	if err != nil {
		u.showError(fmt.Errorf("Файл %s: %w", filepath.Base(path), err))
		return
	}
	u.people = people
	u.dir = filepath.Dir(path)
	u.fileLabel.SetText(filepath.Base(path))

	sum := 0
	for _, p := range people {
		sum += p.Points
	}
	u.statsLabel.SetText(fmt.Sprintf("%d %s · сумма баллов %d", len(people), plural(len(people), "участник", "участника", "участников"), sum))
	u.list.Refresh()

	if k, err := strconv.Atoi(strings.TrimSpace(u.teamsEntry.Text)); err != nil || k > len(people) {
		u.teamsEntry.SetText(strconv.Itoa(min(2, len(people))))
	}
	u.distribute()
}

// ---------- Параметры ----------

func (u *ui) stepTeams(d int) {
	k, err := strconv.Atoi(strings.TrimSpace(u.teamsEntry.Text))
	if err != nil {
		k = 2
	}
	k += d
	if n := len(u.people); n > 0 {
		k = min(k, n)
	}
	k = max(k, 1)
	u.teamsEntry.SetText(strconv.Itoa(k))
	if u.teams != nil {
		u.distribute()
	}
}

func (u *ui) nextSeed() {
	s, err := strconv.ParseInt(strings.TrimSpace(u.seedEntry.Text), 10, 64)
	if err != nil {
		s = 0
	}
	u.seedEntry.SetText(strconv.FormatInt(s+1, 10))
	u.distribute()
}

func (u *ui) params() (k int, seed int64, err error) {
	n := len(u.people)
	k, err = strconv.Atoi(strings.TrimSpace(u.teamsEntry.Text))
	if err != nil || k < 1 || k > n {
		return 0, 0, fmt.Errorf("количество команд должно быть целым числом от 1 до %d", n)
	}
	seed, err = strconv.ParseInt(strings.TrimSpace(u.seedEntry.Text), 10, 64)
	if err != nil {
		return 0, 0, errors.New("вариант (seed) должен быть целым числом")
	}
	return k, seed, nil
}

// ---------- Распределение и вывод ----------

func (u *ui) distribute() {
	if len(u.people) == 0 {
		u.showError(errors.New("сначала откройте JSON-файл с участниками"))
		return
	}
	k, seed, err := u.params()
	if err != nil {
		u.showError(err)
		return
	}
	teams, err := balance.Split(u.people, k, u.equalCheck.Checked, seed)
	if err != nil {
		u.showError(err)
		return
	}
	u.teams = teams
	u.render(seed)
}

func (u *ui) render(seed int64) {
	lo, hi, sum := u.teams[0].Total, u.teams[0].Total, 0
	for _, t := range u.teams {
		lo, hi, sum = min(lo, t.Total), max(hi, t.Total), sum+t.Total
	}
	avg := float64(sum) / float64(len(u.teams))

	verdict := "идеально ровно"
	if spread := hi - lo; spread > 0 {
		verdict = fmt.Sprintf("разброс %d", spread)
	}
	u.summary.SetText(fmt.Sprintf("Команд: %d   ·   средняя сумма %.1f   ·   %s   ·   вариант %d",
		len(u.teams), avg, verdict, seed))

	cards := make([]fyne.CanvasObject, len(u.teams))
	for i, t := range u.teams {
		cards[i] = teamCard(i+1, t, hi)
	}
	u.results.Content = container.NewGridWithColumns(min(len(u.teams), 3), cards...)
	u.results.Refresh()
	u.results.ScrollToTop()
	u.copyBtn.Enable()
}

func teamCard(num int, t balance.Team, maxTotal int) fyne.CanvasObject {
	bar := widget.NewProgressBar()
	bar.Max = float64(max(maxTotal, 1))
	bar.SetValue(float64(t.Total))
	total := t.Total
	bar.TextFormatter = func() string { return fmt.Sprintf("%d баллов", total) }

	rows := []fyne.CanvasObject{bar, widget.NewSeparator()}
	for _, m := range t.Members {
		name := widget.NewLabel(displayName(m))
		name.Truncation = fyne.TextTruncateEllipsis
		pts := widget.NewLabel(strconv.Itoa(m.Points))
		pts.TextStyle.Bold = true
		pts.Importance = pointsImportance(m.Points)
		rows = append(rows, container.NewBorder(nil, nil, nil, pts, name))
	}

	subtitle := fmt.Sprintf("%d %s", len(t.Members), plural(len(t.Members), "человек", "человека", "человек"))
	return widget.NewCard(fmt.Sprintf("Команда %d", num), subtitle, container.NewVBox(rows...))
}

func (u *ui) copyResult() {
	if u.teams == nil {
		return
	}
	var b strings.Builder
	for i, t := range u.teams {
		fmt.Fprintf(&b, "Команда %d — %d чел., сумма баллов: %d\n", i+1, len(t.Members), t.Total)
		for _, m := range t.Members {
			fmt.Fprintf(&b, "  #%d %s — %d\n", m.ID, displayName(m), m.Points)
		}
		b.WriteString("\n")
	}
	b.WriteString(u.summary.Text)
	u.app.Clipboard().SetContent(b.String())
	u.app.SendNotification(fyne.NewNotification("Скопировано", "Состав команд скопирован в буфер обмена"))
}

// ---------- Вспомогательное ----------

func (u *ui) showError(err error) {
	msg := err.Error()
	if r := []rune(msg); len(r) > 0 {
		msg = strings.ToUpper(string(r[0])) + string(r[1:])
	}
	text := widget.NewLabel(msg)
	text.Wrapping = fyne.TextWrapWord
	content := container.NewVScroll(text)
	content.SetMinSize(fyne.NewSize(520, float32(min(80+30*strings.Count(msg, "\n"), 360))))
	dialog.ShowCustom("Ошибка", "Закрыть", content, u.win)
}

func placeholder(text string) fyne.CanvasObject {
	l := widget.NewLabel(text)
	l.Importance = widget.LowImportance
	l.Alignment = fyne.TextAlignCenter
	return container.NewCenter(l)
}

func displayName(p balance.Programmer) string {
	if p.Name == "" {
		return "(без имени)"
	}
	return p.Name
}

// pointsImportance подсвечивает баллы цветом по уровню из README.
func pointsImportance(p int) widget.Importance {
	switch {
	case p >= 75:
		return widget.SuccessImportance
	case p >= 50:
		return widget.HighImportance
	case p >= 25:
		return widget.MediumImportance
	default:
		return widget.LowImportance
	}
}

func plural(n int, one, few, many string) string {
	n %= 100
	if n >= 11 && n <= 14 {
		return many
	}
	switch n % 10 {
	case 1:
		return one
	case 2, 3, 4:
		return few
	}
	return many
}

package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"awesomeProject/internal/cli"
	"awesomeProject/internal/model"
	"awesomeProject/internal/service"
	"awesomeProject/internal/storage"
)

func newCLI(t *testing.T, input string) (out, errBuf bytes.Buffer) {
	t.Helper()
	now := time.Date(2026, 10, 14, 0, 0, 0, 0, time.Local)
	svc := service.NewWithClock(storage.NewMemoryStore(), func() time.Time { return now })
	app := cli.New(svc, strings.NewReader(input), &out, &errBuf)
	if err := app.Run(); err != nil {
		t.Fatal(err)
	}
	return out, errBuf
}

func TestCLIHelpAndExit(t *testing.T) {
	out, errBuf := newCLI(t, "help\nexit\n")
	if errBuf.Len() != 0 {
		t.Fatalf("stderr: %s", errBuf.String())
	}
	got := out.String()
	for _, cmd := range []string{"help", "add", "list", "get", "status", "edit", "delete", "import", "exit"} {
		if !strings.Contains(got, cmd) {
			t.Fatalf("help missing %s: %s", cmd, got)
		}
	}
}

func TestCLIUnknownCommand(t *testing.T) {
	_, errBuf := newCLI(t, "foo\nexit\n")
	if !strings.Contains(errBuf.String(), "неизвестная команда") {
		t.Fatalf("%s", errBuf.String())
	}
}

func TestCLIFlow(t *testing.T) {
	now := time.Date(2026, 10, 14, 0, 0, 0, 0, time.Local)
	svc := service.NewWithClock(storage.NewMemoryStore(), func() time.Time { return now })

	input := strings.Join([]string{
		`add 2026-12-01 Купить молоко`,
		`add 2026-11-01 "Сдать ДЗ"`,
		`list`,
		`get 1`,
		`status 1 in_progress`,
		`edit 2 --title "Сдать ДЗ вовремя"`,
		`list --query дз`,
		`delete 1`,
		`get 1`,
		`exit`,
	}, "\n") + "\n"

	var out, errBuf bytes.Buffer
	app := cli.New(svc, strings.NewReader(input), &out, &errBuf)
	if err := app.Run(); err != nil {
		t.Fatal(err)
	}
	got := out.String() + errBuf.String()
	if !strings.Contains(got, "добавлена задача 1") {
		t.Fatalf("missing create: %s", got)
	}
	if !strings.Contains(got, "Сдать ДЗ") {
		t.Fatalf("missing second task: %s", got)
	}
	if !strings.Contains(got, "всего:") || !strings.Contains(got, "страница:") {
		t.Fatalf("list must show totals: %s", got)
	}
	if !strings.Contains(got, "задача не найдена") {
		t.Fatalf("missing not found: %s", got)
	}
}

func TestCLISpecErrors(t *testing.T) {
	input := strings.Join([]string{
		`add 2026-12-01`,
		`add 2020-01-01 просроченный дедлайн`,
		`add 2026-12-01 "  "`,
		`get 99`,
		`add 2026-12-01 нормальная`,
		`status 1 planned`,
		`status 1 done`,
		`status 1 in_progress`,
		`edit 1 --title новое`,
		`delete 99`,
		`exit`,
	}, "\n") + "\n"

	out, errBuf := newCLI(t, input)
	errs := errBuf.String()
	if !strings.Contains(errs, "использование: add") {
		t.Fatalf("add usage: %s", errs)
	}
	if !strings.Contains(errs, "дедлайн не может быть в прошлом") {
		t.Fatalf("past deadline: %s", errs)
	}
	if !strings.Contains(errs, "заголовок не может быть пустым") {
		t.Fatalf("empty title: %s", errs)
	}
	if !strings.Contains(errs, "задача не найдена") {
		t.Fatalf("not found: %s", errs)
	}
	if !strings.Contains(errs, "уже находится в этом статусе") {
		t.Fatalf("same status: %s", errs)
	}
	if !strings.Contains(errs, "done или canceled") {
		t.Fatalf("closed status: %s / out=%s", errs, out.String())
	}
}

func TestCLIListFiltersAndPagination(t *testing.T) {
	now := time.Date(2026, 10, 20, 0, 0, 0, 0, time.Local)
	store := storage.NewMemoryStore()
	early := service.NewWithClock(store, func() time.Time {
		return time.Date(2026, 10, 10, 0, 0, 0, 0, time.Local)
	})
	_, _ = early.Add("старая открытая", time.Date(2026, 10, 10, 0, 0, 0, 0, time.Local))
	done, _ := early.Add("старая закрытая", time.Date(2026, 10, 10, 0, 0, 0, 0, time.Local))
	if _, err := early.SetStatus(done.ID, model.StatusDone); err != nil {
		t.Fatal(err)
	}
	_, _ = early.Add("свежая", time.Date(2026, 12, 1, 0, 0, 0, 0, time.Local))

	svc := service.NewWithClock(store, func() time.Time { return now })
	input := strings.Join([]string{
		`list --overdue`,
		`list --status planned --limit 1 --page 1`,
		`list --query СВЕЖ`,
		`exit`,
	}, "\n") + "\n"

	var out, errBuf bytes.Buffer
	app := cli.New(svc, strings.NewReader(input), &out, &errBuf)
	if err := app.Run(); err != nil {
		t.Fatal(err)
	}
	if errBuf.Len() != 0 {
		t.Fatalf("stderr: %s", errBuf.String())
	}
	got := out.String()
	if !strings.Contains(got, "старая открытая") {
		t.Fatalf("overdue open missing: %s", got)
	}
	if strings.Count(got, "старая закрытая") != 0 {
		t.Fatalf("closed should not be overdue: %s", got)
	}
	if !strings.Contains(got, "свежая") {
		t.Fatalf("query: %s", got)
	}
	if !strings.Contains(got, "limit: 1") {
		t.Fatalf("pagination: %s", got)
	}
}

func TestCLIImport(t *testing.T) {
	now := time.Date(2026, 10, 14, 0, 0, 0, 0, time.Local)
	svc := service.NewWithClock(storage.NewMemoryStore(), func() time.Time { return now })

	path := filepath.Join(t.TempDir(), "import.json")
	content := `[
  {"title": "хорошая", "deadline": "2026-12-01"},
  {"title": "", "deadline": "2026-12-01"},
  {"title": "плохая дата", "deadline": "не дата"},
  {"title": "вторая хорошая", "deadline": "2026-12-02"}
]`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	input := "import " + path + "\nexit\n"
	var out, errBuf bytes.Buffer
	app := cli.New(svc, strings.NewReader(input), &out, &errBuf)
	if err := app.Run(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "добавлено 2, ошибок 2") {
		t.Fatalf("out=%s err=%s", out.String(), errBuf.String())
	}
	if !strings.Contains(errBuf.String(), "строка 2") || !strings.Contains(errBuf.String(), "строка 3") {
		t.Fatalf("errors: %s", errBuf.String())
	}
}

func TestCLIImportBadJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, errBuf := newCLI(t, "import "+path+"\nexit\n")
	if !strings.Contains(errBuf.String(), "разбор JSON импорта") {
		t.Fatalf("%s", errBuf.String())
	}
}

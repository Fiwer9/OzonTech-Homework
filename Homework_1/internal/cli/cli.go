package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"awesomeProject/internal/model"
	"awesomeProject/internal/service"
)

const dateLayout = "2006-01-02"

const helpText = `help              что умеет программа
add <дата> <текст>   новая задача, дата вида YYYY-MM-DD
list              все задачи, сначала у кого дедлайн раньше
                  --status planned|in_progress|canceled|done
                  --overdue только просроченные
                  --query текст поиск по названию
                  --page N --limit N страницы, по умолчанию 1 и 20
get <id>          одна задача
status <id> <статус>  planned, in_progress, canceled, done
edit <id>         --title "новое имя" и/или --deadline 2026-12-01
delete <id>       удалить задачу
import <файл>     взять задачи из json
exit              закрыть программу

Название с пробелами пиши в кавычках: add 2026-12-01 "сходить в магазин"
`

type CLI struct {
	svc *service.Service
	in  io.Reader
	out io.Writer
	err io.Writer
}

func New(svc *service.Service, in io.Reader, out, err io.Writer) *CLI {
	return &CLI{svc: svc, in: in, out: out, err: err}
}

func (c *CLI) Run() error {
	scanner := bufio.NewScanner(c.in)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	fmt.Fprint(c.out, "> ")
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			fmt.Fprint(c.out, "> ")
			continue
		}
		args := splitCommand(line)
		if len(args) == 0 {
			fmt.Fprint(c.out, "> ")
			continue
		}
		exit, err := c.dispatch(args)
		if err != nil {
			fmt.Fprintf(c.err, "ошибка: %v\n", err)
		}
		if exit {
			return nil
		}
		fmt.Fprint(c.out, "> ")
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("чтение ввода: %w", err)
	}
	return nil
}

func (c *CLI) dispatch(args []string) (bool, error) {
	switch strings.ToLower(args[0]) {
	case "help":
		fmt.Fprint(c.out, helpText)
		return false, nil
	case "exit":
		return true, nil
	case "add":
		return false, c.cmdAdd(args[1:])
	case "list":
		return false, c.cmdList(args[1:])
	case "get":
		return false, c.cmdGet(args[1:])
	case "status":
		return false, c.cmdStatus(args[1:])
	case "edit":
		return false, c.cmdEdit(args[1:])
	case "delete":
		return false, c.cmdDelete(args[1:])
	case "import":
		return false, c.cmdImport(args[1:])
	default:
		return false, fmt.Errorf("неизвестная команда %q, введите help", args[0])
	}
}

func (c *CLI) cmdAdd(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("использование: add <дедлайн YYYY-MM-DD> <заголовок>")
	}
	deadline, err := parseDate(args[0])
	if err != nil {
		return err
	}
	title := strings.Join(args[1:], " ")
	task, err := c.svc.Add(title, deadline)
	if err != nil {
		return err
	}
	fmt.Fprintf(c.out, "добавлена задача %d\n", task.ID)
	c.printTask(task)
	return nil
}

func (c *CLI) cmdList(args []string) error {
	filter := service.ListFilter{}
	if len(args) > 0 {
		f, err := parseFlags(args)
		if err != nil {
			return fmt.Errorf("list: %w", err)
		}
		if raw, ok := f.values["status"]; ok {
			st, err := model.ParseStatus(raw)
			if err != nil {
				return err
			}
			filter.Status = &st
		}
		filter.Overdue = f.bools["overdue"]
		filter.Query = f.values["query"]
		if raw, ok := f.values["page"]; ok {
			n, err := strconv.Atoi(raw)
			if err != nil {
				return fmt.Errorf("некорректный page: %w", err)
			}
			filter.Page = n
		}
		if raw, ok := f.values["limit"]; ok {
			n, err := strconv.Atoi(raw)
			if err != nil {
				return fmt.Errorf("некорректный limit: %w", err)
			}
			filter.Limit = n
		}
	}

	result, err := c.svc.List(filter)
	if err != nil {
		return err
	}
	if len(result.Tasks) == 0 {
		fmt.Fprintln(c.out, "пока пусто")
	} else {
		w := tabwriter.NewWriter(c.out, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tстатус\tдедлайн\tназвание")
		for _, t := range result.Tasks {
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", t.ID, t.Status, t.Deadline.Format(dateLayout), t.Title)
		}
		if err := w.Flush(); err != nil {
			return fmt.Errorf("вывод списка: %w", err)
		}
	}
	fmt.Fprintf(c.out, "всего: %d  страница: %d/%d  limit: %d\n", result.Total, result.Page, result.Pages, result.Limit)
	return nil
}

func (c *CLI) cmdGet(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("использование: get <id>")
	}
	id, err := model.ParseID(args[0])
	if err != nil {
		return err
	}
	task, err := c.svc.Get(id)
	if err != nil {
		return err
	}
	c.printTask(task)
	return nil
}

func (c *CLI) cmdStatus(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("использование: status <id> <status>")
	}
	id, err := model.ParseID(args[0])
	if err != nil {
		return err
	}
	st, err := model.ParseStatus(args[1])
	if err != nil {
		return err
	}
	task, err := c.svc.SetStatus(id, st)
	if err != nil {
		return err
	}
	fmt.Fprintf(c.out, "у задачи %d теперь статус %s\n", task.ID, task.Status)
	return nil
}

func (c *CLI) cmdEdit(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("использование: edit <id> [--title TEXT] [--deadline YYYY-MM-DD]")
	}
	id, err := model.ParseID(args[0])
	if err != nil {
		return err
	}
	f, err := parseFlags(args[1:])
	if err != nil {
		return fmt.Errorf("edit: %w", err)
	}
	var title *string
	var deadline *time.Time
	if raw, ok := f.values["title"]; ok {
		title = &raw
	}
	if raw, ok := f.values["deadline"]; ok {
		d, err := parseDate(raw)
		if err != nil {
			return err
		}
		deadline = &d
	}
	task, err := c.svc.Edit(id, title, deadline)
	if err != nil {
		return err
	}
	fmt.Fprintf(c.out, "задача %d изменена\n", task.ID)
	c.printTask(task)
	return nil
}

func (c *CLI) cmdDelete(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("использование: delete <id>")
	}
	id, err := model.ParseID(args[0])
	if err != nil {
		return err
	}
	if err := c.svc.Delete(id); err != nil {
		return err
	}
	fmt.Fprintf(c.out, "задача %d удалена\n", id)
	return nil
}

type importItem struct {
	Title    string `json:"title"`
	Deadline string `json:"deadline"`
}

func (c *CLI) cmdImport(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("использование: import <путь>")
	}
	path := strings.Join(args, " ")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("чтение файла импорта: %w", err)
	}
	var items []importItem
	if err := json.Unmarshal(data, &items); err != nil {
		return fmt.Errorf("разбор JSON импорта: %w", err)
	}

	added := 0
	failed := 0
	for i, item := range items {
		idx := i + 1
		deadline, err := parseDate(item.Deadline)
		if err != nil {
			fmt.Fprintf(c.err, "строка %d: плохая дата: %v\n", idx, err)
			failed++
			continue
		}
		task, err := c.svc.Add(item.Title, deadline)
		if err != nil {
			fmt.Fprintf(c.err, "строка %d: %v\n", idx, err)
			failed++
			continue
		}
		fmt.Fprintf(c.out, "из файла: задача %d (%s)\n", task.ID, task.Title)
		added++
	}
	fmt.Fprintf(c.out, "импорт: добавлено %d, ошибок %d\n", added, failed)
	return nil
}

func (c *CLI) printTask(t model.Task) {
	fmt.Fprintf(c.out, "id: %d\n", t.ID)
	fmt.Fprintf(c.out, "название: %s\n", t.Title)
	fmt.Fprintf(c.out, "статус: %s\n", t.Status)
	fmt.Fprintf(c.out, "дедлайн: %s\n", t.Deadline.Format(dateLayout))
}

func parseDate(s string) (time.Time, error) {
	t, err := time.ParseInLocation(dateLayout, s, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("дата должна быть в формате YYYY-MM-DD: %w", err)
	}
	return t, nil
}

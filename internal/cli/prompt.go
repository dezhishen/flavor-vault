package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// prompt 读取一行输入，带默认值
func prompt(reader *bufio.Reader, label, def string) (string, error) {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return def, nil
	}
	return line, nil
}

// promptBool 读取 yes/no
func promptBool(reader *bufio.Reader, label string, def bool) (bool, error) {
	hint := "y/N"
	if def {
		hint = "Y/n"
	}
	line, err := prompt(reader, label+" ("+hint+")", "")
	if err != nil {
		return def, err
	}
	lower := strings.ToLower(line)
	if lower == "" {
		return def, nil
	}
	return lower == "y" || lower == "yes" || lower == "是", nil
}

// promptCSV 读取逗号分隔的多个值
func promptCSV(reader *bufio.Reader, label, hint string) ([]string, error) {
	line, err := prompt(reader, label+"（逗号分隔）"+hint, "")
	if err != nil {
		return nil, err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return []string{}, nil
	}
	parts := strings.Split(line, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out, nil
}

// promptInt 读取整数
func promptInt(reader *bufio.Reader, label string, def int) (int, error) {
	line, err := prompt(reader, label, fmt.Sprintf("%d", def))
	if err != nil {
		return def, err
	}
	if line == "" {
		return def, nil
	}
	var n int
	if _, err := fmt.Sscanf(line, "%d", &n); err != nil {
		return def, fmt.Errorf("请输入有效数字")
	}
	return n, nil
}

// newLineReader 创建从标准输入读取的 reader
func newLineReader() *bufio.Reader {
	return bufio.NewReader(os.Stdin)
}
